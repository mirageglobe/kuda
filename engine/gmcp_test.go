package engine

import (
	"testing"
)

func newTestEngine() *Engine {
	src := &mockSource{ch: make(chan Event, 8), errCh: make(chan error, 1)}
	e := &Engine{
		state:  &state{},
		source: src,
		roomCh: make(chan RoomInfo, 32),
	}
	return e
}

func TestHandleGMCP_charVitals(t *testing.T) {
	e := newTestEngine()
	e.handleGMCP([]byte(`char.vitals {"hp":100,"maxhp":200,"mana":50,"maxmana":100,"moves":80,"maxmoves":150}`))
	v := e.Vitals()
	if v.HP != 100 || v.MaxHP != 200 {
		t.Errorf("HP: got %d/%d, want 100/200", v.HP, v.MaxHP)
	}
	if v.Mana != 50 || v.MaxMana != 100 {
		t.Errorf("Mana: got %d/%d, want 50/100", v.Mana, v.MaxMana)
	}
}

func TestHandleGMCP_charName(t *testing.T) {
	e := newTestEngine()
	e.handleGMCP([]byte(`char.name {"name":"Aard"}`))
	if got := e.CharName(); got != "Aard" {
		t.Errorf("want %q, got %q", "Aard", got)
	}
}

func TestHandleGMCP_roomInfo(t *testing.T) {
	e := newTestEngine()
	e.handleGMCP([]byte(`room.info {"num":12345,"name":"Town Square","coords":{"x":10,"y":20,"z":0}}`))
	r := e.Room()
	if r.Vnum != 12345 {
		t.Errorf("vnum: got %d, want 12345", r.Vnum)
	}
	if r.Name != "Town Square" {
		t.Errorf("name: got %q, want %q", r.Name, "Town Square")
	}
	if !r.HasCoords {
		t.Error("want HasCoords=true")
	}
	// Aardwolf coords.x=NS, coords.y=EW; swapped to mapper convention X=EW, Y=NS.
	if r.X != 20 || r.Y != 10 {
		t.Errorf("coords: got X=%d Y=%d, want X=20 Y=10", r.X, r.Y)
	}
}

func TestHandleGMCP_roomExits(t *testing.T) {
	e := newTestEngine()
	e.handleGMCP([]byte(`room.exits {"north":100,"south":200}`))
	exits := e.Room().Exits
	if exits["n"] != 100 || exits["s"] != 200 {
		t.Errorf("exits: got %v", exits)
	}
}

func TestHandleGMCP_unknown(t *testing.T) {
	e := newTestEngine()
	// should not panic on unknown message
	e.handleGMCP([]byte(`unknown.message {}`))
}
