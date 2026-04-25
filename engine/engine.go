package engine

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Engine is the central brain that consumes network events and updates state.
type Engine struct {
	state      *state
	source     EventSource
	scripts    *ScriptEngine
	luaEnabled bool
}

var _ GameState = (*Engine)(nil)

func NewEngine(source EventSource) *Engine {
	e := &Engine{
		state:      &state{},
		source:     source,
		luaEnabled: true,
	}
	e.scripts = newScriptEngine(e)
	go e.listen()
	return e
}

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
					// Split into lines for trigger evaluation
					lines := strings.Split(string(ev.Data), "\n")
					for _, l := range lines {
						if l != "" {
							e.scripts.evalTrigger(l)
						}
					}
				}
			}
		case <-e.source.ErrorsCh():
			// ...

			// ignore errors for now, main.go handles connection drops
		}
	}
}

func (e *Engine) handleGMCP(data []byte) {
	// GMCP format: Module.Submessage [payload]
	// Some servers don't put a space before the JSON payload
	idx := bytes.IndexAny(data, " {")
	var msg string
	var payload []byte
	if idx == -1 {
		msg = string(data)
	} else {
		msg = string(data[:idx])
		payload = bytes.TrimSpace(data[idx:])
	}

	// Case-insensitive matching for modules
	msgLower := strings.ToLower(msg)

	e.state.mu.Lock()
	defer e.state.mu.Unlock()

	switch {
	case msgLower == "char.vitals":
		var v struct {
			HP    int `json:"hp"`
			Max   int `json:"maxhp"`
			MN    int `json:"mana"`
			MXMN  int `json:"maxmana"`
			MV    int `json:"moves"`
			MXMV  int `json:"maxmoves"`
			MP    int `json:"mp"`
			MaxMP int `json:"maxmp"`
			SP    int `json:"sp"`
			MaxSP int `json:"maxsp"`
		}
		if err := json.Unmarshal(payload, &v); err == nil {
			vi := VitalsInfo{
				HP: v.HP, MaxHP: v.Max,
				Mana: v.MN, MaxMana: v.MXMN,
				Move: v.MV, MaxMove: v.MXMV,
			}
			// fallback for mp/sp naming
			if vi.Mana == 0 {
				vi.Mana = v.MP
			}
			if vi.MaxMana == 0 {
				vi.MaxMana = v.MaxMP
			}
			if vi.Move == 0 {
				vi.Move = v.SP
			}
			if vi.MaxMove == 0 {
				vi.MaxMove = v.MaxSP
			}

			e.state.vitals = vi
		}

	case msgLower == "char.name":
		var n struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(payload, &n); err == nil {
			e.state.charName = n.Name
		}

	case strings.HasPrefix(msgLower, "room.info"):
		var r struct {
			Name string      `json:"name"`
			Vnum interface{} `json:"vnum"`
		}
		if err := json.Unmarshal(payload, &r); err == nil {
			var vnum int
			switch v := r.Vnum.(type) {
			case float64:
				vnum = int(v)
			case string:
				// just ignore or parse if needed
			}
			e.state.room = RoomInfo{Vnum: vnum, Name: r.Name}
		}
	}
}
