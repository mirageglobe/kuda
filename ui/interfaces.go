package ui

// EventType categorises events the ui receives from the network layer.
type EventType int

const (
	EventText EventType = iota
	EventGMCP
	EventTelnetCommand
)

// Event is ui's view of a network event.
// The adapter in main.go converts network.Event to this type.
type Event struct {
	Type EventType
	Data []byte
}

// Connection is the subset of the network client that ui requires.
// Depend on this interface, not *network.Client directly.
type Connection interface {
	Write(data []byte) error
	EventsCh() <-chan Event
	ErrorsCh() <-chan error
}

// MapView is the rendering interface for the mapper.
// Defined here so ui does not import the mapper package directly.
type MapView interface {
	Render(w, h int) string
}

// Telnet constants for UI interpretation.
const (
	TelnetCmdWILL = 251
	TelnetCmdWONT = 252
	TelnetCmdGA   = 249
	TelnetOptEcho = 1
)
