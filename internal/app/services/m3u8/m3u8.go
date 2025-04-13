package m3u8

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	segmentId           int

	dataSegments       []byte
	downloadedSegments *OrderedSet
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
		isNeedCut:          false,
		isCancel:           false,
		downloadedSegments: NewOrderedSet(),
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
		m.downloadedSegments.TrimToLast(50)

		isSplit := m.sm.SplitSegments && *m.sm.TotalDurationStream-*m.sm.StartDurationStream > time.Duration(m.sm.TimeSegment)*time.Second
		if isSplit || m.GetIsNeedCut() || m.GetIsCancel() || isErrDownload {
			pathTempWithoutExt, pathMediaWithoutExt := m.generateFilePaths(streamDir)

			err := m.FlushTxtToDisk(pathTempWithoutExt)
			if err != nil {
				m.log.Error(fmt.Sprintf("[%s/%s] Error flush txt to disk", m.sm.Username, m.sm.Platform), err)
				return err
			}

			go func(pathTempWithoutExt, pathMediaWithoutExt string) {
				m.ConcatAndCleanup(pathTempWithoutExt, pathMediaWithoutExt)
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

func (m *M3u8) ConcatAndCleanup(pathTempWithoutExt, pathMediaWithoutExt string) {
	runConcat := func(inputTxt, outputFile, vCodec, aCodec string) {
		ff, err := ffmpeg.NewFfmpeg(m.c.FFmpegPATH)
		if err != nil {
			m.log.Error(fmt.Sprintf("[%s/%s] Initialize ffmpeg", m.sm.Username, m.sm.Platform), err)
			return
		}

		err = ff.Yes().
			LogLevel("warning").
			Format("concat").
			VideoCodec(vCodec).
			AudioCodec(aCodec).
			Execute([]string{inputTxt}, outputFile)
		if err != nil {
			m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
			return
		}

		segments, err := m.u.ExtractFilenamesFromTxt(inputTxt)
		if err != nil {
			m.log.Error("Extract segments failed", err)
			return
		}

		dir := filepath.Dir(pathTempWithoutExt)
		for _, file := range segments {
			os.Remove(filepath.Join(dir, file))
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		runConcat(pathTempWithoutExt+"_video.txt", fmt.Sprintf("%s.%s", pathTempWithoutExt, m.c.FileFormat), "copy", "none")
	}()

	go func() {
		defer wg.Done()
		runConcat(pathTempWithoutExt+"_audio.txt", fmt.Sprintf("%s.%s", pathTempWithoutExt, getRecommendedFormat(m.c.AudioCodec)), "none", "copy")
	}()
	wg.Wait()

	ffConcat, err := ffmpeg.NewFfmpeg(m.c.FFmpegPATH)
	if err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Initialize ffmpeg", m.sm.Username, m.sm.Platform), err)
	}

	err = ffConcat.Yes().
		LogLevel("warning").
		VideoCodec("copy").
		AudioCodec("copy").
		Execute([]string{
			fmt.Sprintf("%s.%s", pathTempWithoutExt, m.c.FileFormat),
			fmt.Sprintf("%s.%s", pathTempWithoutExt, getRecommendedFormat(m.c.AudioCodec)),
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
		fmt.Sprintf("%s.%s", pathTempWithoutExt, getRecommendedFormat(m.c.AudioCodec)),
		pathTempWithoutExt + "_video.txt",
		pathTempWithoutExt + "_audio.txt",
	}
	for _, file := range intermediates {
		os.Remove(file)
	}

	m.log.Info("Segment is recorded")
}

func (m *M3u8) FlushTxtToDisk(pathWithoutExtension string) error {
	mediaTypes := []struct {
		fileSuffix string
		segmentExt string
	}{
		{"_video.txt", fmt.Sprintf(".%s", m.c.FileFormat)},
		{"_audio.txt", fmt.Sprintf(".%s", getRecommendedFormat(m.c.AudioCodec))},
	}

	dir := filepath.Dir(pathWithoutExtension)
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.log.Error("Error reading directory", err, slog.String("path", dir), slog.String("username", m.sm.Username), slog.String("platform", m.sm.Platform))
		return err
	}

	for _, mt := range mediaTypes {
		filePath := pathWithoutExtension + mt.fileSuffix
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			m.log.Error("Error creating segment list file", err, slog.String("path", filePath), slog.String("username", m.sm.Username), slog.String("platform", m.sm.Platform))
			return err
		}
		defer f.Close()

		segments := m.filterAndSortSegments(entries, mt.segmentExt)
		if err := m.writeSegmentsList(f, segments); err != nil {
			m.log.Error("Error writing segments list", err, slog.String("path", filePath), slog.String("username", m.sm.Username), slog.String("platform", m.sm.Platform))
			return err
		}
	}

	return nil
}

func (m *M3u8) filterAndSortSegments(entries []os.DirEntry, ext string) []string {
	var segments []string

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ext {
			segments = append(segments, entry.Name())
		}
	}

	sort.Slice(segments, func(i, j int) bool {
		return extractNumber(segments[i]) < extractNumber(segments[j])
	})

	return segments
}

func (m *M3u8) writeSegmentsList(f *os.File, segments []string) error {
	for _, segment := range segments {
		if _, err := fmt.Fprintf(f, "file '%s'\n", segment); err != nil {
			return fmt.Errorf("failed to write segment %q: %w", segment, err)
		}
	}
	return nil
}

func extractNumber(filename string) int {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 1 {
		return -1
	}
	num, _ := strconv.Atoi(parts[0])
	return num
}

func getRecommendedFormat(codec string) string {
	codec = strings.ToLower(strings.TrimSpace(codec))

	switch codec {
	case "mp3", "libmp3lame":
		return "mp3"
	case "aac", "libfdk_aac", "aac_latm":
		return "aac"
	case "flac":
		return "flac"
	case "vorbis", "libvorbis":
		return "ogg"
	case "opus", "libopus":
		return "opus"
	case "pcm_s16le", "pcm_s24le", "pcm_s32le":
		return "wav"
	default:
		return ""
	}
}
