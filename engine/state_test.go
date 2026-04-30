package engine

import (
	"sync"
	"testing"
)

func TestState_defaults(t *testing.T) {
	s := &state{}
	if s.CharName() != "" {
		t.Errorf("want empty char name, got %q", s.CharName())
	}
	if got := s.Room(); got.Vnum != 0 {
		t.Errorf("want zero vnum, got %d", got.Vnum)
	}
	if got := s.Vitals(); got.HP != 0 {
		t.Errorf("want zero hp, got %d", got.HP)
	}
}

func TestState_concurrent(t *testing.T) {
	s := &state{}
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.mu.Lock()
			s.charName = "Tester"
			s.mu.Unlock()
		}()
		go func() { defer wg.Done(); _ = s.CharName() }()
	}
	wg.Wait()
}
