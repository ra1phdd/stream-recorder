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
	"strconv"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/services/ffmpeg"
	"stream-recorder/internal/app/services/models"
	"stream-recorder/internal/app/services/runner"
	"stream-recorder/internal/app/services/streamlink"
	"stream-recorder/pkg/logger"
	"strings"
	"sync"
	"time"
)

type M3u8 struct {
	mu          sync.Mutex
	HTTPClient  *http.Client
	sem         chan struct{}
	currentDate string
	f           *ffmpeg.Ffmpeg
	c           *config.Config
	st          *streamlink.TwitchAPI

	sm        *models.StreamMetadata
	isNeedCut bool
	isCancel  bool

	rottenDownloadedSegments []string
	downloadedSegments       []string
}

func New(platform, username string, splitSegments bool, timeSegment int, rp *runner.Process, c *config.Config, sem chan struct{}) *M3u8 {
	return &M3u8{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
			},
			Timeout: 60 * time.Second,
		},
		sem:         sem,
		currentDate: time.Now().Format("2006-01-02"),
		f:           ffmpeg.New(rp, c),
		c:           c,
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
		return nil, false, errors.New(strconv.Itoa(resp.StatusCode))
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
		fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.currentDate),
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

	streamDir := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.currentDate)
	if err := CreateDirectoryIfNotExist(filepath.Join(m.c.TempPATH, streamDir)); err != nil {
		return err
	}
	if err := CreateDirectoryIfNotExist(filepath.Join(m.c.MediaPATH, streamDir)); err != nil {
		return err
	}

	var fileSegments, oldFileSegments, filePathWithoutExtension string
	var isFirst = true
	var file *os.File
	for {
		segments, _, err := m.fetchSegments(playlistURL)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				logger.Infof("The streamer has finished the live broadcast, and I'm starting the final processing...", m.sm.Username, m.sm.Platform)
				m.isCancel = true
			} else {
				logger.Errorf("Error fetching playlist", m.sm.Username, m.sm.Platform, zap.String("playlistURL", playlistURL), zap.Error(err))
				time.Sleep(m.sm.WaitingTime)
				continue
			}
		} else {
			m.processSegments(segments)
		}

		if isFirst {
			fileSegments, filePathWithoutExtension = m.generateFilePaths()
			isFirst = false
		}

		if file == nil || fileSegments != oldFileSegments {
			file, err = os.OpenFile(fileSegments, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
			if err != nil {
				logger.Error("Error opening file", zap.Error(err))
				return err
			}
			oldFileSegments = fileSegments
		}

		m.mu.Lock()
		for _, segment := range m.downloadedSegments {
			if _, err := file.WriteString(fmt.Sprintf("file '%s'\n", segment)); err != nil {
				logger.Errorf("Error writing to segment file", m.sm.Username, m.sm.Platform, zap.String("fileSegments", fileSegments), zap.String("segment", segment), zap.Error(err))
				return err
			}
		}
		m.mu.Unlock()

		if (m.sm.SplitSegments && m.sm.TotalDurationStream-m.sm.StartDurationStream > time.Duration(m.sm.TimeSegment)*time.Second) || m.isNeedCut || m.isCancel {
			file.Close()

			if err := m.f.Run(fileSegments, filePathWithoutExtension); err != nil {
				logger.Errorf("Error running external process", m.sm.Username, m.sm.Platform, zap.String("fileSegments", fileSegments), zap.String("filepath", filePathWithoutExtension), zap.Error(err))
			}

			m.sm.StartDurationStream = 0
			m.isNeedCut = false
			isFirst = true

			if m.isCancel {
				break
			}
		}

		m.rottenDownloadedSegments = append(m.rottenDownloadedSegments, m.downloadedSegments...)
		if len(m.rottenDownloadedSegments) > 50 {
			m.rottenDownloadedSegments = m.rottenDownloadedSegments[len(m.rottenDownloadedSegments)-50:]
		}
		m.downloadedSegments = m.downloadedSegments[:0]
		time.Sleep(m.sm.WaitingTime)
	}

	return nil
}

func CreateDirectoryIfNotExist(path string) error {
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

func (m *M3u8) generateFilePaths() (fileSegments, filePathWithoutExtension string) {
	dirName := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.currentDate)
	fileName := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.formatDuration(m.sm.StartDurationStream))

	fileSegments = filepath.Join(m.c.TempPATH, dirName, fileName+".txt")
	filePathWithoutExtension = filepath.Join(m.c.MediaPATH, dirName, fileName)

	return fileSegments, filePathWithoutExtension
}

func (m *M3u8) fetchSegments(playlistURL string) ([]string, bool, error) {
	switch m.sm.Platform {
	case "twitch":
		return m.FetchPlaylist(playlistURL, m.st.ParseM3u8)
	default:
		return nil, false, fmt.Errorf("unsupported platform: %s", m.sm.Platform)
	}
}

func (m *M3u8) processSegments(segments []string) {
	var wg sync.WaitGroup

	for _, segment := range segments {
		url := m.GetShortFileName(segment)
		if !m.contains(m.rottenDownloadedSegments, url) {
			wg.Add(1)
			m.sem <- struct{}{}

			m.mu.Lock()
			m.downloadedSegments = append(m.downloadedSegments, url)
			m.mu.Unlock()

			go func(segment, url string) {
				defer wg.Done()
				defer func() { <-m.sem }()

				if err := m.DownloadSegment(segment); err != nil {
					logger.Errorf("Error downloading segment", m.sm.Username, m.sm.Platform, zap.String("segmentURL", segment), zap.Error(err))

					m.mu.Lock()
					for i, v := range m.downloadedSegments {
						if v == url {
							m.downloadedSegments = append(m.downloadedSegments[:i], m.downloadedSegments[i+1:]...)
							break
						}
					}
					m.mu.Unlock()
				}
			}(segment, url)
		}
	}
	wg.Wait()
}

func (m *M3u8) contains(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

func (m *M3u8) formatDuration(d time.Duration) string {
	hours := d / time.Hour
	d -= hours * time.Hour
	mins := d / time.Minute
	d -= mins * time.Minute
	secs := d / time.Second
	return fmt.Sprintf("%dh%dm%ds", hours, mins, secs)
}

func (m *M3u8) ChangeIsNeedCut(value bool) {
	m.isNeedCut = value
}

func (m *M3u8) ChangeIsCancel(value bool) {
	m.isCancel = value
}
