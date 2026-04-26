# Kuda — Specification & Architecture

> A MUD client built in Go, optimized for Aardwolf.

---

## 1. Overview

**Kuda** is a modern MUD client built in Go, designed for performance and extensibility. Aardwolf [Aardwolf](https://www.aardwolf.com/) is the primary target for testing, but Kuda is designed to support generic MUD protocols.

---

## 2. Complexity Score

| Dimension     | Score | Notes                                                          |
| :------------ | :---- | :------------------------------------------------------------- |
| overall       | 3 / 5 | moderate; multi-package Go with protocol parsing and Lua scripting |
| network layer | 3 / 5 | telnet state machine, GMCP framing, channel-based i/o          |
| engine / lua  | 4 / 5 | embedded Lua VM, trigger/alias eval, GMCP-fed state            |
| ui / tui      | 2 / 5 | standard bubbletea patterns, two simple models                 |
| mapper        | 3 / 5 | room graph traversal, map rendering                            |

---

## 3. Technology Stack

| Tool         | Purpose          |
| :----------- | :--------------- |
| Go           | Primary language |
| `bubbletea`  | TUI framework    |
| `lua`        | Scripting engine |

---

## 4. Architecture

```
kuda/
├── main.go             # entry point — launches bubbletea program
├── network/            # tcp, telnet, gmcp parsing — no game logic
│   ├── doc.go          # package ownership declaration
│   ├── client.go       # tcp/tls connection lifecycle
│   ├── client_test.go
│   ├── telnet.go       # telnet state machine (IAC/DO/WILL/SB/SE)
│   └── events.go       # event types and telnet/gmcp constants
├── engine/             # game state, lua vm, triggers and aliases
│   ├── doc.go          # package ownership declaration
│   ├── engine.go       # engine lifecycle
│   ├── interfaces.go   # EventSource and GameState interfaces
│   ├── state.go        # character/room/world state (fed by gmcp)
│   └── lua.go          # lua vm integration
├── mapper/             # room graph and map rendering
│   ├── doc.go          # package ownership declaration
│   └── mapper.go
└── ui/                 # bubbletea tui views
    ├── doc.go          # package ownership declaration
    ├── interfaces.go   # Connection interface consumed by ui
    ├── provider.go     # server list and connection config
    ├── splash.go       # splash/intro screen model
    ├── launch.go       # server selection screen
    ├── client.go       # main connected session view
    └── styles.go       # shared lipgloss styles
```

### Package Responsibilities

| Package    | Owns                                              | Does NOT own                   |
| :--------- | :------------------------------------------------ | :----------------------------- |
| `network`  | tcp i/o, telnet state machine, gmcp framing       | game state, ui state           |
| `engine`   | lua vm lifecycle, trigger/alias eval, gmcp-fed state | rendering, network i/o      |
| `mapper`   | room graph, map rendering                         | game state (reads from engine) |
| `ui`       | bubbletea models, view rendering, input handling  | business logic, network calls  |

### Interfaces

cross-package communication is enforced via interfaces. concrete types must not be imported across boundaries.

| Interface            | Defined in              | Implemented by              | Used by                   |
| :------------------- | :---------------------- | :-------------------------- | :------------------------ |
| `ui.Connection`      | `ui/interfaces.go`      | `clientAdapter` (main.go)   | `ui.ClientModel`          |
| `engine.EventSource` | `engine/interfaces.go`  | adapter (future, main.go)   | `engine.Engine` (future)  |
| `engine.GameState`   | `engine/interfaces.go`  | `engine.Engine` (future)    | `ui`, `mapper`            |

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

### feature detection over mud-specific drivers
kuda uses a common telnet state machine that negotiates capabilities (GMCP, MCCP, TTYPE) rather than using hardcoded "drivers" for different MUDs. reason: the Telnet RFC is designed for feature negotiation; sticking to this allows kuda to be universal while still supporting the advanced features of specific servers. mud-specific logic is handled by reacting to the *protocols* detected (e.g. enabling a mapper when GMCP room data is received).

---

## 6. Development Roadmap

### Milestones

| Milestone                  | Goal                                             | Status      |
| :------------------------- | :----------------------------------------------- | :---------- |
| M1 — minimal viable client | TCP connection, raw stream display, basic input  | complete    |
| M2 — protocol foundation   | Telnet negotiation (GA/ECHO), MCCP compression   | complete    |
| M3 — aardwolf protocols    | GMCP parsing + state management, MSP support     | complete    |
| M4 — extensibility         | Lua scripting, basic mapper                      | in progress |

---

### M1 — Minimal Viable Client
- [x] basic TCP socket connection to a host/port.
- [x] raw stream display in a simple TUI.
- [x] basic user command input.
- [x] ANSI colour support and stripping.

### M2 — Protocol Foundation
- [x] basic Telnet negotiation (support for standard GA/ECHO).
- [x] implement MCCP (compression) for performance.
- [x] TLS support for secure connections (generic MUDs via `tls://` prefix; Aardwolf does not offer TLS).

### M3 — Aardwolf & Advanced Protocols
- [x] GMCP parsing and state management.

### M4 — Extensibility
- [/] integrate Lua for user-defined triggers/aliases (VM embedded, basic API).
- [x] basic mapper implementation for visual room tracking.

---

### Near Term
- [x] `[ui]` toggle raw mode — print all received bytes unprocessed for debugging [easy]
- [x] `[mapper]` persist room graph to disk with periodic auto-save and load on startup [easy]
- [ ] `[network]` implement auto-reconnect with configurable backoff [easy]
- [ ] `[network]` add Aardwolf-specific GMCP module handlers (stats, room, inventory) [medium]
- [ ] `[network]` add MSP (MUD Sound Protocol) support [medium]
- [ ] `[build]` setup GoReleaser for automated versioning and Homebrew deployment [medium]
- [ ] `[engine/ui]` add support for user-defined hotkeys/aliases via Lua [medium]
- [ ] `[ui]` implement layered input manager to handle keybinding/command conflicts [medium]
- [ ] `[ui]` add search feature to scrollback buffer [medium]
- [ ] `[network/engine/ui]` implement virtual scrollback buffer for performance [hard]
- [ ] `[engine/ui]` implement regex-based line diversion (combat/spells/chat) [medium]
- [ ] `[engine/ui]` add event-driven spell/buff dashboard [medium]
- [ ] `[network/engine]` improve speedwalks and speedrun capabilities [medium]
- [ ] `[ui]` add settings/connection status display (MCCP, telnet details) [easy]
- [ ] `[engine]` add toggle for filtering Aardwolf-specific text tags (e.g., {say}, {affoff}) [easy]
- [ ] `[ui]` investigate dedicated UI panels for GMCP-backed chat and equipment [medium]
- [ ] `[ui]` add Aardwolf in-game time and date ticker to top bar [medium]
- [ ] `[ui]` add system info (date, time, CPU/memory) to top bar [easy]
- [ ] `[ui]` add top bar with version, project name, and GitHub link [easy]
- [ ] `[ui]` add theming support and theme switcher [medium]
- [ ] `[ui]` improve UI with icons and visual symbols [easy]
- [ ] `[engine/ui]` add toggle for showing and editing lua based aliases [medium]
- [x] `[ui]` toggle map mode (new screen or overlay) [medium]
- [ ] `[ui]` allow quitting mud session to return to the main screen [easy]
- [x] `[ui]` improve display with border panes [medium]
- [ ] `[ui]` have a help menu hotkey "?" [easy]
- [x] `[ui]` navigation up and down should go back in history of commands [easy]
- [ ] `[network]` allow aardwolf to quit game cleanly [easy]
- [ ] complete M1 TCP connection and TUI output rendering. [easy]
- [ ] wire up basic input loop with command history. [easy]
- [ ] establish project structure for engine, network, ui, and mapper packages. [medium]

### Ideas
- split-pane layout (main output + status/map panel).
- visual mapper with room graph rendering.
- plugin system for protocol extensions.
