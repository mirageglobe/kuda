# kuda — agents instructions

this document is intended for ai coding assistants working in the `kuda` directory.

## commands

- **build**: `make build`
- **test**: `make test`
- **clean**: `make clean`
- **install deps**: `brew bundle`

## project context

- **description**: a modern mud client built in go, optimized for aardwolf.
- **technology stack**: go, bubbletea (tui), lua (scripting).
- **architecture**: see `SPEC.md` for full details on conventions and structure.

## conventions

- **language**: go (golang).
- **tui framework**: bubbletea for terminal interface.
- **scripting**: lua for triggers and aliases.
- **testing**: follow idiomatic go testing patterns.
- **no emojis**: use plain ascii or unicode characters for all content.
- **comments**: add concise inline comments explaining non-obvious logic.

## package boundaries — do not cross

each package has a strict ownership boundary documented in its `doc.go`. violations will cause architectural drift that is hard to reverse.

| do NOT | instead |
| :--- | :--- |
| add game logic (state, triggers) to `network/` | put it in `engine/` |
| call network i/o from `ui/` | pass a `ui.Connection` interface |
| call network i/o from `engine/` | pass an `engine.EventSource` interface |
| read/write game state from `ui/` or `mapper/` | read through `engine.GameState` interface |
| merge `LaunchModel` and `ClientModel` into one struct | keep them as separate bubbletea models |
| add rendering logic to `engine/` or `mapper/` | rendering belongs in `ui/` |
| depend on `*network.Client` directly outside `network/` | use the defined interface |

## interface contracts

- `ui.Connection` — what `ui` requires from the network layer (`ui/interfaces.go`)
- `engine.EventSource` — what `engine` consumes from network (`engine/interfaces.go`)
- `engine.GameState` — read-only state exposed to `ui` and `mapper` (`engine/interfaces.go`)

compile-time interface checks (`var _ Interface = (*Impl)(nil)`) are present in each consumer package. do not remove them.

## file size guideline

prefer files that do one thing over files that are short. the signal for splitting is responsibility, not line count — a cohesive telnet state machine or gmcp parser that runs to 200 lines is better left together than artificially split. split when a file clearly owns two distinct concerns, not because it crossed a number.
