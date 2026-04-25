package network

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

// Client represents a connection to a MUD server.
type Client struct {
	conn   net.Conn
	events chan Event
	errors chan error
}

// NewClient creates a new network client.
func NewClient() *Client {
	return &Client{
		events: make(chan Event, 100),
		errors: make(chan error, 10),
	}
}

// newClientWithConn creates a client with a pre-existing connection (for tests).
func newClientWithConn(conn net.Conn) *Client {
	c := NewClient()
	c.conn = conn
	return c
}

// Connect establishes a TCP connection to the given address.
func (c *Client) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	c.conn = conn
	go c.listen()
	return nil
}

// Close shuts down the connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// listen reads from the connection and feeds the telnet parser.
func (c *Client) listen() {
	reader := bufio.NewReader(c.conn)
	p := &telnetParser{client: c}
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err != io.EOF {
				c.errors <- err
			}
			close(c.events)
			return
		}
		p.handleByte(b)
	}
}

// EventsCh returns the read-only event channel for use by consumers.
func (c *Client) EventsCh() <-chan Event { return c.events }

// ErrorsCh returns the read-only error channel for use by consumers.
func (c *Client) ErrorsCh() <-chan error { return c.errors }

// Write sends raw bytes to the server.
func (c *Client) Write(data []byte) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	_, err := c.conn.Write(data)
	return err
}
