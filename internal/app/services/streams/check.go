package streams

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/repository"
	"stream-recorder/internal/app/services/ffmpeg"
	"stream-recorder/internal/app/services/m3u8"
	"stream-recorder/internal/app/services/models"
	"stream-recorder/internal/app/services/runner"
	"stream-recorder/internal/app/services/streamlink"
	"stream-recorder/pkg/logger"
	"strings"
	"time"
)

type Streams struct {
	sr  *repository.StreamersRepository
	sl  *streamlink.Streamlink
	rp  *runner.Process
	cfg *config.Config

	activeStreamers map[string]bool
	activeM3u8      map[string]*m3u8.M3u8
	sem             chan struct{}
}

func New(sr *repository.StreamersRepository, sl *streamlink.Streamlink, rp *runner.Process, cfg *config.Config, activeStreamers map[string]bool, activeM3u8 map[string]*m3u8.M3u8, sem chan struct{}) *Streams {
	return &Streams{
		sr:              sr,
		sl:              sl,
		rp:              rp,
		cfg:             cfg,
		activeStreamers: activeStreamers,
		activeM3u8:      activeM3u8,
		sem:             sem,
	}
}

func (s *Streams) CheckingForStreams() {
	for {
		streamers, err := s.sr.Get()
		if err != nil {
			logger.Error("Error getting streamers", zap.Error(err))
			time.Sleep(time.Duration(s.cfg.TimeCheck) * time.Second)
			continue
		}

		for _, stream := range streamers {
			if !s.activeStreamers[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] {
				s.activeStreamers[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = true
				go s.checkingForStream(stream)
			}
		}

		time.Sleep(time.Duration(s.cfg.TimeCheck) * time.Second)
	}
}

func (s *Streams) checkingForStream(stream models.Streamers) {
	var masterHls, mediaHls string
	var err error
	masterHls, err = s.sl.Twitch.GetMasterPlaylist(stream.Username)
	if err != nil {
		logger.Error("Error getting master playlist", zap.Error(err))
		s.activeStreamers[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = false
		return
	}

	for {
		mediaHls, err = s.sl.Twitch.FindMediaPlaylist(masterHls, stream.Quality)
		if err == nil {
			break
		} else if strings.Contains(err.Error(), "HTTP error: 403") {
			masterHls, err = s.sl.Twitch.GetMasterPlaylist(stream.Username)
			if err != nil {
				logger.Error("Error getting master playlist", zap.Error(err))
				s.activeStreamers[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = false
				return
			}
		}

		logger.Debugf("The streamer is not broadcasting live, waiting...", stream.Username, stream.Platform)
		time.Sleep(time.Duration(s.cfg.TimeCheck) * time.Second)
	}

	logger.Infof("The streamer has started a live broadcast, I'm starting the recording...", stream.Username, stream.Platform)
	s.activeM3u8[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = m3u8.New(stream.Platform, stream.Username, stream.SplitSegments, stream.TimeSegment, s.rp, s.cfg, s.sem)
	err = s.activeM3u8[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)].Run(mediaHls)
	if err != nil {
		logger.Error("Error running m3u8", zap.Error(err))
		s.activeStreamers[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = false
		return
	}
	s.activeStreamers[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = false
}

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
