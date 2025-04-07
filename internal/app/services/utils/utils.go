package utils

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
