# Kuda — Specification & Architecture

> A general MUD client built in Go, optimized for Aardwolf.

---

## 1. overview

**Kuda** is a modern MUD client built in Go, designed for performance and extensibility. Aardwolf [Aardwolf](https://www.aardwolf.com/) is the primary target for testing, but Kuda is designed to support generic MUD protocols.

---

## 2. complexity score

| Dimension     | Score | Notes                                                          |
| :------------ | :---- | :------------------------------------------------------------- |
| overall       | 3 / 5 | moderate; multi-package Go with protocol parsing and Lua scripting |
| network layer | 3 / 5 | telnet state machine, GMCP framing, channel-based i/o          |
| engine / lua  | 4 / 5 | embedded Lua VM, trigger/alias eval, GMCP-fed state            |
| ui / tui      | 3 / 5 | mapper panel, raw mode, scrollback, help overlay, cmd history  |
| mapper        | 3 / 5 | room graph traversal, map rendering                            |

---

## 3. technology stack

| Tool         | Purpose          |
| :----------- | :--------------- |
| Go           | Primary language |
| `bubbletea`  | TUI framework    |
| `lua`        | Scripting engine |

---

## 4. architecture

```
kuda/
├── main.go             # entry point — rootModel and bubbletea program setup
├── adapter.go          # wiring layer — clientAdapter, engineAdapter, mockConnection
├── network/            # tcp, telnet, gmcp parsing — no game logic
│   ├── doc.go          # package ownership declaration
│   ├── client.go       # tcp/tls connection lifecycle
│   ├── client_test.go
│   ├── telnet.go       # telnet state machine (IAC/DO/WILL/SB/SE)
│   └── events.go       # event types and telnet/gmcp constants
├── engine/             # game state, lua vm, triggers and aliases
│   ├── doc.go          # package ownership declaration
│   ├── engine.go       # engine lifecycle and GameState implementation
│   ├── engine_test.go
│   ├── gmcp.go         # gmcp message dispatch and state mutations
│   ├── gmcp_test.go
│   ├── interfaces.go   # EventSource and GameState interfaces
│   ├── lua.go          # lua vm integration (ScriptEngine)
│   ├── state.go        # character/room/world state (fed by gmcp)
│   └── state_test.go
├── mapper/             # room graph and map rendering
│   ├── doc.go          # package ownership declaration
│   ├── mapper.go
│   └── persist.go      # json save/load and backup/reset
└── ui/                 # bubbletea tui views
    ├── doc.go          # package ownership declaration
    ├── interfaces.go   # Connection interface consumed by ui
    ├── provider.go     # server list and connection config
    ├── splash.go       # splash/intro screen model
    ├── launch.go       # server selection screen
    ├── client.go       # ClientModel struct, Update, and event handling
    ├── view.go         # ClientModel View, helpView, and render helpers
    └── styles.go       # shared lipgloss styles
```

### package responsibilities

| package    | owns                                              | does not own                   |
| :--------- | :------------------------------------------------ | :----------------------------- |
| `network`  | tcp i/o, telnet state machine, gmcp framing       | game state, ui state           |
| `engine`   | lua vm lifecycle, trigger/alias eval, gmcp-fed state | rendering, network i/o      |
| `mapper`   | room graph, map rendering                         | game state (reads from engine) |
| `ui`       | bubbletea models, view rendering, input handling  | business logic, network calls  |

### interfaces

cross-package communication is enforced via interfaces. concrete types must not be imported across boundaries.

| Interface            | Defined in              | Implemented by              | Used by                   |
| :------------------- | :---------------------- | :-------------------------- | :------------------------ |
| `ui.Connection`      | `ui/interfaces.go`      | `clientAdapter` (adapter.go)  | `ui.ClientModel`          |
| `engine.EventSource` | `engine/interfaces.go`  | `engineAdapter` (adapter.go)  | `engine.Engine`           |
| `engine.GameState`   | `engine/interfaces.go`  | `engine.Engine`               | `ui`, `mapper`            |

---

## 5. architecture decisions

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

### adapter.go as explicit wiring layer
interface adapters (`clientAdapter`, `engineAdapter`, `mockConnection`) live in `adapter.go` rather than `main.go`. reason: `main.go` should only own program startup and model transitions; mixing adapter logic there made the file 300+ lines and obscured the separation between wiring and entry point. adapter.go is the one file allowed to import all packages simultaneously.

### tls opt-in via functional option
`InsecureSkipVerify` is disabled by default; callers must pass `network.WithInsecureTLS()` explicitly. reason: unconditional certificate bypass is a security smell even for MUD servers; opt-in makes the risk visible at the call site.

### aardwolf GMCP Room.Info conventions
Aardwolf embeds exits inline inside `Room.Info` JSON (`"exits": {"n": 12345, ...}`) rather than sending a separate `Room.Exits` message. `gmcp.go` parses exits from `Room.Info` directly. A separate `Room.Exits` message is also handled for servers that send it independently.

Aardwolf coordinate axes are non-standard: `coords.x` = north-south, `coords.y` = east-west. `gmcp.go` swaps them on ingestion so the mapper's internal convention is always `X=EW, Y=NS` (standard cartesian). Direction keys in exits are abbreviated at the GMCP boundary (`"north"` → `"n"`) so the mapper's `dirDelta` table matches without special-casing.

### aardwolf character conventions

| Character | Role | Context |
| :--- | :--- | :--- |
| `@` | **Protected** | color codes (e.g. `@R` for red), must use `@@` for literal |
| `#` | Common | client-side speedwalking and client commands (MUSHclient) |
| `/` | Common | socials and client-side command prefixes |
| `!` | Standard | repeat last command |
| `.` | Standard | item indexing (e.g. `get 2.sword`) |
| `{` | System | used for Aardwolf-specific state tags (e.g. `{say}`) |
| `~` | Special | used in some internal string delimiters |

---

## 6. build & release

### local development

```bash
make build       # compile binary to bin/kuda
make test        # run tests and linter
make run         # build and launch
make release     # local snapshot build via goreleaser (requires: brew install goreleaser)
```

### publishing a release

releases are automated via goreleaser and GitHub Actions (`.github/workflows/release.yml`). the workflow triggers on any `v*` tag push. the homebrew tap is updated manually after each release.

**one-time setup — before first release:**

1. create the homebrew tap repository at `github.com/mirageglobe/homebrew-tap` with a `Formula/` directory.
2. generate a GitHub PAT with `repo` write scope for the tap repo.
3. add the PAT as a repository secret named `HOMEBREW_TAP_GITHUB_TOKEN` in the kuda repo settings (Settings → Secrets → Actions).

### prerequisites

- `GITHUB_TOKEN` available in repo secrets (GitHub provides this automatically for Actions)

### version bump guide

| change type                                      | bump    | example         |
| :----------------------------------------------- | :------ | :-------------- |
| bug fixes only                                   | patch   | v0.3.0 → v0.3.1 |
| new user-facing features, no breaking changes    | minor   | v0.3.0 → v0.4.0 |
| breaking changes to behaviour or config format   | major   | v0.3.0 → v1.0.0 |

### steps

#### phase 1 — prepare changelog (on feature branch)

```bash
# 1. decide the target version using the bump guide above (e.g. v0.5.0)
#    decide before editing — the version determines the changelog heading

# 2. update CHANGELOG.md — move [unreleased] items under the new version heading
#    e.g. ## [v0.5.0] — 2026-05-01
#    add a fresh empty [unreleased] section at the top for the next cycle

# 3. commit and push the changelog update
git add CHANGELOG.md && git commit -m "docs: finalize changelog for vX.Y.Z" && git push

# 4. open a PR and merge into main
```

#### phase 2 — tag and publish (on main)

```bash
# 5. sync local main after merge
git checkout main && git pull

# 6. tag the next version — pick one based on the bump guide above
make bump-patch   # bug fixes only         e.g. v0.3.0 -> v0.3.1
make bump-minor   # new features           e.g. v0.3.0 -> v0.4.0
make bump-major   # breaking changes       e.g. v0.3.0 -> v1.0.0

# 7. push the tag to origin — triggers CI to build and publish the GitHub release
make push-tags
```

this triggers the workflow which:
- cross-compiles for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- creates a GitHub release with archives and `checksums.txt`

#### phase 3 — update homebrew tap (after CI completes)

```bash
# 8. download release assets and compute sha256
gh release download vX.Y.Z --repo mirageglobe/kuda --dir /tmp/kuda-vX.Y.Z --clobber
shasum -a 256 /tmp/kuda-vX.Y.Z/*

# 9. update homebrew-tap/Formula/kuda.rb with new version, urls, and sha256 values

# 10. commit and push the tap update
git add Formula/kuda.rb && git commit -m "feat: update kuda to vX.Y.Z" && git push
```

### local validation (optional)

```bash
make release-dry   # dry-run via goreleaser: builds binaries and archives locally, no publish
```

### troubleshooting

**release fails with `422 Validation Failed — tag_name already_exists`**

this happens when a previous goreleaser run partially created a GitHub release for the same tag (e.g. interrupted mid-upload). goreleaser cannot overwrite an existing release.

fix: delete the partial release(s) and re-run:

```bash
make release-reset   # deletes any existing GitHub release for the current tag
make release
```

---

## 7. roadmap

### milestones

| milestone                  | goal                                             | status      |
| :------------------------- | :----------------------------------------------- | :---------- |
| M1 — minimal viable client | TCP connection, raw stream display, basic input  | complete    |
| M2 — protocol foundation   | Telnet negotiation (GA/ECHO), MCCP compression   | complete    |
| M3 — aardwolf protocols    | GMCP parsing + state management, MSP support     | complete    |
| M4 — extensibility         | Lua scripting, basic mapper                      | complete    |

---

### m1 — minimal viable client
- [x] basic TCP socket connection to a host/port.
- [x] raw stream display in a simple TUI.
- [x] basic user command input.
- [x] ANSI colour support and stripping.

### m2 — protocol foundation
- [x] basic Telnet negotiation (support for standard GA/ECHO).
- [x] implement MCCP (compression) for performance.
- [x] TLS support for secure connections (generic MUDs via `tls://` prefix; Aardwolf does not offer TLS).

### m3 — aardwolf & advanced protocols
- [x] GMCP parsing and state management.

### m4 — extensibility
- [/] integrate Lua for user-defined triggers/aliases (VM embedded, basic API).
- [x] basic mapper implementation for visual room tracking.

---

### near term
- [ ] `[mapper]` implement multilevel map support (Z-axis filtering and vertical exits) [medium]
- [ ] `[mapper]` handle special rooms (magic, clan, no-coord) via zone-based partitioning [medium]
- [ ] `[ui]` implement command mode triggered by "/" (e.g. /help, /map, /raw, /quit) [medium]
- [ ] `[ui]` implement buffer lock for scrolling to buffer history [medium]
- [ ] `[ui]` implement navigation lock for using arrow keys to move [medium]
- [ ] `[ui]` disable escape back when in MUD session [easy]
- [ ] `[ui]` cap scrollback buffer size to prevent unbounded memory growth in long sessions [medium]
- [ ] `[ui]` do not hide or filter chats etc from main stream for ease of debugging [easy]
- [ ] `[ui]` hotkey toggle arrow keys for movement [easy]
- [ ] `[ui]` aardwolf allow user to enter quit command. on quit (wait ensure disconnected), and return to main screen. there is bug when entering quit in aardwolf [easy]
- [ ] `[ui]` make sure all toggles and hotkeys references are in help menu [easy]
- [ ] `[network]` add Aardwolf-specific GMCP module handlers (stats, room, inventory) [medium]
- [ ] `[network]` add MSP (MUD Sound Protocol) support [medium]
- [ ] `[engine/ui]` add support for user-defined hotkeys/aliases via Lua [medium]
- [ ] `[ui]` implement layered input manager to handle keybinding/command conflicts [medium]
- [ ] `[ui]` add search feature to scrollback buffer [medium]
- [ ] `[engine/ui]` implement regex-based line diversion (combat/spells/chat) [medium]
- [ ] `[engine/ui]` add event-driven spell/buff dashboard [medium]
- [ ] `[network/engine]` improve speedwalks and speedrun capabilities [medium]
- [ ] `[ui]` investigate dedicated UI panels for GMCP-backed chat and equipment [medium]
- [ ] `[ui]` add Aardwolf in-game time and date ticker to top bar [medium]
- [ ] `[ui]` add theming support and theme switcher [medium]
- [ ] `[engine/ui]` add toggle for showing and editing lua based aliases [medium]
- [x] establish project structure for engine, network, ui, and mapper packages. [medium]
- [ ] `[network/engine/ui]` implement virtual scrollback buffer for performance. toggle for search and scrollback [hard]
- [ ] `[engine]` add toggle for filtering Aardwolf-specific text tags (e.g., {say}, {affoff}) [easy]
- [x] `[network]` implement auto-reconnect with configurable backoff [easy]
- [x] `[ui]` add settings/connection status display (MCCP, telnet details) [easy]
- [x] `[ui]` add system info (date, time, CPU/memory) to top bar [easy]
- [x] `[ui]` add top bar with version, project name, and GitHub link [easy]
- [x] `[ui]` improve UI with icons and visual symbols [easy]
- [x] `[ui]` allow quitting mud session to return to the main screen [easy]
- [x] `[network]` allow aardwolf to quit game cleanly [easy]
- [x] `[ui]` toggle raw mode — print all received bytes unprocessed for debugging [easy]
- [x] `[mapper]` persist room graph to disk with periodic auto-save and load on startup [easy]
- [x] `[ui]` have a help menu hotkey "?" [easy]
- [x] `[ui]` navigation up and down should go back in history of commands [easy]
- [x] `[build]` setup GoReleaser for automated versioning and Homebrew deployment [medium]
- [x] `[ui]` toggle map mode (new screen or overlay) [medium]
- [x] `[ui]` improve display with border panes [medium]

### ideas
- split-pane layout (main output + status/map panel).
- visual mapper with room graph rendering.
- plugin system for protocol extensions.
