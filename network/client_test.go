package network

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func TestClient_TelnetParsing(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// drain server side so handleNegotiation writes don't block
	go func() { io.Copy(io.Discard, serverConn) }() //nolint:errcheck

	client := newClientWithConn(clientConn)
	go client.listen()

	events := client.EventsCh()

	// case 1: normal text
	serverConn.Write([]byte("hello")) //nolint:errcheck
	for _, char := range "hello" {
		select {
		case ev := <-events:
			if ev.Type != EventText || string(ev.Data) != string(char) {
				t.Errorf("expected %c, got %v", char, ev)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout waiting for %c", char)
		}
	}

	// case 2: IAC WILL ECHO stripped to negotiation event + following text
	serverConn.Write([]byte{IAC, WILL, TelnetOptionEcho, 'w'}) //nolint:errcheck

	gotNegotiation, gotW := false, false
	for i := 0; i < 2; i++ {
		select {
		case ev := <-events:
			switch {
			case ev.Type == EventTelnetCommand && ev.Data[0] == WILL && ev.Data[1] == TelnetOptionEcho:
				gotNegotiation = true
			case ev.Type == EventText && string(ev.Data) == "w":
				gotW = true
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("timeout: gotNegotiation=%v gotW=%v", gotNegotiation, gotW)
		}
	}
	if !gotNegotiation {
		t.Error("did not receive telnet negotiation event")
	}
	if !gotW {
		t.Error("did not receive 'w' text event")
	}

	// case 3: GA command
	serverConn.Write([]byte{IAC, GA}) //nolint:errcheck
	select {
	case ev := <-events:
		if ev.Type != EventTelnetCommand || ev.Data[0] != GA {
			t.Errorf("expected GA command, got %v", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for GA command")
	}
}

func TestClient_GMCPCapture(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	go func() { io.Copy(io.Discard, serverConn) }() //nolint:errcheck

	client := newClientWithConn(clientConn)
	go client.listen()

	// IAC SB GMCP <data> IAC SE
	gmcpData := []byte("core.ping")
	payload := append([]byte{IAC, SB, TelnetOptionGMCP}, gmcpData...)
	payload = append(payload, IAC, SE)
	serverConn.Write(payload) //nolint:errcheck

	select {
	case ev := <-client.EventsCh():
		if ev.Type != EventGMCP {
			t.Errorf("expected EventGMCP, got %v", ev.Type)
		}
		if !bytes.Equal(ev.Data, gmcpData) {
			t.Errorf("expected %q, got %q", gmcpData, ev.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for GMCP event")
	}
}
