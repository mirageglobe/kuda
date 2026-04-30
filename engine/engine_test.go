package engine

import (
	"testing"
)

type mockSource struct {
	ch    chan Event
	errCh chan error
	sent  [][]byte
}

func (m *mockSource) EventsCh() <-chan Event  { return m.ch }
func (m *mockSource) ErrorsCh() <-chan error  { return m.errCh }
func (m *mockSource) Write(data []byte) error { m.sent = append(m.sent, data); return nil }

func newEngineWithMock() (*Engine, *mockSource) {
	src := &mockSource{ch: make(chan Event, 8), errCh: make(chan error, 1)}
	e := NewEngine(src)
	return e, src
}

func TestNewEngine_initialState(t *testing.T) {
	e, _ := newEngineWithMock()
	close(e.source.(*mockSource).ch)

	if e.CharName() != "" {
		t.Errorf("want empty char name, got %q", e.CharName())
	}
	if e.Room().Vnum != 0 {
		t.Error("want zero room vnum")
	}
}

func TestEngine_toggleLua(t *testing.T) {
	e, src := newEngineWithMock()
	defer close(src.ch)

	// starts enabled; first toggle -> disabled
	if e.ToggleLua() {
		t.Error("first toggle should disable lua")
	}
	// second toggle -> enabled
	if !e.ToggleLua() {
		t.Error("second toggle should enable lua")
	}
}

func TestEngine_executePassthrough(t *testing.T) {
	e, src := newEngineWithMock()
	e.luaEnabled = false
	defer close(src.ch)

	if err := e.Execute("look"); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(src.sent) != 1 || string(src.sent[0]) != "look\n" {
		t.Errorf("want sent=[\"look\\n\"], got %v", src.sent)
	}
}
