package mapper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const saveInterval = 10

// savedRoom is the on-disk representation of a room graph node.
type savedRoom struct {
	Vnum  int            `json:"vnum"`
	Name  string         `json:"name"`
	Exits map[string]int `json:"exits,omitempty"`
	X     int            `json:"x"`
	Y     int            `json:"y"`
}

// Load reads a previously saved room graph from path and returns a ready
// Mapper. If the file does not exist or cannot be parsed, an empty Mapper
// is returned — the path is still stored for future Save calls.
func Load(path string) *Mapper {
	m := &Mapper{rooms: make(map[int]*room), savePath: path}

	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	var saved []savedRoom
	if err := json.Unmarshal(data, &saved); err != nil {
		return m
	}
	for _, sr := range saved {
		m.rooms[sr.Vnum] = &room{
			vnum: sr.Vnum, name: sr.Name,
			exits: sr.Exits, x: sr.X, y: sr.Y,
		}
	}
	return m
}

// Save writes the current room graph to the path set at Load time.
func (m *Mapper) Save() error {
	if m.savePath == "" {
		return nil
	}

	m.mu.RLock()
	saved := make([]savedRoom, 0, len(m.rooms))
	for _, r := range m.rooms {
		saved = append(saved, savedRoom{
			Vnum: r.vnum, Name: r.name,
			Exits: r.exits, X: r.x, Y: r.y,
		})
	}
	m.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(m.savePath), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(saved)
	if err != nil {
		return err
	}
	return os.WriteFile(m.savePath, data, 0o644)
}

// Reset backs up the current save file with a timestamp suffix and clears the
// room graph. Returns the backup path. Safe to call with no save file present.
func (m *Mapper) Reset() (string, error) {
	backupPath := ""
	if m.savePath != "" {
		backupPath = fmt.Sprintf("%s.%s.bak", m.savePath, time.Now().Format("20060102-150405"))
		if data, err := os.ReadFile(m.savePath); err == nil {
			if err := os.WriteFile(backupPath, data, 0o644); err != nil {
				return "", err
			}
		}
	}

	m.mu.Lock()
	m.rooms = make(map[int]*room)
	m.current = 0
	m.updateCount = 0
	m.mu.Unlock()

	if m.savePath != "" {
		if err := os.MkdirAll(filepath.Dir(m.savePath), 0o755); err != nil {
			return backupPath, err
		}
		if err := os.WriteFile(m.savePath, []byte("[]"), 0o644); err != nil {
			return backupPath, err
		}
	}
	return backupPath, nil
}
