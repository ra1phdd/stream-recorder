package state

import (
	"stream-recorder/internal/app/services/m3u8"
	"sync"
)

type State struct {
	am map[string]*m3u8.M3u8
	as map[string]bool

	muAm sync.Mutex
	muAs sync.Mutex
}

func New() *State {
	return &State{
		am: make(map[string]*m3u8.M3u8),
		as: make(map[string]bool),
	}
}

func (s *State) GetActiveStreamers(key string) bool {
	s.muAs.Lock()
	defer s.muAs.Unlock()

	return s.as[key]
}

func (s *State) GetActiveM3u8(key string) *m3u8.M3u8 {
	s.muAm.Lock()
	defer s.muAm.Unlock()

	return s.am[key]
}

func (s *State) UpdateActiveStreamers(key string, value bool) {
	s.muAs.Lock()
	defer s.muAs.Unlock()

	s.as[key] = value
}

func (s *State) UpdateActiveM3u8(key string, value *m3u8.M3u8) {
	s.muAm.Lock()
	defer s.muAm.Unlock()

	s.am[key] = value
}
