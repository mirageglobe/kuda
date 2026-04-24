# Kuda — Specification & Architecture

> A MUD client built in Go, optimized for Aardwolf.

---

## 1. Overview

**Kuda** is a modern MUD client built in Go, designed to provide a performant, extensible, and user-friendly interface for connecting to MUDs. While Aardwolf [Aardwolf](https://www.aardwolf.com/) is the primary target for development and testing, Kuda aims to provide generic support for other MUDs as well.

### Goals

| Goal | Status |
| :--- | :--- |
| Core TCP/Telnet connectivity | [ ] |
| Terminal User Interface (TUI) | [ ] |
| Lua Scripting engine | [ ] |
| Aardwolf protocol support (GMCP/MSP/MCCP) | [ ] |
| Mapper system | [ ] |

---

## 2. Technology Stack

| Tool | Purpose |
| :--- | :--- |
| Go | Primary language |
| `bubbletea` | TUI framework |
| `gdam` | Terminal handling |
| `lua` | Scripting engine |

---

## 3. Architecture

```
kuda/
├── src/
│   ├── client/        # Network handling, connection management
│   ├── ui/            # Bubbletea components, layout
│   ├── scripts/       # Lua integration
│   ├── mapper/        # Mapping logic
│   └── main.go        # Entry point
├── docs/              # Documentation
├── Makefile           # Build and test orchestration
└── README.md          # Project overview
```

### 3.1 Data Flow

```
Socket ──► Connection Handler ──► Protocol Parser (GMCP/Telnet/MCCP) ──► UI State ──► TUI
```

---

## 4. Development Roadmap

### Phase 1: Core
- [ ] Implement TCP connection handler.
- [ ] Implement basic Telnet/GA protocol handling.
- [ ] Basic TUI layout (input field, output window).

### Phase 2: Aardwolf Integration
- [ ] Implement GMCP (Generic MUD Communication Protocol) support.
- [ ] Implement support for MUD Sound Protocol (MSP) / triggers.
- [ ] Implement MCCP (Mud Client Compression Protocol) support.

### Phase 3: Extensibility & Features
- [ ] Integrate Lua scripting engine for triggers/aliases/macros.
- [ ] Implement Mapper system for real-time room tracking and visualization.
