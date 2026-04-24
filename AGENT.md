# kuda — agent instructions

this document is intended for ai coding assistants working in the `kuda` directory.

## commands

- **build maps**: `make maps`
- **build website**: `make website`
- **develop locally**: `make website-develop`
- **install deps**: `brew bundle`

## project context

- **description**: a map-building toolkit for aardwolf mud zones, rendered as diagrams via plantuml, graphviz, and mermaid.
- **output formats**: svg (plantuml, graphviz), static website (hugo)
- **map source files**: `src/maps/` — `.puml`, `.dot`, `.mmd`
- **website source**: `src/website/` — hugo project; built output goes to `docs/`
- **architecture**: see `SPEC.md` for full details on conventions and structure.

## conventions

- **room naming**: `level_roomnumber` — e.g. `50_0001` means land level 50, room 0001.
  - level 50 = ground level; 51/52 = above ground; 49/48 = below ground.
- **connections**: use plantuml directional connectors (`-u-`, `-d-`, `-l-`, `-r-`) for cardinal directions.
- **multi-floor links**: use dotted arrows with direction label (e.g. `50_0003 .r.> 51_0001 : up`).
- **map format preference**: plantuml (`.puml`) is the primary format; `.dot` and `.mmd` are supplementary examples.
- **generated files**: `.svg` outputs are gitignored; they are regenerated on `make maps`.
- **no emojis**: use plain ascii or unicode characters for all content.
- **comments**: add concise inline comments explaining room context where helpful.
