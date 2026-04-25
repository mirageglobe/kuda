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

	// Server side reader to prevent blocking on handleNegotiation writes
	go func() {
		io.Copy(io.Discard, serverConn)
	}()

	client := NewClient()
	client.Conn = clientConn
	go client.listen()

	// Test case 1: Normal text
	serverConn.Write([]byte("hello"))
	
	expected := "hello"
	for _, char := range expected {
		select {
		case ev := <-client.Events:
			if ev.Type != EventText || string(ev.Data) != string(char) {
				t.Errorf("expected %c, got %v", char, ev)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout waiting for %c", char)
		}
	}

	// Test case 2: Stripping IAC WILL ECHO
	serverConn.Write([]byte{IAC, WILL, TelnetOptionEcho, 'w'})
	
	gotNegotiation := false
	gotW := false
	
	for i := 0; i < 2; i++ {
		select {
		case ev := <-client.Events:
			if ev.Type == EventTelnetCommand {
				if ev.Data[0] == WILL && ev.Data[1] == TelnetOptionEcho {
					gotNegotiation = true
				}
			} else if ev.Type == EventText && string(ev.Data) == "w" {
				gotW = true
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("timeout waiting for events, gotNegotiation=%v, gotW=%v", gotNegotiation, gotW)
		}
	}

	if !gotNegotiation {
		t.Error("did not get expected Telnet negotiation event")
	}
	if !gotW {
		t.Error("did not get expected 'w' text")
	}
}

func TestClient_GMCPCapture(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	go func() {
		io.Copy(io.Discard, serverConn)
	}()

	client := NewClient()
	client.Conn = clientConn
	go client.listen()

	// Send GMCP: IAC SB GMCP "core.ping" IAC SE
	gmcpData := []byte("core.ping")
	payload := append([]byte{IAC, SB, TelnetOptionGMCP}, gmcpData...)
	payload = append(payload, IAC, SE)
	
	serverConn.Write(payload)

	select {
	case ev := <-client.Events:
		if ev.Type != EventGMCP {
			t.Errorf("expected EventGMCP, got %v", ev.Type)
		}
		if !bytes.Equal(ev.Data, gmcpData) {
			t.Errorf("expected %s, got %s", string(gmcpData), string(ev.Data))
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for GMCP")
	}
}
