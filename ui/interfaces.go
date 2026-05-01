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

// ConnStatusInfo holds negotiated protocol state for display purposes.
type ConnStatusInfo struct {
	MCCPActive bool
	GMCPActive bool
	EchoActive bool
}

// Connection is the subset of the network client that ui requires.
// Depend on this interface, not *network.Client directly.
type Connection interface {
	Write(data []byte) error
	EventsCh() <-chan Event
	ErrorsCh() <-chan error
	ConnStatus() ConnStatusInfo
	Close() error
}

// MapView is the rendering interface for the mapper.
// Defined here so ui does not import the mapper package directly.
type MapView interface {
	Render(w, h int) string
	// Reset backs up the current map file and clears the room graph.
	// Returns the backup path on success.
	Reset() (string, error)
}

// Telnet constants for UI interpretation.
const (
	TelnetCmdWILL = 251
	TelnetCmdWONT = 252
	TelnetCmdGA   = 249
	TelnetOptEcho = 1
)
