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
	"sync"
	"time"
)

type M3u8 struct {
	log *logger.Logger
	c   *config.Config
	sl  *streamlink.Streamlink
	u   *utils.Utils

	HTTPClient *http.Client
	sm         *models.StreamMetadata

	muCut, muCancel     sync.Mutex
	isNeedCut, isCancel bool

	downloadedSegments       *OrderedSet
	rottenDownloadedSegments *OrderedSet
}

func New(log *logger.Logger, platform, username string, splitSegments bool, timeSegment int, c *config.Config, u *utils.Utils) (*M3u8, error) {
	skipTargetDuration := false
	totalDurationStream := time.Duration(0)
	startDurationStream := time.Duration(0)
	waitingTime := time.Duration(1)

	return &M3u8{
		log: log,
		c:   c,
		sl:  streamlink.New(log, "twitch"),
		u:   u,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
			},
			Timeout: 60 * time.Second,
		},
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
	}, nil
}

func (m *M3u8) Run(playlistURL string) error {
	m.log.Debug(fmt.Sprintf("[%s/%s] Starting playlist monitoring", m.sm.Username, m.sm.Platform), slog.String("playlistURL", playlistURL))
	if playlistURL == "" {
		return errors.New("playlistURL is empty")
	}

	streamDir := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, time.Now().Format("2006-01-02"))
	if err := m.u.CreateDirectoryIfNotExist(filepath.Join(m.c.TempPATH, streamDir)); err != nil {
		return err
	}
	if err := m.u.CreateDirectoryIfNotExist(filepath.Join(m.c.MediaPATH, streamDir)); err != nil {
		return err
	}

	for {
		segments, err := m.fetchPlaylist(playlistURL)
		if err != nil {
			if !strings.Contains(err.Error(), "404") {
				m.log.Error(fmt.Sprintf("[%s/%s] Error fetching playlist", m.sm.Username, m.sm.Platform), err, slog.String("playlistURL", playlistURL))
				time.Sleep(*m.sm.WaitingTime)
				continue
			}

			m.log.Info(fmt.Sprintf("[%s/%s] The streamer has finished the live broadcast, and I'm starting the final processing...", m.sm.Username, m.sm.Platform))
			m.isCancel = true
		}
		isErrDownload := m.processSegments(segments, filepath.Join(m.c.TempPATH, streamDir))

		isSplit := m.sm.SplitSegments && *m.sm.TotalDurationStream-*m.sm.StartDurationStream > time.Duration(m.sm.TimeSegment)*time.Second
		if isSplit || m.GetIsNeedCut() || m.GetIsCancel() || isErrDownload {
			pathTempWithoutExt, pathMediaWithoutExt := m.generateFilePaths(streamDir)

			err := m.flushTxtToDisk(pathTempWithoutExt)
			if err != nil {
				m.log.Error(fmt.Sprintf("[%s/%s] Error flush txt to disk", m.sm.Username, m.sm.Platform), err)
				return err
			}

			go func(pathTempWithoutExt, pathMediaWithoutExt string) {
				m.concatAndCleanup(pathTempWithoutExt, pathMediaWithoutExt)
			}(pathTempWithoutExt, pathMediaWithoutExt)

			m.ChangeIsNeedCut(false)
			if m.GetIsCancel() {
				break
			}
		}
		time.Sleep(*m.sm.WaitingTime)
	}

	return nil
}

func (m *M3u8) concatAndCleanup(pathTempWithoutExt, pathMediaWithoutExt string) {
	runConcat := func(inputTxt, outputFile, vCodec, aCodec string) error {
		ff, err := ffmpeg.NewFfmpeg(m.c.FFmpegPATH)
		if err != nil {
			m.log.Error(fmt.Sprintf("[%s/%s] Initialize ffmpeg", m.sm.Username, m.sm.Platform), err)
		}

		return ff.Yes().
			ErrDetect("ignore_err").
			LogLevel("warning").
			Format("concat").
			VideoCodec(vCodec).
			AudioCodec(aCodec).
			Execute([]string{inputTxt}, outputFile)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()

		err := runConcat(pathTempWithoutExt+"_video.txt", fmt.Sprintf("%s.%s", pathTempWithoutExt, m.c.FileFormat), "copy", "none")
		if err != nil {
			m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
		}
	}()

	go func() {
		defer wg.Done()

		err := runConcat(pathTempWithoutExt+"_audio.txt", fmt.Sprintf("%s.wav", pathTempWithoutExt), "none", "copy")
		if err != nil {
			m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
		}
	}()

	wg.Wait()

	var filesToDelete []string
	for _, suffix := range []string{"_video.txt", "_audio.txt"} {
		segments, err := m.u.ExtractFilenamesFromTxt(pathTempWithoutExt + suffix)
		if err != nil {
			m.log.Error("Extract segments failed", err)
			continue
		}
		filesToDelete = append(filesToDelete, segments...)
	}

	dir := filepath.Dir(pathTempWithoutExt)
	for _, file := range filesToDelete {
		os.Remove(filepath.Join(dir, file))
	}

	ffConcat, err := ffmpeg.NewFfmpeg(m.c.FFmpegPATH)
	if err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Initialize ffmpeg", m.sm.Username, m.sm.Platform), err)
	}

	err = ffConcat.Yes().
		ErrDetect("ignore_err").
		LogLevel("warning").
		VideoCodec("copy").
		AudioCodec("copy").
		Execute([]string{
			fmt.Sprintf("%s.%s", pathTempWithoutExt, m.c.FileFormat),
			fmt.Sprintf("%s.wav", pathTempWithoutExt),
		}, fmt.Sprintf("%s_download.%s", pathMediaWithoutExt, m.c.FileFormat))
	if err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
	}

	err = os.Rename(fmt.Sprintf("%s_download.%s", pathMediaWithoutExt, m.c.FileFormat), fmt.Sprintf("%s.%s", pathMediaWithoutExt, m.c.FileFormat))
	if err != nil {
		m.log.Error("Failed to rename ffmpeg", err)
	}

	intermediates := []string{
		fmt.Sprintf("%s.%s", pathTempWithoutExt, m.c.FileFormat),
		fmt.Sprintf("%s.wav", pathTempWithoutExt),
		pathTempWithoutExt + "_video.txt",
		pathTempWithoutExt + "_audio.txt",
	}
	for _, file := range intermediates {
		os.Remove(file)
	}

	m.log.Info("Segment is recorded")
}

func (m *M3u8) flushTxtToDisk(pathWithoutExtension string) error {
	videoTxt, err := os.OpenFile(pathWithoutExtension+"_video.txt", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Error opening file", m.sm.Username, m.sm.Platform), err)
		return err
	}
	defer videoTxt.Close()

	audioTxt, err := os.OpenFile(pathWithoutExtension+"_audio.txt", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Error opening file", m.sm.Username, m.sm.Platform), err)
		return err
	}
	defer audioTxt.Close()

	for _, segment := range m.downloadedSegments.Get() {
		if _, err := videoTxt.WriteString(fmt.Sprintf("file '%s.ts'\n", segment)); err != nil {
			m.log.Error(fmt.Sprintf("[%s/%s] Error writing to segment file", m.sm.Username, m.sm.Platform), err, slog.String("pathWithoutExtension", pathWithoutExtension), slog.String("segment", segment))
			continue
		}

		if _, err := audioTxt.WriteString(fmt.Sprintf("file '%s.wav'\n", segment)); err != nil {
			m.log.Error(fmt.Sprintf("[%s/%s] Error writing to segment file", m.sm.Username, m.sm.Platform), err, slog.String("pathWithoutExtension", pathWithoutExtension), slog.String("segment", segment))
			continue
		}
	}
	*m.rottenDownloadedSegments = *m.downloadedSegments
	m.rottenDownloadedSegments.TrimToLast(50)
	m.downloadedSegments.Clear()

	return nil
}
