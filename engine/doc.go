// Package engine owns game state, the lua scripting vm, and trigger/alias evaluation.
// It consumes network.Events (via EventSource) and exposes parsed state to ui and mapper
// through the GameState interface.
//
// Responsibility boundary:
//   - owns: character/room/world state (fed by gmcp), lua vm lifecycle, trigger and alias matching
//   - does NOT own: tcp i/o, telnet parsing, rendering, bubbletea models
//
// State mutations happen only here. ui and mapper read state; they never write it.
package engine
