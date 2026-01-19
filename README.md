# mcp-skill-cli

Cross-platform CLI installer for MCP servers (`mcp`) and skills (`skill`). It
manages local cache under `~/.mcp-skill/`, updates client configs, and ships
prebuilt binaries via GitHub Releases.

## Install

```bash
npm install -g mcp-skill-cli
```

## Quick Start

```bash
# list installed skills (project scope)
skill list

# install a skill from the registry
skill install react-best-practices -c opencode

# list MCP servers (user scope)
mcp list -g

# install an MCP server by name
mcp install context7 -g -c codex
```

## Supported Clients

- `claude`
- `codex`
- `gemini`
- `opencode`
- `cursor`
- `amp`
- `kilocode`
- `roo`
- `goose`
- `antigravity`
- `copilot`
- `clawdbot`
- `droid`
- `windsurf`

## Local Cache

The CLI stores cached assets here:

- `~/.mcp-skill/skill/`
- `~/.mcp-skill/mcp/`
- `~/.mcp-skill/index.skill.json`
- `~/.mcp-skill/index.mcp.json`

## Environment Variables

- `MCP_SKIP_DOWNLOAD=1` skips downloading release binaries.
- `MCP_SKILL_RELEASE_REPO` overrides the GitHub repo for releases.

## Releases

The npm package is a thin wrapper. On install, it downloads the matching
`mcp` and `skill` binaries from GitHub Releases based on OS and architecture.
To update, bump the npm version and publish a new release tag.

## Troubleshooting

- If install fails, ensure the GitHub Release for your version exists.
- Windows users should confirm that antivirus did not block the downloaded `.exe`.
