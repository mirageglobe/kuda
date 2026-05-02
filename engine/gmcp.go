package engine

import (
	"bytes"
	"encoding/json"
	"strings"
)

var dirAbbrev = map[string]string{
	"north": "n", "south": "s", "east": "e", "west": "w",
	"northeast": "ne", "northwest": "nw",
	"southeast": "se", "southwest": "sw",
	"up": "u", "down": "d",
}

func normalizeExits(exits map[string]int) map[string]int {
	out := make(map[string]int, len(exits))
	for k, v := range exits {
		if short, ok := dirAbbrev[strings.ToLower(k)]; ok {
			out[short] = v
		} else {
			out[k] = v
		}
	}
	return out
}

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

	var room RoomInfo
	var sendRoom bool

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
			e.state.setVitals(vi)
		}

	case msgLower == "char.name":
		var n struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(payload, &n); err == nil {
			e.state.setCharName(n.Name)
		}

	case strings.HasPrefix(msgLower, "room.info"):
		var r struct {
			Num    int         `json:"num"`
			Vnum   interface{} `json:"vnum"`
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
			ri := RoomInfo{Vnum: vnum, Name: r.Name}
			if r.Coords != nil {
				ri.X, ri.Y, ri.Z = r.Coords.X, r.Coords.Y, r.Coords.Z
				ri.HasCoords = true
			}
			room = e.state.setRoom(ri)
			sendRoom = true
		}

	case strings.HasPrefix(msgLower, "room.exits"):
		var exits map[string]int
		if err := json.Unmarshal(payload, &exits); err == nil {
			room = e.state.setRoomExits(normalizeExits(exits))
			sendRoom = true
		}
	}

	if sendRoom {
		select {
		case e.roomCh <- room:
		default:
		}
	}
}
