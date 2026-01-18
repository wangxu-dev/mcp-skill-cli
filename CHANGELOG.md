# Changelog

## 0.0.4 - 2026-01-18
### Added
- `skill view` defaulting to registry metadata (raw fetch) with `--installed` for local view.
- `skill update` to refresh installed skills when registry versions or content change.
### Changed
- `skill list` now shows version and description (fallback to `SKILL.md` frontmatter).
- Install flow now prompts before overwrite and uses a quiet clone with spinner.
- Cache cleanup now forces resync when missing cached entries.

## 0.0.3 - 2026-01-18
### Fixed
- Cleaned up Windows asset naming and ensured the installer can resolve release files reliably.
### Notes
- Planned re-publish after the npm 24-hour lock expires (2026-01-18).

## 0.0.2 - 2026-01-17
### Added
- Prepared the npm installer wrapper for `mcp` and `skill`.
### Notes
- First publish attempt; npm package was locked after unpublish, delaying release.

## 0.0.1 - 2026-01-17
### Added
- Initial `mcp` and `skill` CLI structure.
- Local cache support under `~/.mcp-skill/`.
- Registry index sync and basic install/list/uninstall/clean flows.
- GitHub Release pipeline for multi-platform binaries.
