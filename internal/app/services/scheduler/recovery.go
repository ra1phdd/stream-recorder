package scheduler

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"stream-recorder/internal/app/services/utils"
	"stream-recorder/pkg/ffmpeg"
	"strings"
)

func (s *Scheduler) Recovery() {
	s.log.Warn("Recovering streams...")
	var files = make(map[string]string)

	err := filepath.Walk(s.cfg.TempPATH, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			if info.Size() != 0 {
				s.log.Warn("Video file found", slog.String("path", filepath.Dir(path)), slog.String("name", info.Name()))
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
		s.log.Error("Error reading temp path", err)
		return
	}

	u := utils.New(s.log)
	f, err := ffmpeg.NewFfmpeg(s.cfg.FFmpegPATH)
	if err != nil {
		return
	}

	for dir, name := range files {
		err := u.CreateDirectoryIfNotExist(dir)
		if err != nil {
			s.log.Error("Error creating directory", err, slog.String("path", dir))
			return
		}

		filePath := filepath.Join(dir, name)
		output := filepath.Join(s.cfg.MediaPATH, strings.TrimPrefix(filepath.Clean(dir), s.cfg.TempPATH), strings.TrimSuffix(name, ".txt")+"."+s.cfg.FileFormat)

		err = f.Yes().
			ErrDetect("ignore_err").
			LogLevel("warning").
			Format("concat").
			Safe(0).
			Async(1).
			FpsMode("cfr").
			VideoCodec("copy").
			AudioCodec("copy").
			Execute([]string{filePath}, output)
		if err != nil {
			s.log.Error("Error run ffmpeg", err, slog.String("path", filePath), slog.String("output", output))
		}

		txt, err := u.ExtractFilenamesFromTxt(filePath)
		if err != nil {
			s.log.Error("Error extracting filenames", err, slog.String("path", filePath))
			return
		}

		for _, file := range txt {
			os.Remove(filepath.Join(dir, file))
		}

		os.Remove(filePath)
	}

	s.log.Warn("Successfully recovered streams")
}
