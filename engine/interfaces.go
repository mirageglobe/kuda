package engine

import "github.com/mirageglobe/kuda/network"

// EventSource is the subset of network.Client that engine consumes.
// Engine reads events from network through this interface, not *network.Client directly.
type EventSource interface {
	EventsCh() <-chan network.Event
	ErrorsCh() <-chan error
}

// GameState exposes read-only game state to ui and mapper.
// All mutations happen inside engine; consumers only read through this interface.
type GameState interface {
	// Room returns the current room vnum and name.
	Room() (vnum int, name string)
	// CharName returns the player character name.
	CharName() string
}
