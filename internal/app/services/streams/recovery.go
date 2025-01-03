package streams

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"stream-recorder/internal/app/services/ffmpeg"
	"stream-recorder/internal/app/services/m3u8"
	"stream-recorder/pkg/logger"
	"strings"
)

func (s *Streams) Recovery() {
	logger.Warn("Recovering streams...")
	var files = make(map[string]string)

	err := filepath.Walk(s.cfg.TempPATH, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			if info.Size() != 0 {
				logger.Warn("Video file found", zap.String("path", filepath.Dir(path)), zap.String("name", info.Name()))
				files[filepath.Dir(path)] = info.Name()
			} else {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove %s: %w", path, err)
				}
			}
		}

		return nil
	})

	if err != nil {
		logger.Error("Error reading temp path", zap.Error(err))
		return
	}

	for dir, name := range files {
		err := m3u8.CreateDirectoryIfNotExist(dir)
		if err != nil {
			logger.Error("Error creating directory", zap.String("path", dir), zap.Error(err))
			return
		}

		filePath := filepath.Join(dir, name)
		output := filepath.Join(s.cfg.MediaPATH, strings.TrimPrefix(dir, fmt.Sprintf("%s/", s.cfg.TempPATH)), strings.TrimSuffix(name, ".txt"))
		if err := ffmpeg.New(s.rp, s.cfg).Run(filePath, output); err != nil {
			logger.Error("Error running external process", zap.String("fileSegments", name), zap.String("filepath", filePath), zap.String("output", output), zap.Error(err))
		}
	}

	logger.Warn("Successfully recovered streams")
}
