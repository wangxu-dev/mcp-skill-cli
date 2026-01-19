package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mcp-skill-manager/internal/installer"
)

func CodexConfigPath(scope, cwd string) (string, error) {
	switch scope {
	case installer.ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".codex", "config.toml"), nil
	case installer.ScopeProject:
		return "", fmt.Errorf("codex does not support project-scoped MCP")
	default:
		return "", fmt.Errorf("invalid scope: %s", scope)
	}
}

func InstallCodex(def Definition, scope, cwd string, force bool) (string, error) {
	switch scope {
	case installer.ScopeUser:
		return installCodexGlobal(def, cwd, force)
	default:
		return "", fmt.Errorf("invalid scope: %s", scope)
	}
}

func UninstallCodex(name, scope, cwd string, force bool) (string, error) {
	switch scope {
	case installer.ScopeUser:
		return uninstallCodexGlobal(name, cwd, force)
	default:
		return "", fmt.Errorf("invalid scope: %s", scope)
	}
}

func ListCodex(scope, cwd string) ([]Entry, string, error) {
	switch scope {
	case installer.ScopeUser:
		return listCodexGlobal(cwd)
	default:
		return nil, "", fmt.Errorf("invalid scope: %s", scope)
	}
}

func installCodexGlobal(def Definition, cwd string, force bool) (string, error) {
	path, err := CodexConfigPath(installer.ScopeUser, cwd)
	if err != nil {
		return "", err
	}
	blocks, err := parseTomlBlocks(path)
	if err != nil {
		return "", err
	}

	updated := false
	for i, block := range blocks {
		if block.kind == "mcp" && block.name == def.Name {
			if !force {
				return "", fmt.Errorf("server already exists: %s", def.Name)
			}
			blocks[i].lines = formatTomlEntry(def)
			updated = true
			break
		}
	}

	if !updated {
		if len(blocks) > 0 && len(blocks[len(blocks)-1].lines) > 0 {
			blocks = append(blocks, tomlBlock{kind: "other", lines: []string{""}})
		}
		blocks = append(blocks, tomlBlock{kind: "mcp", name: def.Name, lines: formatTomlEntry(def)})
	}

	if err := writeTomlBlocks(path, blocks); err != nil {
		return "", err
	}
	return path, nil
}

func uninstallCodexGlobal(name, cwd string, force bool) (string, error) {
	path, err := CodexConfigPath(installer.ScopeUser, cwd)
	if err != nil {
		return "", err
	}
	blocks, err := parseTomlBlocks(path)
	if err != nil {
		return "", err
	}

	var updated []tomlBlock
	found := false
	for _, block := range blocks {
		if block.kind == "mcp" && block.name == name {
			found = true
			continue
		}
		updated = append(updated, block)
	}

	if !found && !force {
		return "", fmt.Errorf("server not found: %s", name)
	}
	if err := writeTomlBlocks(path, updated); err != nil {
		return "", err
	}
	return path, nil
}

func listCodexGlobal(cwd string) ([]Entry, string, error) {
	path, err := CodexConfigPath(installer.ScopeUser, cwd)
	if err != nil {
		return nil, "", err
	}
	blocks, err := parseTomlBlocks(path)
	if err != nil {
		return nil, "", err
	}

	var entries []Entry
	for _, block := range blocks {
		if block.kind != "mcp" {
			continue
		}
		entries = append(entries, Entry{
			Name:      block.name,
			Transport: detectTomlTransport(block.lines),
		})
	}
	return entries, path, nil
}

func detectTomlTransport(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "url ") || strings.HasPrefix(trimmed, "url=") {
			return "http"
		}
		if strings.HasPrefix(trimmed, "command ") || strings.HasPrefix(trimmed, "command=") {
			return "stdio"
		}
	}
	return ""
}
