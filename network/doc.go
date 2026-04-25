// Package network owns tcp i/o, telnet protocol negotiation, and gmcp framing.
// It parses the raw byte stream into typed Events and sends them on a channel.
//
// Responsibility boundary:
//   - owns: tcp connection lifecycle, telnet state machine (IAC/DO/WILL/SB/SE), gmcp subnegotiation
//   - does NOT own: game state, ui state, lua scripting, trigger evaluation
//
// Consumers receive events via the Connection interface — do not depend on *Client directly.
package network
