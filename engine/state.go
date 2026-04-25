package engine

import "sync"

// state holds the current world and character state.
type state struct {
	mu       sync.RWMutex
	charName string
	vitals   VitalsInfo
	room     RoomInfo
}

func (s *state) Room() RoomInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.room
}

func (s *state) Vitals() VitalsInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vitals
}

func (s *state) CharName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.charName
}
