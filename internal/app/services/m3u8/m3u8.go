package m3u8

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/models"
	"stream-recorder/internal/app/services/streamlink"
	"stream-recorder/internal/app/services/utils"
	"stream-recorder/pkg/ffmpeg"
	"stream-recorder/pkg/logger"
	"strings"
	"time"
)

type M3u8 struct {
	log          *logger.Logger
	concatFFmpeg *ffmpeg.FFmpeg
	c            *config.Config
	sl           *streamlink.Streamlink
	u            *utils.Utils

	HTTPClient  *http.Client
	currentDate string

	sm        *models.StreamMetadata
	isNeedCut bool
	isCancel  bool

	oldSegments              []string
	downloadedSegments       *OrderedSet
	rottenDownloadedSegments *OrderedSet
	txtFileSegments          *OrderedSet
}

func New(log *logger.Logger, platform, username string, splitSegments bool, timeSegment int, c *config.Config, u *utils.Utils) (*M3u8, error) {
	concatFFmpeg, err := ffmpeg.NewFfmpeg(c.FFmpegPATH)
	if err != nil {
		return nil, err
	}

	skipTargetDuration := false
	totalDurationStream := time.Duration(0)
	startDurationStream := time.Duration(0)
	waitingTime := time.Duration(1)

	return &M3u8{
		log:          log,
		concatFFmpeg: concatFFmpeg,
		c:            c,
		sl:           streamlink.New(log, "twitch"),
		u:            u,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
			},
			Timeout: 60 * time.Second,
		},
		currentDate: time.Now().Format("2006-01-02"),
		sm: &models.StreamMetadata{
			SkipTargetDuration:  &skipTargetDuration,
			TotalDurationStream: &totalDurationStream,
			StartDurationStream: &startDurationStream,
			WaitingTime:         &waitingTime,
			Username:            username,
			Platform:            platform,
			SplitSegments:       splitSegments,
			TimeSegment:         timeSegment,
		},
		isNeedCut:                false,
		isCancel:                 false,
		downloadedSegments:       NewOrderedSet(),
		rottenDownloadedSegments: NewOrderedSet(),
		txtFileSegments:          NewOrderedSet(),
	}, nil
}

func (m *M3u8) Run(playlistURL string) error {
	m.log.Debug(fmt.Sprintf("[%s/%s] Starting playlist monitoring", m.sm.Username, m.sm.Platform), slog.String("playlistURL", playlistURL))
	if playlistURL == "" {
		return errors.New("playlistURL is empty")
	}

	streamDir := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.currentDate)
	tempPath := filepath.Join(m.c.TempPATH, streamDir)
	mediaPath := filepath.Join(m.c.MediaPATH, streamDir)
	if err := m.u.CreateDirectoryIfNotExist(tempPath); err != nil {
		return err
	}
	if err := m.u.CreateDirectoryIfNotExist(mediaPath); err != nil {
		return err
	}

	var pathTxt, oldPathTxt, pathFileWithoutExt string
	var isFirst = true
	var fileTxt *os.File

	for {
		newSegments, err := m.fetchPlaylist(playlistURL)
		if err != nil {
			if !strings.Contains(err.Error(), "404") {
				m.log.Error(fmt.Sprintf("[%s/%s] Error fetching playlist", m.sm.Username, m.sm.Platform), err, slog.String("playlistURL", playlistURL))
				time.Sleep(*m.sm.WaitingTime)
				continue
			}

			m.log.Info(fmt.Sprintf("[%s/%s] The streamer has finished the live broadcast, and I'm starting the final processing...", m.sm.Username, m.sm.Platform))
			m.isCancel = true
		}

		segments := newSegments
		if len(m.oldSegments) > 0 {
			segments = append(m.oldSegments, newSegments...)
			m.oldSegments = m.oldSegments[:0]
		}

		if isFirst {
			pathTxt, pathFileWithoutExt = m.generateFilePaths()
			isFirst = false
		}

		if fileTxt == nil || pathTxt != oldPathTxt {
			fileTxt, err = os.OpenFile(pathTxt, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
			if err != nil {
				m.log.Error(fmt.Sprintf("[%s/%s] Error opening file", m.sm.Username, m.sm.Platform), err)
				return err
			}
			oldPathTxt = pathTxt
		}

		m.processSegments(segments, tempPath)
		for _, segment := range m.downloadedSegments.Get() {
			if segment == "" {
				fileTxt.Close()
				m.checkAndSplitSegments(pathTxt, pathFileWithoutExt, &isFirst)

				allSegments := m.downloadedSegments.Get()
				var lastEmptyIndex int
				for i := len(allSegments) - 1; i >= 0; i-- {
					if allSegments[i] == "" {
						lastEmptyIndex = i
						break
					}
				}

				for _, s := range allSegments[lastEmptyIndex+1:] {
					if s != "" {
						m.oldSegments = append(m.oldSegments, s)
					}
				}
				break
			}

			if m.txtFileSegments.Has(segment) {
				continue
			}

			if _, err := fileTxt.WriteString(fmt.Sprintf("file '%s.%s'\n", segment, m.c.FileFormat)); err != nil {
				m.log.Error(fmt.Sprintf("[%s/%s] Error writing to segment file", m.sm.Username, m.sm.Platform), err, slog.String("pathTxt", pathTxt), slog.String("segment", segment))
				return err
			}
			m.txtFileSegments.Add(segment)

		}
		m.downloadedSegments.Clear()

		isSplit := m.sm.SplitSegments && *m.sm.TotalDurationStream-*m.sm.StartDurationStream > time.Duration(m.sm.TimeSegment)*time.Second
		if isSplit || m.isNeedCut || m.isCancel {
			fileTxt.Close()
			m.checkAndSplitSegments(pathTxt, pathFileWithoutExt, &isFirst)
			if m.isCancel {
				break
			}
		}

		if m.rottenDownloadedSegments.Len() > 50 {
			m.rottenDownloadedSegments.TrimToLast(50)
		}
		time.Sleep(*m.sm.WaitingTime)
	}

	return nil
}

func (m *M3u8) checkAndSplitSegments(pathTxt, pathFileWithoutExt string, isFirst *bool) {
	go func(pathTxt, pathFileWithoutExt string) {
		m.concatAndCleanup(pathTxt, pathFileWithoutExt)
	}(pathTxt, pathFileWithoutExt)

	*isFirst = true
	m.isNeedCut = false
}

func (m *M3u8) concatAndCleanup(pathTxt, pathFileWithoutExt string) {
	err := m.concatFFmpeg.Yes().
		ErrDetect("ignore_err").
		LogLevel("warning").
		Format("concat").
		Safe(0).
		Async(1).
		FpsMode("cfr").
		VideoCodec("copy").
		AudioCodec("copy").
		Execute(pathTxt, fmt.Sprintf("%s_download.%s", pathFileWithoutExt, m.c.FileFormat))
	if err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
	}

	err = os.Rename(fmt.Sprintf("%s_download.%s", pathFileWithoutExt, m.c.FileFormat), fmt.Sprintf("%s.%s", pathFileWithoutExt, m.c.FileFormat))
	if err != nil {
		m.log.Error("Failed to rename ffmpeg", err)
	}

	txt, err := m.u.ExtractFilenamesFromTxt(pathTxt)
	if err != nil {
		m.log.Error("Error extracting filenames", err, slog.String("path", pathTxt))
		return
	}
	dir, _ := filepath.Split(pathTxt)

	for _, file := range txt {
		os.Remove(filepath.Join(dir, file))
	}

	os.Remove(pathTxt)
	m.log.Info("Segment is recorded")
}
