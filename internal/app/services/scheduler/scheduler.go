package scheduler

import (
	"fmt"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/models"
	"stream-recorder/internal/app/repository"
	"stream-recorder/internal/app/services/m3u8"
	"stream-recorder/internal/app/services/state"
	"stream-recorder/internal/app/services/streamlink"
	"stream-recorder/internal/app/services/utils"
	"stream-recorder/pkg/logger"
	"strings"
	"time"
)

type Scheduler struct {
	log *logger.Logger
	sr  *repository.StreamersRepository
	sl  *streamlink.Streamlink
	cfg *config.Config
	st  *state.State
	u   *utils.Utils
}

func New(log *logger.Logger, sr *repository.StreamersRepository, sl *streamlink.Streamlink, cfg *config.Config, st *state.State, u *utils.Utils) *Scheduler {
	return &Scheduler{
		log: log,
		sr:  sr,
		sl:  sl,
		cfg: cfg,
		st:  st,
		u:   u,
	}
}

func (s *Scheduler) CheckingForStreams() {
	for {
		streamers, err := s.sr.Get()
		if err != nil {
			s.log.Error("Error getting streamers", err)
			time.Sleep(time.Duration(s.cfg.TimeCheck) * time.Second)
			continue
		}

		for _, stream := range streamers {
			key := fmt.Sprintf("%s-%s", stream.Platform, stream.Username)
			if !s.st.GetActiveStreamers(key) {
				s.st.UpdateActiveStreamers(key, true)
				go s.checkingForStream(stream)
			}
		}

		time.Sleep(time.Duration(s.cfg.TimeCheck) * time.Second)
	}
}

func (s *Scheduler) checkingForStream(stream models.Streamers) {
	key := fmt.Sprintf("%s-%s", stream.Platform, stream.Username)
	var masterHls, mediaHls string
	var err error
	masterHls, err = s.sl.Platform.GetMasterPlaylist(stream.Username)
	if err != nil {
		s.log.Error("Error getting master playlist", err)
		s.st.UpdateActiveStreamers(key, false)
		return
	}

	for {
		if !s.st.GetActiveStreamers(key) {
			return
		}

		mediaHls, err = s.sl.Platform.FindMediaPlaylist(masterHls, stream.Quality)
		if err == nil {
			break
		} else if strings.Contains(err.Error(), "HTTP error: 403") {
			masterHls, err = s.sl.Platform.GetMasterPlaylist(stream.Username)
			if err != nil {
				s.log.Error("Error getting master playlist", err)
				s.st.UpdateActiveStreamers(key, false)
				return
			}
		}

		s.log.Debug(fmt.Sprintf("[%s/%s] The streamer is not broadcasting live, waiting...", stream.Username, stream.Platform))
		time.Sleep(time.Duration(s.cfg.TimeCheck) * time.Second)
	}

	s.log.Info(fmt.Sprintf("[%s/%s] The streamer has started a live broadcast, I'm starting the recording...", stream.Username, stream.Platform))

	val, err := m3u8.New(s.log, stream.Platform, stream.Username, stream.SplitSegments, stream.TimeSegment, s.cfg, s.u)
	if err != nil {
		s.log.Error("Error creating m3u8", err)
	}
	s.st.UpdateActiveM3u8(key, val)

	err = s.st.GetActiveM3u8(key).Run(mediaHls)
	if err != nil {
		s.log.Error("Error running m3u8", err)
		s.st.UpdateActiveStreamers(key, false)
		return
	}
	s.st.UpdateActiveStreamers(key, false)
}
