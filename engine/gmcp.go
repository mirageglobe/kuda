package engine

import (
	"bytes"
	"encoding/json"
	"strings"
)

func (e *Engine) handleGMCP(data []byte) {
	idx := bytes.IndexAny(data, " {")
	var msg string
	var payload []byte
	if idx == -1 {
		msg = string(data)
	} else {
		msg = string(data[:idx])
		payload = bytes.TrimSpace(data[idx:])
	}
	msgLower := strings.ToLower(msg)

	var sendRoom bool

	e.state.mu.Lock()

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
			Num    int         `json:"num"`  // Aardwolf uses "num" for room ID
			Vnum   interface{} `json:"vnum"` // generic MUD fallback
			Name   string      `json:"name"`
			Coords *struct {
				X int `json:"x"`
				Y int `json:"y"`
				Z int `json:"z"`
			} `json:"coords"`
		}
		if err := json.Unmarshal(payload, &r); err == nil {
			vnum := r.Num
			if vnum == 0 {
				if v, ok := r.Vnum.(float64); ok {
					vnum = int(v)
				}
			}
			room := RoomInfo{Vnum: vnum, Name: r.Name}
			if r.Coords != nil {
				room.X, room.Y, room.Z = r.Coords.X, r.Coords.Y, r.Coords.Z
				room.HasCoords = true
			}
			e.state.room = room
			sendRoom = true
		}

	case strings.HasPrefix(msgLower, "room.exits"):
		var exits map[string]int
		if err := json.Unmarshal(payload, &exits); err == nil {
			e.state.room.Exits = exits
			sendRoom = true
		}
	}

	room := e.state.room
	e.state.mu.Unlock()

	if sendRoom {
		select {
		case e.roomCh <- room:
		default:
		}
	}
}
