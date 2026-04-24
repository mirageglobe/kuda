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
├── src/
│   ├── network/       # TCP, Telnet, Protocol parsing (GMCP/MCCP)
│   ├── ui/            # TUI (Bubbletea)
│   ├── engine/        # Lua integration & State management
│   ├── mapper/        # Mapping logic
│   └── main.go        # Entry point
└── ...
```

---

## 4. Development Roadmap

### Phase 1: Minimal Viable Client (MVC)
- [ ] Basic TCP socket connection to a host/port.
- [ ] Raw stream display in a simple TUI.
- [ ] Basic user command input.

### Phase 2: Protocol Foundation
- [ ] Basic Telnet negotiation (support for standard GA/ECHO).
- [ ] Implement MCCP (Compression) for performance.

### Phase 3: Aardwolf & Advanced Protocols
- [ ] GMCP parsing and state management.
- [ ] MSP support.

### Phase 4: Extensibility
- [ ] Integrate Lua for user-defined triggers/aliases.
- [ ] Basic Mapper implementation for visual room tracking.
