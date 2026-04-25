package engine

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Engine is the central brain that consumes network events and updates state.
type Engine struct {
	state   *state
	source  EventSource
	scripts *ScriptEngine
}

var _ GameState = (*Engine)(nil)

func NewEngine(source EventSource) *Engine {
	e := &Engine{
		state:  &state{},
		source: source,
	}
	e.scripts = newScriptEngine(e)
	go e.listen()
	return e
}

func (e *Engine) Room() RoomInfo     { return e.state.Room() }
func (e *Engine) Vitals() VitalsInfo { return e.state.Vitals() }
func (e *Engine) CharName() string   { return e.state.CharName() }

// Execute runs a command through the alias engine and sends it to the server.
func (e *Engine) Execute(cmd string) error {
	newCmd, swallowed := e.scripts.evalAlias(cmd)
	if swallowed {
		return nil
	}
	return e.source.Write([]byte(newCmd + "\n"))
}

func (e *Engine) listen() {
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
				// Split into lines for trigger evaluation
				lines := strings.Split(string(ev.Data), "\n")
				for _, l := range lines {
					if l != "" {
						e.scripts.evalTrigger(l)
					}
				}
			}
		case <-e.source.ErrorsCh():
			// ignore errors for now, main.go handles connection drops
		}
	}
}

func (e *Engine) handleGMCP(data []byte) {
	// GMCP format: Module.Submessage Payload
	parts := bytes.SplitN(data, []byte(" "), 2)
	if len(parts) < 2 {
		return
	}
	msg := string(parts[0])
	payload := parts[1]

	e.state.mu.Lock()
	defer e.state.mu.Unlock()

	switch {
	case msg == "Char.Vitals":
		var v struct {
			HP   int `json:"hp"`
			Max  int `json:"maxhp"`
			MN   int `json:"mana"`
			MXMN int `json:"maxmana"`
			MV   int `json:"moves"`
			MXMV int `json:"maxmoves"`
		}
		if err := json.Unmarshal(payload, &v); err == nil {
			e.state.vitals = VitalsInfo{
				HP: v.HP, MaxHP: v.Max,
				Mana: v.MN, MaxMana: v.MXMN,
				Move: v.MV, MaxMove: v.MXMV,
			}
		}

	case msg == "Char.Name":
		var n struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(payload, &n); err == nil {
			e.state.charName = n.Name
		}

	case strings.HasPrefix(msg, "Room.Info"):
		var r struct {
			Name string `json:"name"`
			Vnum int    `json:"vnum"`
		}
		if err := json.Unmarshal(payload, &r); err == nil {
			e.state.room = RoomInfo{Vnum: r.Vnum, Name: r.Name}
		}
	}
}
