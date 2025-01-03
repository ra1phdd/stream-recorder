package tmp

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func ClearToTime(dirPath string, dur time.Duration) error {
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

func RemoveEmptyDirs(dir string) error {
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
