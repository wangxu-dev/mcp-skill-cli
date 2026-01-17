# Repository Guidelines

## Project Structure & Module Organization
This repository targets standalone CLI binaries that install skills and MCP servers.
- `cmd/mcp/` and `cmd/skill/` for CLI entrypoints.
- `internal/mcp/` and `internal/skill/` for core logic.
- `tests/` for integration tests; `testdata/` for fixtures.
- `scripts/` for release automation.
- `docs/` for specs and API notes.
- Local store root: `~/.mcp-skill/` with `skill/` and `mcp/`.

## Build, Test, and Development Commands
Keep these commands working and documented:
- `go build ./cmd/mcp` - build the `mcp` binary.
- `go build ./cmd/skill` - build the `skill` binary.
- `go test ./...` - run the test suite.

## Coding Style & Naming Conventions
- Go formatting: `gofmt` on all `.go` files.
- Package names: short, lowercase, no underscores.
- CLI flags: `--global|-g`, `--local|-l`, `--force|-f`, `--client|-c`, `--all|-a` with `claude,codex,gemini,opencode`.
- Keep CLI parsing thin in `cmd/` and move logic into `internal/`.

## Testing Guidelines
- Use Go’s `testing` package.
- Name tests `TestXxx` and keep fixtures in `testdata/`.
- Cover command parsing, git clone behavior, and install flows.

## Commit & Pull Request Guidelines
Use Conventional Commits:
- `feat: add registry search`
- `fix: handle missing config`
PRs should include a clear description, test steps, and screenshots for UI changes.

## Security & Configuration Tips
- Do not add flags for arbitrary sources; the CLI installs from GitHub repos only.
- Do not commit tokens or private endpoints.
