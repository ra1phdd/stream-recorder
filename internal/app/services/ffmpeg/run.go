package ffmpeg

import (
	"bufio"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"path/filepath"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/services/runner"
	"stream-recorder/pkg/embed"
	"stream-recorder/pkg/logger"
	"strings"
)

type Ffmpeg struct {
	rp *runner.Process
	c  *config.Config

	cmd *exec.Cmd
}

func New(rp *runner.Process, c *config.Config) *Ffmpeg {
	return &Ffmpeg{
		rp:  rp,
		c:   c,
		cmd: nil,
	}
}

func (f *Ffmpeg) Run(filePath, output string) error {
	logger.Debug("Starting ffmpeg", zap.String("filepath", filePath), zap.String("output", output))

	args := []string{
		"-y",
		"-err_detect", "ignore_err",
		"-loglevel", "warning",
		"-f", "concat",
		"-safe", "0",
		"-i", filePath,
		"-async", "1",
		"-fps_mode", "cfr",
		"-c:v", f.c.VideoCodec,
		"-c:a", f.c.AudioCodec,
		fmt.Sprintf("%s_download.%s", output, f.c.FileFormat),
	}

	f.cmd = exec.Command(embed.GetTempFileName("ffmpeg"), args...)
	err := f.rp.Run("ffmpeg", f.cmd, filePath, output, f.handlerStdout, f.waitForExit)
	if err != nil {
		return err
	}

	logger.Debug("ffmpeg started successfully", zap.Any("cmd", f.cmd))
	return nil
}

func (f *Ffmpeg) handlerStdout(line string) {
	if strings.Contains(line, "started") {
		// TODO
	}
}

func (f *Ffmpeg) waitForExit(cmd *exec.Cmd, filePath, output string) {
	if err := cmd.Wait(); err != nil {
		logger.Error("ffmpeg exited with an error", zap.Any("cmd", cmd), zap.String("filepath", filePath), zap.Error(err))
	} else {
		err := os.Rename(fmt.Sprintf("%s_download.%s", output, f.c.FileFormat), fmt.Sprintf("%s.%s", output, f.c.FileFormat))
		if err != nil {
			logger.Error("Failed to rename ffmpeg", zap.String("filepath", filePath), zap.String("output", output), zap.Error(err))
		}

		file, err := os.Open(filePath)
		if err != nil {
			logger.Error("ffmpeg failed to open file", zap.String("filepath", filePath), zap.Error(err))
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "file '") {
				name := strings.TrimSuffix(strings.TrimPrefix(line, "file '"), "'")

				dir, _ := filepath.Split(filePath)
				err := os.Remove(filepath.Join(dir, name))
				if err != nil {
					logger.Error("Error when deleting a chunk file from tmp", zap.String("filename", name), zap.Error(err))
				} else {
					logger.Debug("Deleted a chunk file from tmp", zap.String("filename", name))
				}
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Error("Buffer scanning error", zap.String("filepath", filePath), zap.Error(err))
		}

		err = os.Remove(filePath)
		if err != nil {
			logger.Error("Error when deleting a ffmpeg file in tmp", zap.String("filepath", filePath), zap.Error(err))
		} else {
			logger.Debug("Deleted a txt file from ffmpeg in tmp", zap.String("filepath", filePath))
		}
		logger.Info("Segment is recorded", zap.String("filepath", filePath))
	}
}

func (f *Ffmpeg) Kill() error {
	logger.Debug("Stopping ffmpeg", zap.Any("cmd", f.cmd))

	err := f.rp.Kill("ffmpeg", f.cmd)
	if err != nil {
		logger.Error("Failed to kill ffmpeg", zap.Error(err))
		return err
	}

	logger.Debug("ffmpeg stopped successfully", zap.Any("cmd", f.cmd))
	return nil
}
