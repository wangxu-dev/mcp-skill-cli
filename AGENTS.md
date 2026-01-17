# Repository Guidelines

## Project Structure & Module Organization
This repository builds two Go CLIs: `mcp` (MCP servers) and `skill` (skills). Keep entrypoints in `cmd/` thin and push logic into `internal/`.
- `cmd/mcp/` and `cmd/skill/` are the binaries’ entrypoints.
- `internal/mcp/` handles per-client config read/write (Claude, Codex, Gemini, OpenCode).
- `internal/installer/` handles skill discovery, copying, and local cache.
- `internal/registryindex/` syncs cloud index files and local metadata.
- `internal/mcpcli/` and `internal/skillcli/` wire up flags and output.
- `bin/` and `scripts/` at repo root hold the npm wrapper that downloads release binaries.
- Local cache root: `~/.mcp-skill/` with `skill/`, `mcp/`, and `index.*.json`.

## Build, Test, and Development Commands
- `go build ./cmd/skill` builds the skill CLI.
- `go build ./cmd/mcp` builds the MCP CLI.
- `go test ./...` runs tests (currently none; still keep it green).

## Coding Style & Naming Conventions
- Run `gofmt` on all Go files.
- Package names are lowercase and short (`mcpcli`, `registryindex`).
- Keep flags consistent (`--global|-g`, `--local|-l`, `--force|-f`, `--client|-c`, `--all|-a`).
- Use ASCII-only strings unless the file already contains Unicode.

## Testing Guidelines
There are no automated tests yet. If adding tests, use the standard `testing` package, name files `*_test.go`, and keep fixtures close to the package (e.g., `internal/mcp/testdata/`).

## Commit & Pull Request Guidelines
Git history uses emoji + Conventional Commits (examples: `✨ feat: ...`, `♻️ refactor: ...`, `🔧 chore: ...`). Keep commit subjects short and scoped to a change. PRs should include a brief description, CLI examples used for verification, and note any config files touched.

## Security & Configuration Tips
- Avoid adding flags for arbitrary remote sources; the registry index is the default source of truth.
- Do not commit tokens, private URLs, or user-specific config files (e.g., `~/.codex/config.toml`).
