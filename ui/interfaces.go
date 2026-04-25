package ui

import "github.com/mirageglobe/kuda/network"

// Connection is the subset of network.Client that ui requires.
// Depend on this interface, not *network.Client directly.
type Connection interface {
	Write(data []byte) error
	EventsCh() <-chan network.Event
	ErrorsCh() <-chan error
}
