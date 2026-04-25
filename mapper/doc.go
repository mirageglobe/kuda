// Package mapper owns the room graph and its tui rendering.
// It reads room state from engine.GameState; it does not hold state of its own.
//
// Responsibility boundary:
//   - owns: room graph construction, pathfinding, map pane rendering
//   - does NOT own: game state (read-only via engine.GameState), network i/o, lua scripting
package mapper
