// Package mapper owns the room graph and 2-D ASCII map rendering.
// Room updates are fed by the wiring layer (main.go) via Update(); the mapper
// does not read from the engine or network directly.
//
// Responsibility boundary:
//   - owns: room graph, grid coordinate assignment, map pane rendering
//   - does NOT own: game state, network i/o, lua scripting
package mapper
