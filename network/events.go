package network

// EventType defines the type of event received from the network.
type EventType int

const (
	// EventText represents clean display text.
	EventText EventType = iota
	// EventGMCP represents out-of-band GMCP data.
	EventGMCP
	// EventTelnetCommand represents a standard Telnet command (e.g. WILL, DO).
	EventTelnetCommand
)

// Event wraps data received from the MUD server.
type Event struct {
	Type EventType
	Data []byte
}

// Telnet Constants
const (
	IAC  byte = 255
	DONT byte = 254
	DO   byte = 253
	WONT byte = 252
	WILL byte = 251
	SB   byte = 250 // Sub-negotiation Begin
	GA   byte = 249 // Go Ahead
	EL   byte = 248 // Erase Line
	EC   byte = 247 // Erase Character
	AYT  byte = 246 // Are You There
	AO   byte = 245 // Abort Output
	IP   byte = 244 // Interrupt Process
	BRK  byte = 243 // Break
	DM   byte = 242 // Data Mark
	NOP  byte = 241 // No Operation
	SE   byte = 240 // Sub-negotiation End
)

// MUD Protocol Constants
const (
	TelnetOptionEcho   byte = 1
	TelnetOptionMCCP   byte = 86
	TelnetOptionGMCP   byte = 201
	TelnetOptionMXP    byte = 91
	TelnetOptionMSSP   byte = 70
)
