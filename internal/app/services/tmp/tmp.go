package tmp

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Очистить все файлы в директории, старше dur
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
