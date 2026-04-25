# changelog

all notable changes to kuda are documented here.
format follows [keep a changelog](https://keepachangelog.com/en/1.1.0/).
versions follow [semantic versioning](https://semver.org/).

---

## [unreleased]

### added
- `AGENTS.md` with ai assistant guidelines for this project
- `SPEC.md` with project specification and architecture
- `CHANGELOG.md` (this file)
- `GEMINI.md` and `CLAUDE.md` symlinks pointing to `AGENTS.md`
- `Makefile` for building and testing
- `Brewfile` with project dependencies
- basic project structure in root folder

---

## [v0.1.0] — 2026-04-24

### added
- initial project setup for go-based mud client
- `.gitignore` for binaries and common temporary files

### changed
- moved source files from `src/` to root for idiomatic go structure
- cleaned up legacy references to hugo and mapping tools
