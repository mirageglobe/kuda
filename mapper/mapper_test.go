package mapper

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFirstRoomAtOrigin(t *testing.T) {
	m := New()
	m.Update(RoomData{Vnum: 1, Name: "Start"})
	r := m.rooms[1]
	if r == nil {
		t.Fatal("room not added")
	}
	if r.x != 0 || r.y != 0 {
		t.Errorf("want (0,0), got (%d,%d)", r.x, r.y)
	}
}

func TestProvidedCoords(t *testing.T) {
	m := New()
	m.Update(RoomData{Vnum: 1, Name: "Start", X: 5, Y: 3, HasCoords: true})
	r := m.rooms[1]
	if r.x != 5 || r.y != 3 {
		t.Errorf("want (5,3), got (%d,%d)", r.x, r.y)
	}
}

func TestInferredNorthPosition(t *testing.T) {
	m := New()
	m.Update(RoomData{Vnum: 1, Name: "Start", Exits: map[string]int{"n": 2}})
	m.Update(RoomData{Vnum: 2, Name: "North"})
	r := m.rooms[2]
	if r.x != 0 || r.y != 1 {
		t.Errorf("want (0,1) north of origin, got (%d,%d)", r.x, r.y)
	}
}

func TestRenderEmptyMapper(t *testing.T) {
	m := New()
	out := m.Render(20, 5)
	if len(out) == 0 {
		t.Error("render should return non-empty string even with no rooms")
	}
}

func TestRenderCurrentRoomMarker(t *testing.T) {
	m := New()
	m.Update(RoomData{Vnum: 1, Name: "Test", X: 0, Y: 0, HasCoords: true})
	out := m.Render(20, 5)
	if !strings.Contains(out, "[*]") {
		t.Errorf("expected [*] in output:\n%s", out)
	}
}

func TestRenderEastExitConnector(t *testing.T) {
	m := New()
	m.Update(RoomData{Vnum: 1, Name: "A", X: 0, Y: 0, HasCoords: true, Exits: map[string]int{"e": 2}})
	m.Update(RoomData{Vnum: 2, Name: "B", X: 1, Y: 0, HasCoords: true})
	out := m.Render(40, 5)
	if !strings.Contains(out, "-") {
		t.Errorf("expected '-' east connector in output:\n%s", out)
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map.json")

	m := Load(path)
	m.Update(RoomData{Vnum: 1, Name: "Start", X: 0, Y: 0, HasCoords: true})
	m.Update(RoomData{Vnum: 2, Name: "North", X: 0, Y: 1, HasCoords: true})

	if err := m.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	m2 := Load(path)
	if len(m2.rooms) != 2 {
		t.Errorf("want 2 rooms after load, got %d", len(m2.rooms))
	}
	if r := m2.rooms[1]; r == nil || r.name != "Start" {
		t.Errorf("room 1 not restored correctly: %+v", r)
	}
	if r := m2.rooms[2]; r == nil || r.x != 0 || r.y != 1 {
		t.Errorf("room 2 coords not restored: %+v", r)
	}
}

func TestLoadMissingFile(t *testing.T) {
	m := Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if m == nil {
		t.Fatal("Load should return non-nil mapper for missing file")
	}
	if len(m.rooms) != 0 {
		t.Errorf("want 0 rooms for missing file, got %d", len(m.rooms))
	}
}

func TestUpdateExitsOnExistingRoom(t *testing.T) {
	m := New()
	m.Update(RoomData{Vnum: 1, Name: "Start"})
	m.Update(RoomData{Vnum: 1, Name: "Start", Exits: map[string]int{"n": 2}})
	r := m.rooms[1]
	if r.exits["n"] != 2 {
		t.Errorf("expected exit n=2, got %v", r.exits)
	}
}
