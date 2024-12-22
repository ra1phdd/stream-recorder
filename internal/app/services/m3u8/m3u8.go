package m3u8

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/services/ffmpeg"
	"stream-recorder/internal/app/services/models"
	"stream-recorder/internal/app/services/runner"
	"stream-recorder/internal/app/services/streamlink"
	"stream-recorder/pkg/logger"
	"time"
)

type M3u8 struct {
	HTTPClient *http.Client
	f          *ffmpeg.Ffmpeg
	c          *config.Config
	st         *streamlink.TwitchAPI

	sm        *models.StreamMetadata
	isNeedCut bool
	isCancel  bool

	rottenDownloadedSegments []string
	downloadedSegments       []string
}

func New(platform, username string, splitSegments bool, timeSegment int, rp *runner.Process, c *config.Config) *M3u8 {
	return &M3u8{
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		f:          ffmpeg.New(rp, c),
		c:          c,
		sm: &models.StreamMetadata{
			SkipTargetDuration:  false,
			TotalDurationStream: 0,
			StartDurationStream: 0,
			WaitingTime:         1,
			Username:            username,
			Platform:            platform,
			SplitSegments:       splitSegments,
			TimeSegment:         timeSegment,
		},
		isNeedCut:                false,
		rottenDownloadedSegments: make([]string, 0),
		downloadedSegments:       make([]string, 0),
		isCancel:                 false,
	}
}

func (m *M3u8) FetchPlaylist(url string, parseM3u8 func(string, *models.StreamMetadata) (int, bool, string)) ([]string, bool, error) {
	resp, err := m.HTTPClient.Get(url)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("Failed to fetch master playlist", m.sm.Username, m.sm.Platform, zap.Int("status_code", resp.StatusCode))
		return nil, false, errors.New("failed to fetch master playlist")
	}

	var segments []string
	scanner := bufio.NewScanner(resp.Body)
	skipCount := 0
	isSkipSegments := false

	for scanner.Scan() {
		line := scanner.Text()

		if skipCount > 0 {
			isSkipSegments = true
			skipCount--
			continue
		}

		var isSegment bool
		var segmentURL string
		skipCount, isSegment, segmentURL = parseM3u8(line, m.sm)
		if isSegment {
			segments = append(segments, segmentURL)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Errorf("Buffer scanning error", m.sm.Username, m.sm.Platform, zap.Error(err))
		return nil, isSkipSegments, err
	}

	return segments, isSkipSegments, nil
}

func (m *M3u8) DownloadSegment(url string) error {
	logger.Debugf("Starting download segment", m.sm.Username, m.sm.Platform, zap.String("url", url))

	resp, err := m.HTTPClient.Get(url)
	if err != nil {
		logger.Errorf("Failed to download segment", m.sm.Username, m.sm.Platform, zap.String("url", url), zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("Received non-OK status code while downloading segment", m.sm.Username, m.sm.Platform, zap.String("url", url), zap.Int("status_code", resp.StatusCode))
		return errors.New("failed to download segment")
	}

	fileName := m.GetShortFileName(url)
	filePath := filepath.Join(
		m.c.TempPATH,
		fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, time.Now().Format("2006-01-02")),
		fileName,
	)
	logger.Debugf("Creating file for segment", m.sm.Username, m.sm.Platform, zap.String("filePath", filePath))

	file, err := os.Create(filePath)
	if err != nil {
		logger.Errorf("Failed to create file for segment", m.sm.Username, m.sm.Platform, zap.String("filePath", filePath), zap.Error(err))
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		logger.Errorf("Failed to write segment to file", m.sm.Username, m.sm.Platform, zap.String("filePath", filePath), zap.Error(err))
		return err
	}

	logger.Debugf("Successfully downloaded segment", m.sm.Username, m.sm.Platform, zap.String("filePath", filePath))
	return nil
}

func (m *M3u8) GetShortFileName(url string) string {
	hasher := md5.New()
	hasher.Write([]byte(url))
	return hex.EncodeToString(hasher.Sum(nil)) + ".ts"
}

func (m *M3u8) Run(playlistURL string) error {
	logger.Debugf("Starting playlist monitoring", m.sm.Username, m.sm.Platform, zap.String("playlistURL", playlistURL))
	if playlistURL == "" {
		return errors.New("playlistURL is empty")
	}

	streamDir := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, time.Now().Format("2006-01-02"))
	if err := m.createDirectoryIfNotExist(filepath.Join(m.c.TempPATH, streamDir)); err != nil {
		return err
	}
	if err := m.createDirectoryIfNotExist(filepath.Join(m.c.MediaPATH, streamDir)); err != nil {
		return err
	}

	for {
		var segments []string
		var err error
		switch m.sm.Platform {
		case "twitch":
			segments, _, err = m.FetchPlaylist(playlistURL, m.st.ParseM3u8)
		}
		if err != nil {
			logger.Errorf("Error fetching playlist", m.sm.Username, m.sm.Platform, zap.String("playlistURL", playlistURL), zap.Error(err))
			time.Sleep(m.sm.WaitingTime)
			continue
		}

		m.processSegments(segments)

		if (m.sm.SplitSegments && m.sm.TotalDurationStream-m.sm.StartDurationStream > time.Duration(m.sm.TimeSegment)*time.Second) || m.isNeedCut || m.isCancel {
			fileSegments := filepath.Join(
				m.c.TempPATH,
				fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, time.Now().Format("2006-01-02")),
				fmt.Sprintf("%s_%s_%s.txt", m.sm.Platform, m.sm.Username, m.sm.StartDurationStream),
			)
			filePathWithoutExtension := filepath.Join(
				m.c.MediaPATH,
				fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, time.Now().Format("2006-01-02")),
				fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.sm.StartDurationStream),
			)

			if len(m.downloadedSegments) != 0 {
				logger.Infof("Creating segment file", m.sm.Username, m.sm.Platform, zap.String("fileSegments", fileSegments), zap.String("filepath", filePathWithoutExtension))

				if err := m.createSegmentFile(fileSegments); err != nil {
					return err
				}

				if err := m.f.Run(fileSegments, filePathWithoutExtension); err != nil {
					logger.Errorf("Error running external process", m.sm.Username, m.sm.Platform, zap.String("fileSegments", fileSegments), zap.String("filepath", filePathWithoutExtension), zap.Error(err))
				}

				logger.Infof("Completed segment processing", m.sm.Username, m.sm.Platform, zap.String("filepath", filePathWithoutExtension))
			}

			m.sm.StartDurationStream = 0
			m.rottenDownloadedSegments = m.downloadedSegments
			m.downloadedSegments = m.downloadedSegments[:0]
			m.isNeedCut = false

			if m.isCancel {
				break
			}
		}

		time.Sleep(m.sm.WaitingTime)
	}

	return nil
}

func (m *M3u8) createDirectoryIfNotExist(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		logger.Debug("Directory does not exist. Creating...", zap.String("outputDir", path))
		if err := os.Mkdir(path, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		logger.Debug("Directory created successfully", zap.String("outputDir", path))
	}
	return nil
}

func (m *M3u8) processSegments(segments []string) {
	for _, segment := range segments {
		url := m.GetShortFileName(segment)
		if !m.contains(m.downloadedSegments, url) && !m.contains(m.rottenDownloadedSegments, url) {
			if err := m.DownloadSegment(segment); err != nil {
				logger.Errorf("Error downloading segment", m.sm.Username, m.sm.Platform, zap.String("segmentURL", segment), zap.Error(err))
			} else {
				m.downloadedSegments = append(m.downloadedSegments, url)
			}
		}
	}
}

func (m *M3u8) createSegmentFile(fileSegments string) error {
	file, err := os.Create(fileSegments)
	if err != nil {
		logger.Errorf("Error creating segment file", m.sm.Username, m.sm.Platform, zap.String("fileSegments", fileSegments), zap.Error(err))
		return err
	}
	defer file.Close()

	for _, segment := range m.downloadedSegments {
		if _, err := file.WriteString(fmt.Sprintf("file '%s'\n", segment)); err != nil {
			logger.Errorf("Error writing to segment file", m.sm.Username, m.sm.Platform, zap.String("fileSegments", fileSegments), zap.Error(err))
			return err
		}
	}
	return nil
}

func (m *M3u8) contains(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

func (m *M3u8) ChangeIsNeedCut(value bool) {
	m.isNeedCut = value
}

func (m *M3u8) ChangeIsCancel(value bool) {
	m.isCancel = value
}
