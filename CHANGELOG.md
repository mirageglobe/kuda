# changelog

all notable changes to kuda are documented here.
format follows [keep a changelog](https://keepachangelog.com/en/1.1.0/).
versions follow [semantic versioning](https://semver.org/).

---

## [unreleased]

---

## [v0.2.0] — 2026-05-16

### added
- stats display with session counters
- confirm flows for destructive actions
- ui polish pass

---

## [v0.1.0] — 2026-05-16

### added
- initial project setup for go-based mud client
- bubbletea tui with splash, launch, and session screens
- telnet + gmcp + mccp network layer
- lua scripting engine with alias and trigger support
- aardwolf gmcp parser (vitals, room, char name)
- map panel with ascii room graph and auto-discovery
- `AGENTS.md`, `SPEC.md`, `CHANGELOG.md`, `Makefile`, `Brewfile`
- goreleaser config with homebrew tap support

### changed
- all ctrl+ hotkeys replaced with `/` commands (`/map`, `/map reset`, `/lua`, `/raw`)
- `❯` prompt glyph on all text inputs
- `◈` glyph label above input; shows `[warn]` in red on error
- session status bar: vitals + map/raw/lua toggle indicators right-aligned
- launch screen: rounded border on server list, full width
- help panel: any key dismisses; consolidated keybinding list
