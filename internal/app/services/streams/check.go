package streams

import (
	"fmt"
	"go.uber.org/zap"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/repository"
	"stream-recorder/internal/app/services/m3u8"
	"stream-recorder/internal/app/services/models"
	"stream-recorder/internal/app/services/runner"
	"stream-recorder/internal/app/services/streamlink"
	"stream-recorder/pkg/logger"
	"time"
)

type Streams struct {
	sr  *repository.StreamersRepository
	sl  *streamlink.Streamlink
	rp  *runner.Process
	cfg *config.Config

	activeStreamers map[string]bool
	activeM3u8      map[string]*m3u8.M3u8
}

func New(sr *repository.StreamersRepository, sl *streamlink.Streamlink, rp *runner.Process, cfg *config.Config, activeStreamers map[string]bool, activeM3u8 map[string]*m3u8.M3u8) *Streams {
	return &Streams{
		sr:              sr,
		sl:              sl,
		rp:              rp,
		cfg:             cfg,
		activeStreamers: activeStreamers,
		activeM3u8:      activeM3u8,
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
				go s.checkingForStream(stream)
				s.activeStreamers[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = true
			}
		}

		time.Sleep(time.Duration(s.cfg.TimeCheck) * time.Second)
	}
}

func (s *Streams) checkingForStream(stream models.Streamers) {
	for {
		masterHls, err := s.sl.Twitch.GetMasterPlaylist(stream.Username)
		if err != nil {
			s.activeStreamers[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = false
			logger.Error("Error getting master playlist", zap.Error(err))
		}

		var mediaHls string
		for {
			mediaHls, err = s.sl.Twitch.FindMediaPlaylist(masterHls, stream.Quality)
			if err == nil {
				break
			}

			logger.Infof("The streamer is not broadcasting live, waiting...", stream.Username, stream.Platform)
			time.Sleep(time.Duration(s.cfg.TimeCheck) * time.Second)
		}

		s.activeM3u8[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = m3u8.New(stream.Platform, stream.Username, s.rp, s.cfg)
		err = s.activeM3u8[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)].Run(mediaHls)
		if err != nil {
			s.activeStreamers[fmt.Sprintf("%s-%s", stream.Platform, stream.Username)] = false
			logger.Error("Error running m3u8", zap.Error(err))
		}
	}
}
