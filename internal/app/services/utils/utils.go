package utils

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"stream-recorder/pkg/logger"
	"strings"
	"time"
)

type Utils struct {
	log *logger.Logger
}

func New(log *logger.Logger) *Utils {
	return &Utils{
		log: log,
	}
}

func (u *Utils) GetShortFileName(url string) string {
	hasher := md5.New()
	hasher.Write([]byte(url))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (u *Utils) FormatDuration(d time.Duration) string {
	hours := d / time.Hour
	d -= hours * time.Hour
	mins := d / time.Minute
	d -= mins * time.Minute
	secs := d / time.Second
	return fmt.Sprintf("%dh%dm%ds", hours, mins, secs)
}

func (u *Utils) ExtractNumber(filename string) int {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 1 {
		return -1
	}
	num, _ := strconv.Atoi(parts[0])
	return num
}

func (u *Utils) RemoveDateFromPath(input string) string {
	parts := strings.Split(input, "_")
	if len(parts) < 2 {
		return input
	}
	return strings.Join(parts[:len(parts)-1], "_")
}

func (u *Utils) CreateDirectoryIfNotExist(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		u.log.Debug("Directory does not exist. Creating...", slog.String("outputDir", path))
		if err := os.Mkdir(path, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		u.log.Debug("Directory created successfully", slog.String("outputDir", path))
	}
	return nil
}

func (u *Utils) ExtractFilenamesFromTxt(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var files []string
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "file '") && strings.HasSuffix(line, "'") {
			filename := strings.TrimPrefix(line, "file '")
			filename = strings.TrimSuffix(filename, "'")
			files = append(files, filename)
		}
	}
	return files, nil
}

func (u *Utils) ClearToTime(dirPath string, dur time.Duration) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to get info for %s: %w", entryPath, err)
		}

		if time.Since(info.ModTime()) > dur {
			if err := os.RemoveAll(entryPath); err != nil {
				return fmt.Errorf("failed to remove %s: %w", entryPath, err)
			}
		}
	}

	return nil
}

func (u *Utils) RemoveEmptyDirs(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("couldn't read the directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(dir, entry.Name())
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				return fmt.Errorf("couldn't read the directory %s: %w", dir, err)
			}

			if len(subEntries) == 0 {
				err = os.Remove(subDir)
				if err != nil {
					return fmt.Errorf("couldn't read the directory %s: %w", dir, err)
				}
			}
		}
	}

	return nil
}

func (u *Utils) RandomToken(length int, choices string) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be greater than 0")
	}
	if len(choices) == 0 {
		return "", errors.New("choices string must not be empty")
	}

	var result strings.Builder
	choicesLen := len(choices)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < length; i++ {
		randomIndex := r.Intn(choicesLen)
		result.WriteByte(choices[randomIndex])
	}

	return result.String(), nil
}
