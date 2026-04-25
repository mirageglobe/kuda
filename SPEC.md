# Kuda — Specification & Architecture

> A MUD client built in Go, optimized for Aardwolf.

---

## 1. Overview

**Kuda** is a modern MUD client built in Go, designed for performance and extensibility. Aardwolf [Aardwolf](https://www.aardwolf.com/) is the primary target for testing, but Kuda is designed to support generic MUD protocols.

---

## 2. Technology Stack

| Tool | Purpose |
| :--- | :--- |
| Go | Primary language |
| `bubbletea` | TUI framework |
| `lua` | Scripting engine |

---

## 3. Architecture

```
kuda/
├── network/       # TCP, Telnet, Protocol parsing (GMCP/MCCP)
├── ui/            # TUI (Bubbletea)
├── engine/        # Lua integration & State management
├── mapper/        # Mapping logic
├── main.go        # Entry point
└── ...
```

---

## 4. Development Roadmap

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
