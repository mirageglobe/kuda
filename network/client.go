package network

import (
	"bufio"
	"compress/zlib"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

// RetryConfig controls reconnect behaviour after a dropped connection.
// Set MaxAttempts to 0 to disable auto-reconnect.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// DefaultRetry is a sensible reconnect policy for interactive MUD sessions.
var DefaultRetry = RetryConfig{
	MaxAttempts:  5,
	InitialDelay: 2 * time.Second,
	MaxDelay:     30 * time.Second,
}

// Client represents a connection to a MUD server.
type Client struct {
	conn        net.Conn
	events      chan Event
	errors      chan error
	pendingMCCP bool
	stopped     atomic.Bool
	mccpActive  atomic.Bool
	gmcpActive  atomic.Bool
	echoActive  atomic.Bool
	address     string
	retry       RetryConfig
}

// IsMCCPActive reports whether MCCP (zlib compression) is active on this connection.
func (c *Client) IsMCCPActive() bool { return c.mccpActive.Load() }

// IsGMCPActive reports whether GMCP was negotiated with the server.
func (c *Client) IsGMCPActive() bool { return c.gmcpActive.Load() }

// IsEchoActive reports whether the server has taken over echo (e.g. password input).
func (c *Client) IsEchoActive() bool { return c.echoActive.Load() }

// NewClient creates a new network client with default retry settings.
func NewClient() *Client {
	return &Client{
		events: make(chan Event, 1024),
		errors: make(chan error, 10),
		retry:  DefaultRetry,
	}
}

// newClientWithConn creates a client with a pre-existing connection (for tests).
func newClientWithConn(conn net.Conn) *Client {
	c := NewClient()
	c.conn = conn
	c.retry = RetryConfig{} // disable retry for injected test connections
	return c
}

// Connect establishes a TCP connection to the given address and starts the read loop.
// Supports unencrypted connections and TLS if address starts with "tls://".
func (c *Client) Connect(address string) error {
	c.address = address
	if err := c.dial(); err != nil {
		return err
	}
	go c.listen()
	return nil
}

// Close shuts down the connection and prevents auto-reconnect.
func (c *Client) Close() error {
	c.stopped.Store(true)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// dial opens (or reopens) the underlying TCP or TLS connection.
func (c *Client) dial() error {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var (
		conn net.Conn
		err  error
	)
	if strings.HasPrefix(c.address, "tls://") {
		cleanAddr := strings.TrimPrefix(c.address, "tls://")
		conn, err = tls.DialWithDialer(dialer, "tcp", cleanAddr, &tls.Config{
			InsecureSkipVerify: true, // MUDs commonly use self-signed certs
		})
	} else {
		conn, err = dialer.Dial("tcp", c.address)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.address, err)
	}
	c.conn = conn
	return nil
}

// sendText injects a local status line into the events channel without blocking.
func (c *Client) sendText(text string) {
	select {
	case c.events <- Event{Type: EventText, Data: []byte(text)}:
	default:
	}
}

// backoffDelay returns the exponential backoff delay for a given attempt index (0-based).
func (c *Client) backoffDelay(attempt int) time.Duration {
	d := float64(c.retry.InitialDelay) * math.Pow(2, float64(attempt))
	if d > float64(c.retry.MaxDelay) {
		d = float64(c.retry.MaxDelay)
	}
	return time.Duration(d)
}

// listen runs the read loop and attempts reconnection on disconnect per retry config.
func (c *Client) listen() {
	for {
		if err := c.readLoop(); err != nil {
			c.errors <- err
		}

		if c.retry.MaxAttempts == 0 || c.stopped.Load() {
			break
		}

		reconnected := false
		for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
			delay := c.backoffDelay(attempt - 1)
			c.sendText(fmt.Sprintf("\r\n[ connection lost — reconnecting in %v (attempt %d/%d) ]\r\n",
				delay, attempt, c.retry.MaxAttempts))
			time.Sleep(delay)
			if err := c.dial(); err != nil {
				c.sendText(fmt.Sprintf("[ reconnect failed: %v ]\r\n", err))
				continue
			}
			c.sendText("[ reconnected ]\r\n")
			reconnected = true
			break
		}
		if !reconnected {
			c.errors <- fmt.Errorf("connection lost after %d reconnect attempts", c.retry.MaxAttempts)
			break
		}
	}
	close(c.events)
}

// readLoop runs the telnet byte loop until the connection drops.
// Returns non-nil only for unexpected (non-EOF) errors.
func (c *Client) readLoop() error {
	reader := bufio.NewReader(c.conn)
	p := &telnetParser{client: c}
	for {
		b, err := reader.ReadByte()
		if err != nil {
			p.flushText()
			if err == io.EOF {
				return nil
			}
			return err
		}
		p.handleByte(b)
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
