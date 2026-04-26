package engine

import "strings"

// Engine is the central brain that consumes network events and updates state.
type Engine struct {
	state      *state
	source     EventSource
	scripts    *ScriptEngine
	luaEnabled bool
	roomCh     chan RoomInfo
}

var _ GameState = (*Engine)(nil)

func NewEngine(source EventSource) *Engine {
	e := &Engine{
		state:      &state{},
		source:     source,
		luaEnabled: true,
		roomCh:     make(chan RoomInfo, 32),
	}
	e.scripts = newScriptEngine(e)
	go e.listen()
	return e
}

// RoomCh returns a channel that emits RoomInfo whenever the current room changes.
func (e *Engine) RoomCh() <-chan RoomInfo { return e.roomCh }

func (e *Engine) Room() RoomInfo     { return e.state.Room() }
func (e *Engine) Vitals() VitalsInfo { return e.state.Vitals() }
func (e *Engine) CharName() string   { return e.state.CharName() }

// ToggleLua enables or disables the script engine.
func (e *Engine) ToggleLua() bool {
	e.luaEnabled = !e.luaEnabled
	return e.luaEnabled
}

// Execute runs a command through the alias engine and sends it to the server.
func (e *Engine) Execute(cmd string) error {
	if e.luaEnabled {
		newCmd, swallowed := e.scripts.evalAlias(cmd)
		if swallowed {
			return nil
		}
		return e.source.Write([]byte(newCmd + "\n"))
	}
	return e.source.Write([]byte(cmd + "\n"))
}

func (e *Engine) listen() {
	defer close(e.roomCh)
	for {
		select {
		case ev, ok := <-e.source.EventsCh():
			if !ok {
				e.scripts.Close()
				return
			}
			switch ev.Type {
			case EventGMCP:
				e.handleGMCP(ev.Data)
			case EventText:
				if e.luaEnabled {
					lines := strings.Split(string(ev.Data), "\n")
					for _, l := range lines {
						if l != "" {
							e.scripts.evalTrigger(l)
						}
					}
				}
			}
		case <-e.source.ErrorsCh():
			// ignore errors; main.go handles connection drops
		}
	}
}
