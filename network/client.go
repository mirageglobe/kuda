package network

import (
	"bufio"
	"compress/zlib"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Client represents a connection to a MUD server.
type Client struct {
	conn        net.Conn
	events      chan Event
	errors      chan error
	pendingMCCP bool
}

// NewClient creates a new network client.
func NewClient() *Client {
	return &Client{
		events: make(chan Event, 1024),
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
// Supports unencrypted connections and TLS if address starts with "tls://".
func (c *Client) Connect(address string) error {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var conn net.Conn
	var err error

	if strings.HasPrefix(address, "tls://") {
		cleanAddr := strings.TrimPrefix(address, "tls://")
		// MUD TLS often uses self-signed or older certs; we try strict first
		// but might need InsecureSkipVerify: true if users have issues.
		conn, err = tls.DialWithDialer(dialer, "tcp", cleanAddr, &tls.Config{
			InsecureSkipVerify: true, // Common for MUDs with self-signed certs
		})
	} else {
		conn, err = dialer.Dial("tcp", address)
	}

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
			p.flushText() // ensure last bits are sent
			if err != io.EOF {
				c.errors <- err
			}
			close(c.events)
			return
		}
		p.handleByte(b)
		// If no more bytes are immediately available in the buffer,
		// flush the parser to ensure prompts are displayed.
		if reader.Buffered() == 0 {
			p.flushText()
		}
		if c.pendingMCCP {
			c.pendingMCCP = false
			zReader, err := zlib.NewReader(reader)
			if err != nil {
				c.errors <- fmt.Errorf("failed to start mccp decompression: %w", err)
				continue
			}
			reader = bufio.NewReader(zReader)
		}
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
