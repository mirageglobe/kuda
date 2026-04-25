// Package ui owns all bubbletea models, view rendering, and user input handling.
// It talks to network via the Connection interface and to engine via engine.GameState.
//
// Responsibility boundary:
//   - owns: bubbletea model lifecycle, view layout, input dispatch
//   - does NOT own: game state, network i/o, lua scripting, trigger evaluation
//
// Views:
//   - LaunchModel  — server selection screen (launch.go)
//   - ClientModel  — main connected session: output viewport + input bar (client.go)
package ui
