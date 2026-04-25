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
- **comments**: add concise inline comments explaining logic where helpful.
