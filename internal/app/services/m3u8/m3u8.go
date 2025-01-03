package m3u8

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
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
	muDs, muFs, muFtxt sync.Mutex
	HTTPClient         *http.Client
	currentDate        string
	f                  *ffmpeg.Ffmpeg
	c                  *config.Config
	st                 *streamlink.TwitchAPI

	sm        *models.StreamMetadata
	isNeedCut bool
	isCancel  bool

	rottenDownloadedSegments []string
	combinedSegments         []string
	txtFileSegments          []string

	currentMemoryUsage int
	buffer             bytes.Buffer

	nameFileCombinedSegment, oldNameFileCombinedSegment string
}

func New(platform, username string, splitSegments bool, timeSegment int, rp *runner.Process, c *config.Config) *M3u8 {
	return &M3u8{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
			},
			Timeout: 60 * time.Second,
		},
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
		combinedSegments:         make([]string, 0),
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

	var pathTxt, oldPathTxt, pathWithoutExtension string
	var isFirst = true
	var fileTxt, fileSegment *os.File
	for {
		isSplit := m.sm.SplitSegments && m.sm.TotalDurationStream-m.sm.StartDurationStream > time.Duration(m.sm.TimeSegment)*time.Second

		// Скачиваем сегменты
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
		}

		if isFirst {
			pathTxt, pathWithoutExtension = m.generateFilePaths()

			hash, err := generateRandomHash()
			if err != nil {
				return err
			}
			m.nameFileCombinedSegment = fmt.Sprintf("%s.ts", hash)

			isFirst = false
		}

		// Создаем файл сегмента, в который будем сбрасывать данные до заполнения буфера
		if fileSegment == nil || m.nameFileCombinedSegment != m.oldNameFileCombinedSegment {
			outputFilePath := filepath.Join(
				m.c.TempPATH,
				fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.currentDate),
				m.nameFileCombinedSegment,
			)

			fileSegment, err = os.Create(outputFilePath)
			if err != nil {
				logger.Fatalf("Failed to create output file", m.sm.Username, m.sm.Platform, zap.String("filePath", outputFilePath), zap.Error(err))
			}

			m.oldNameFileCombinedSegment = m.nameFileCombinedSegment
		}

		// Сбрасываем данные в файл
		m.processSegments(segments, fileSegment, isSplit || m.isNeedCut || m.isCancel)

		if fileTxt == nil || pathTxt != oldPathTxt {
			fileTxt, err = os.OpenFile(pathTxt, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
			if err != nil {
				logger.Error("Error opening file", zap.Error(err))
				return err
			}
			oldPathTxt = pathTxt
		}

		m.muFtxt.Lock()
		for _, segment := range m.combinedSegments {
			if !m.contains(m.txtFileSegments, segment) {
				if _, err := fileTxt.WriteString(fmt.Sprintf("file '%s'\n", segment)); err != nil {
					logger.Errorf("Error writing to segment file", m.sm.Username, m.sm.Platform, zap.String("pathTxt", pathTxt), zap.String("segment", segment), zap.Error(err))
					return err
				}
				m.txtFileSegments = append(m.txtFileSegments, segment)
			}
		}
		m.muFtxt.Unlock()

		if isSplit || m.isNeedCut || m.isCancel {
			fileTxt.Close()

			if err := m.f.Run(pathTxt, pathWithoutExtension); err != nil {
				logger.Errorf("Error running external process", m.sm.Username, m.sm.Platform, zap.String("pathTxt", pathTxt), zap.String("filepath", pathWithoutExtension), zap.Error(err))
			}

			m.sm.StartDurationStream = 0
			m.isNeedCut = false
			isFirst = true

			if m.isCancel {
				break
			}
		}

		if len(m.rottenDownloadedSegments) > 50 {
			m.rottenDownloadedSegments = m.rottenDownloadedSegments[len(m.rottenDownloadedSegments)-50:]
		}
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

func (m *M3u8) processSegments(segments []string, fileSegment *os.File, isNeedFlush bool) {
	dataMap := make(map[int][]byte)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for index, segment := range segments {
		wg.Add(1)
		go func(index int, segment string) {
			defer wg.Done()

			url := m.GetShortFileName(segment)
			if !m.contains(m.rottenDownloadedSegments, url) {
				m.rottenDownloadedSegments = append(m.rottenDownloadedSegments, url)

				data, err := m.DownloadSegment(segment)
				if err != nil {
					logger.Errorf("Error downloading segment", m.sm.Username, m.sm.Platform, zap.String("segmentURL", segment), zap.Error(err))
					return
				}

				mu.Lock()
				dataMap[index] = data
				mu.Unlock()
			}
		}(index, segment)
	}

	wg.Wait()

	for i := 0; i < len(segments); i++ {
		m.buffer.Write(dataMap[i])
		m.currentMemoryUsage += len(dataMap[i])
	}

	if m.currentMemoryUsage >= m.c.BufferSize*1024*1024 || isNeedFlush {
		m.FlushToDisk(fileSegment)
	}
}

func (m *M3u8) FlushToDisk(fileSegment *os.File) {
	_, err := fileSegment.Write(m.buffer.Bytes())
	if err != nil {
		logger.Errorf("Failed to flush buffer to file", m.sm.Username, m.sm.Platform, zap.String("filePath", fileSegment.Name()), zap.Error(err))
	}
	m.buffer.Reset()
	m.currentMemoryUsage = 0

	m.combinedSegments = append(m.combinedSegments, m.nameFileCombinedSegment)
	fileSegment.Close()

	hash, err := generateRandomHash()
	if err != nil {
		logger.Error("Error generating hash", zap.Error(err))
	}
	m.nameFileCombinedSegment = fmt.Sprintf("%s.ts", hash)
}

func (m *M3u8) DownloadSegment(url string) ([]byte, error) {
	maxRetries := 5
	var attempt int

	for {
		attempt++
		logger.Debugf("Starting download segment", m.sm.Username, m.sm.Platform, zap.String("url", url), zap.Int("attempt", attempt))

		resp, err := m.HTTPClient.Get(url)
		if err != nil {
			logger.Errorf("Failed to download segment", m.sm.Username, m.sm.Platform, zap.String("url", url), zap.Int("attempt", attempt), zap.Error(err))
			if attempt >= maxRetries {
				return nil, fmt.Errorf("max retries reached: %w", err)
			}
			time.Sleep(time.Second * time.Duration(attempt))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			logger.Warnf("Segment not found (404) for url", m.sm.Username, m.sm.Platform, zap.String("url", url))
			return nil, errors.New("segment not found")
		}

		if resp.StatusCode != http.StatusOK {
			logger.Errorf("Received non-OK status code while downloading segment", m.sm.Username, m.sm.Platform, zap.String("url", url), zap.Int("status_code", resp.StatusCode), zap.Int("attempt", attempt))
			if attempt >= maxRetries {
				return nil, fmt.Errorf("max retries reached, last status: %d", resp.StatusCode)
			}
			time.Sleep(time.Second * time.Duration(attempt))
			continue
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("Failed to read segment data", m.sm.Username, m.sm.Platform, zap.String("url", url), zap.Int("attempt", attempt), zap.Error(err))
			if attempt >= maxRetries {
				return nil, fmt.Errorf("max retries reached: %w", err)
			}
			time.Sleep(time.Second * time.Duration(attempt))
			continue
		}

		logger.Debugf("Successfully downloaded segment", m.sm.Username, m.sm.Platform, zap.String("url", url), zap.Int("attempt", attempt))
		return data, nil
	}
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

func generateRandomHash() (string, error) {
	// Создаем буфер для случайных байтов
	randomBytes := make([]byte, 32) // 32 байта -> 256 бит
	_, err := io.ReadFull(rand.Reader, randomBytes)
	if err != nil {
		return "", err
	}

	// Вычисляем хэш от случайных байтов
	hash := sha256.Sum256(randomBytes)

	// Преобразуем хэш в строку в формате HEX
	return hex.EncodeToString(hash[:]), nil
}

func (m *M3u8) ChangeIsNeedCut(value bool) {
	m.isNeedCut = value
}

func (m *M3u8) ChangeIsCancel(value bool) {
	m.isCancel = value
}
