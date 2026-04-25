# Kuda — Specification & Architecture

> A MUD client built in Go, optimized for Aardwolf.

---

## 1. Overview

**Kuda** is a modern MUD client built in Go, designed for performance and extensibility. Aardwolf [Aardwolf](https://www.aardwolf.com/) is the primary target for testing, but Kuda is designed to support generic MUD protocols.

---

## 2. Complexity Score

| Dimension | Score | Notes |
| :--- | :--- | :--- |
| overall | 3 / 5 | moderate; multi-package Go with protocol parsing and Lua scripting |
| network layer | 3 / 5 | telnet state machine, GMCP framing, channel-based i/o |
| engine / lua | 4 / 5 | embedded Lua VM, trigger/alias eval, GMCP-fed state |
| ui / tui | 2 / 5 | standard bubbletea patterns, two simple models |
| mapper | 3 / 5 | room graph traversal, map rendering |

---

## 3. Technology Stack

| Tool | Purpose |
| :--- | :--- |
| Go | Primary language |
| `bubbletea` | TUI framework |
| `lua` | Scripting engine |

---

## 4. Architecture

```
kuda/
├── main.go             # entry point — launches bubbletea program
├── network/            # tcp, telnet, gmcp parsing — no game logic
│   ├── client.go       # tcp connection lifecycle
│   ├── client_test.go
│   ├── telnet.go       # telnet state machine (IAC/DO/WILL/SB/SE)
│   └── events.go       # event types and telnet/gmcp constants
├── engine/             # game state, lua vm, triggers and aliases
│   ├── engine.go       # engine lifecycle
│   ├── state.go        # character/room/world state (fed by gmcp)
│   └── lua.go          # lua vm integration
├── mapper/             # room graph and map rendering
│   └── mapper.go
└── ui/                 # bubbletea tui views
    ├── launch.go       # server selection screen
    ├── client.go       # main connected session view
    └── styles.go       # shared lipgloss styles
```

### Package Responsibilities

| Package | Owns | Does NOT own |
| :--- | :--- | :--- |
| `network` | tcp i/o, telnet state machine, gmcp framing | game state, ui state |
| `engine` | lua vm lifecycle, trigger/alias eval, gmcp-fed state | rendering, network i/o |
| `mapper` | room graph, map rendering | game state (reads from engine) |
| `ui` | bubbletea models, view rendering, input handling | business logic, network calls |

### Interfaces

cross-package communication is enforced via interfaces. concrete types must not be imported across boundaries.

| Interface | Defined in | Implemented by | Used by |
| :--- | :--- | :--- | :--- |
| `ui.Connection` | `ui/interfaces.go` | `clientAdapter` (main.go) | `ui.ClientModel` |
| `engine.EventSource` | `engine/interfaces.go` | adapter (future, main.go) | `engine.Engine` (future) |
| `engine.GameState` | `engine/interfaces.go` | `engine.Engine` (future) | `ui`, `mapper` |

---

## 5. Architecture Decisions

key architectural choices recorded here so they are not accidentally reversed.

### bubbletea model-per-screen
launch and connected session are separate bubbletea models (`LaunchModel`, `ClientModel`), not a single model with a mode flag. reason: each screen has entirely different state and keybindings; merging them creates a god-struct that is hard to test and extend.

### channel-based network events
`network.Client` emits events on a buffered channel rather than using callbacks or a synchronous read loop in `ui`. reason: bubbletea requires commands (functions returning `tea.Msg`) to integrate with its event loop. the channel + `waitForNetworkEvent` cmd pattern is the idiomatic bubbletea approach for external i/o.

### telnet default-refuse negotiation
the telnet state machine replies `WONT`/`DONT` to any option it does not explicitly support. reason: prevents infinite negotiation loops with servers that keep re-offering options. explicit support is added per option as needed.

### engine owns all game state
`ui` and `mapper` are read-only consumers of state via `engine.GameState`. reason: if multiple views can mutate state, consistency bugs are inevitable and hard to trace. a single writer (engine) makes state transitions auditable and testable.

---

## 6. Development Roadmap

### Milestones

| Milestone | Goal | Status |
| :--- | :--- | :--- |
| M1 — minimal viable client | TCP connection, raw stream display, basic input | in progress |
| M2 — protocol foundation | Telnet negotiation (GA/ECHO), MCCP compression | planned |
| M3 — aardwolf protocols | GMCP parsing + state management, MSP support | planned |
| M4 — extensibility | Lua scripting, basic mapper | planned |

---

### M1 — Minimal Viable Client
- [ ] basic TCP socket connection to a host/port.
- [ ] raw stream display in a simple TUI.
- [ ] basic user command input.

### M2 — Protocol Foundation
- [ ] basic Telnet negotiation (support for standard GA/ECHO).
- [ ] implement MCCP (compression) for performance.

### M3 — Aardwolf & Advanced Protocols
- [ ] GMCP parsing and state management.
- [ ] MSP support.

### M4 — Extensibility
- [ ] integrate Lua for user-defined triggers/aliases.
- [ ] basic mapper implementation for visual room tracking.

---

### Near Term
- complete M1 TCP connection and TUI output rendering.
- wire up basic input loop with command history.
- establish project structure for engine, network, ui, and mapper packages.

### Ideas
- scrollback buffer with search.
- split-pane layout (main output + status/map panel).
- ANSI colour support and stripping.
- auto-reconnect with configurable backoff.
- Aardwolf-specific GMCP module handlers (character stats, room info, inventory).
- visual mapper with room graph rendering.
- Lua trigger/alias editor within the TUI.
- plugin system for protocol extensions.
