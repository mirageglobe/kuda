package mapper

// Package mapper builds a room graph from GMCP data and renders a 2-D ASCII
// mini-map. It owns the room graph and rendering. It does NOT own game state —
// the wiring layer (main.go) feeds room updates via Update().

import (
	"strings"
	"sync"
)

// RoomData is the input type for mapper updates, mirroring engine.RoomInfo
// without creating a cross-package dependency.
type RoomData struct {
	Vnum      int
	Name      string
	Exits     map[string]int // direction -> destination vnum
	X, Y, Z   int
	HasCoords bool
}

type room struct {
	vnum  int
	name  string
	exits map[string]int
	x, y  int
}

// cardinal exit directions and their (dx,dy) grid deltas.
var dirDelta = map[string][2]int{
	"n": {0, 1}, "s": {0, -1},
	"e": {1, 0}, "w": {-1, 0},
	"ne": {1, 1}, "nw": {-1, 1},
	"se": {1, -1}, "sw": {-1, -1},
}

// Mapper tracks visited rooms and renders a 2-D mini-map.
type Mapper struct {
	mu          sync.RWMutex
	rooms       map[int]*room
	current     int
	savePath    string
	updateCount int
}

// New returns an empty Mapper.
func New() *Mapper {
	return &Mapper{rooms: make(map[int]*room)}
}

// Update records a room visit. When HasCoords is true the server-provided
// coordinates are used directly; otherwise position is inferred from exits.
func (m *Mapper) Update(d RoomData) {
	m.mu.Lock()

	if _, exists := m.rooms[d.Vnum]; !exists {
		x, y := m.inferPosition(d.Vnum)
		if d.HasCoords {
			x, y = d.X, d.Y
		}
		m.rooms[d.Vnum] = &room{vnum: d.Vnum, name: d.Name, exits: d.Exits, x: x, y: y}
	} else {
		r := m.rooms[d.Vnum]
		r.name = d.Name
		if d.Exits != nil {
			r.exits = d.Exits
		}
		if d.HasCoords {
			r.x, r.y = d.X, d.Y
		}
	}
	m.current = d.Vnum
	m.updateCount++
	shouldSave := m.savePath != "" && m.updateCount%saveInterval == 0

	m.mu.Unlock()

	if shouldSave {
		go func() { _ = m.Save() }()
	}
}

// inferPosition guesses where a new room sits relative to the current room's exits.
func (m *Mapper) inferPosition(vnum int) (int, int) {
	cur := m.rooms[m.current]
	if cur == nil {
		return 0, 0
	}
	for dir, dest := range cur.exits {
		if dest == vnum {
			if d, ok := dirDelta[dir]; ok {
				return cur.x + d[0], cur.y + d[1]
			}
		}
	}
	return cur.x + 1, cur.y
}

// Render produces a w×h ASCII mini-map centred on the current room.
// Room cells are 3 chars wide ("[ ]" or "[*]"); corridors occupy 1 char.
// Grid steps: 4 chars horizontally, 2 chars vertically.
func (m *Mapper) Render(w, h int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cur := m.rooms[m.current]
	if cur == nil || w < 3 || h < 1 {
		return strings.Repeat(" ", w)
	}

	const stepX, stepY = 4, 2
	cols := w / stepX
	rows := h / stepY
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	grid := make([][]byte, h)
	for i := range grid {
		row := make([]byte, w)
		for j := range row {
			row[j] = ' '
		}
		grid[i] = row
	}

	set := func(px, py int, ch byte) {
		if py >= 0 && py < h && px >= 0 && px < w {
			grid[py][px] = ch
		}
	}

	ox := cols / 2
	oy := rows / 2

	for _, r := range m.rooms {
		gx := r.x - cur.x + ox
		gy := r.y - cur.y + oy
		if gx < 0 || gx >= cols || gy < 0 || gy >= rows {
			continue
		}
		px := gx * stepX
		py := (rows - 1 - gy) * stepY // invert Y so north is up

		marker := []byte("[ ]")
		if r.vnum == m.current {
			marker = []byte("[*]")
		}
		for i, ch := range marker {
			set(px+i, py, ch)
		}

		for dir := range r.exits {
			switch dir {
			case "n":
				set(px+1, py-1, '|')
			case "s":
				set(px+1, py+1, '|')
			case "e":
				set(px+3, py, '-')
			case "w":
				set(px-1, py, '-')
			}
		}
	}

	var sb strings.Builder
	for i, row := range grid {
		sb.Write(row)
		if i < len(grid)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
