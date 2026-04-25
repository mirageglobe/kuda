package engine

// EventType categorises events the engine receives from the network layer.
type EventType int

const (
	EventText EventType = iota
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
	Write(data []byte) error
}

// GameState exposes read-only game state and command execution to ui and mapper.
// All mutations happen inside engine; consumers only read through this interface.
type GameState interface {
	// Execute processes a user command through the script engine.
	Execute(cmd string) error
	// ToggleLua enables or disables the script engine. Returns the new state.
	ToggleLua() bool
	// Room returns current room details.
	Room() RoomInfo
	// Vitals returns current player character vitals.
	Vitals() VitalsInfo
	// CharName returns the player character name.
	CharName() string
}

type VitalsInfo struct {
	HP, MaxHP     int
	Mana, MaxMana int
	Move, MaxMove int
}

type RoomInfo struct {
	Vnum int
	Name string
}
