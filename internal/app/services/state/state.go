package state

import "stream-recorder/internal/app/services/m3u8"

type State struct {
	am map[string]*m3u8.M3u8
	as map[string]bool
}

func New() *State {
	return &State{
		am: make(map[string]*m3u8.M3u8),
		as: make(map[string]bool),
	}
}

func (s *State) GetActiveStreamers(key string) bool {
	return s.as[key]
}

func (s *State) GetActiveM3u8(key string) *m3u8.M3u8 {
	return s.am[key]
}

func (s *State) UpdateActiveStreamers(key string, value bool) {
	s.as[key] = value
}

func (s *State) UpdateActiveM3u8(key string, value *m3u8.M3u8) {
	s.am[key] = value
}
