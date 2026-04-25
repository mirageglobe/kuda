package engine

// EventType categorises events the engine receives from the network layer.
type EventType int

const (
	EventText          EventType = iota
	EventGMCP
	EventTelnetCommand
)

// Event is the engine's internal representation of a network event.
// The wiring layer (main.go) adapts network.Event to this type.
type Event struct {
	Type EventType
	Data []byte
}

// EventSource is what engine consumes from the network layer.
// Defined here so engine has no import dependency on network.
type EventSource interface {
	EventsCh() <-chan Event
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
