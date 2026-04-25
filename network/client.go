package network

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

type parserState int

const (
	stateText parserState = iota
	stateIAC
	stateDO
	stateDONT
	stateWILL
	stateWONT
	stateSB
	stateSBIAC
)

// Client represents a connection to a MUD server.
type Client struct {
	Conn   net.Conn
	Events chan Event
	Errors chan error

	state parserState
	sbBuf []byte
}

// NewClient creates a new network client.
func NewClient() *Client {
	return &Client{
		Events: make(chan Event, 100),
		Errors: make(chan error, 10),
	}
}

// Connect establishes a TCP connection to the given address.
func (c *Client) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	c.Conn = conn
	go c.listen()
	return nil
}

// Close shuts down the connection.
func (c *Client) Close() error {
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// listen reads from the connection and processes the Telnet protocol.
func (c *Client) listen() {
	reader := bufio.NewReader(c.Conn)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err != io.EOF {
				c.Errors <- err
			}
			close(c.Events)
			return
		}
		c.handleByte(b)
	}
}

func (c *Client) handleByte(b byte) {
	switch c.state {
	case stateText:
		if b == IAC {
			c.state = stateIAC
		} else {
			c.Events <- Event{Type: EventText, Data: []byte{b}}
		}

	case stateIAC:
		switch b {
		case IAC:
			c.Events <- Event{Type: EventText, Data: []byte{IAC}}
			c.state = stateText
		case DO:
			c.state = stateDO
		case DONT:
			c.state = stateDONT
		case WILL:
			c.state = stateWILL
		case WONT:
			c.state = stateWONT
		case SB:
			c.state = stateSB
			c.sbBuf = nil
		default:
			c.Events <- Event{Type: EventTelnetCommand, Data: []byte{b}}
			c.state = stateText
		}

	case stateDO:
		c.handleNegotiation(DO, b)
		c.state = stateText

	case stateDONT:
		c.handleNegotiation(DONT, b)
		c.state = stateText

	case stateWILL:
		c.handleNegotiation(WILL, b)
		c.state = stateText

	case stateWONT:
		c.handleNegotiation(WONT, b)
		c.state = stateText

	case stateSB:
		if b == IAC {
			c.state = stateSBIAC
		} else {
			c.sbBuf = append(c.sbBuf, b)
		}

	case stateSBIAC:
		if b == SE {
			c.handleSubnegotiation(c.sbBuf)
			c.state = stateText
		} else if b == IAC {
			c.sbBuf = append(c.sbBuf, IAC)
			c.state = stateSB
		} else {
			// Invalid state, but let's try to recover
			c.sbBuf = append(c.sbBuf, IAC, b)
			c.state = stateSB
		}
	}
}

func (c *Client) handleNegotiation(cmd, option byte) {
	// For now, we WONT/DONT everything we don't explicitly support.
	// This prevents infinite negotiation loops.
	fmt.Printf("NEGOTIATION: cmd=%d option=%d\n", cmd, option)
	switch cmd {
	case DO:
		c.Write([]byte{IAC, WONT, option})
	case WILL:
		c.Write([]byte{IAC, DONT, option})
	}
	c.Events <- Event{Type: EventTelnetCommand, Data: []byte{cmd, option}}
}

func (c *Client) handleSubnegotiation(data []byte) {
	if len(data) > 0 && data[0] == TelnetOptionGMCP {
		c.Events <- Event{Type: EventGMCP, Data: data[1:]}
	}
}

// EventsCh returns the read-only event channel for use by consumers.
func (c *Client) EventsCh() <-chan Event { return c.Events }

// ErrorsCh returns the read-only error channel for use by consumers.
func (c *Client) ErrorsCh() <-chan error { return c.Errors }

// Write sends raw bytes to the server.
func (c *Client) Write(data []byte) error {
	if c.Conn == nil {
		return fmt.Errorf("not connected")
	}
	_, err := c.Conn.Write(data)
	return err
}
