package engine

import "sync"

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

func (s *state) setVitals(v VitalsInfo) {
	s.mu.Lock()
	s.vitals = v
	s.mu.Unlock()
}

func (s *state) setCharName(n string) {
	s.mu.Lock()
	s.charName = n
	s.mu.Unlock()
}

func (s *state) setRoom(r RoomInfo) RoomInfo {
	s.mu.Lock()
	s.room = r
	s.mu.Unlock()
	return r
}

func (s *state) setRoomExits(exits map[string]int) RoomInfo {
	s.mu.Lock()
	s.room.Exits = exits
	r := s.room
	s.mu.Unlock()
	return r
}
