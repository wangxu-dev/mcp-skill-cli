package mcp

import (
	"fmt"
	"os"
	"path/filepath"

	"mcp-skill-manager/internal/installer"
)

func ClaudeConfigPath(scope, cwd string) (string, error) {
	switch scope {
	case installer.ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude.json"), nil
	case installer.ScopeProject:
		if cwd == "" {
			return "", fmt.Errorf("project scope requires working directory")
		}
		return filepath.Join(cwd, ".mcp.json"), nil
	default:
		return "", fmt.Errorf("invalid scope: %s", scope)
	}
}

func InstallClaude(def Definition, scope, cwd string, force bool) (string, error) {
	path, err := ClaudeConfigPath(scope, cwd)
	if err != nil {
		return "", err
	}
	config, err := loadJSONConfig(path)
	if err != nil {
		return "", err
	}
	servers := ensureMap(config, "mcpServers")

	if _, exists := servers[def.Name]; exists && !force {
		return "", fmt.Errorf("server already exists: %s", def.Name)
	}

	servers[def.Name] = toClaudeServer(def)
	if err := writeJSONConfig(path, config); err != nil {
		return "", err
	}
	return path, nil
}

func UninstallClaude(name, scope, cwd string, force bool) (string, error) {
	path, err := ClaudeConfigPath(scope, cwd)
	if err != nil {
		return "", err
	}
	config, err := loadJSONConfig(path)
	if err != nil {
		return "", err
	}
	servers := ensureMap(config, "mcpServers")
	if _, exists := servers[name]; !exists {
		if force {
			return path, nil
		}
		return "", fmt.Errorf("server not found: %s", name)
	}

	delete(servers, name)
	if err := writeJSONConfig(path, config); err != nil {
		return "", err
	}
	return path, nil
}

func ListClaude(scope, cwd string) ([]Entry, string, error) {
	path, err := ClaudeConfigPath(scope, cwd)
	if err != nil {
		return nil, "", err
	}
	config, err := loadJSONConfig(path)
	if err != nil {
		return nil, "", err
	}
	servers := ensureMap(config, "mcpServers")
	return extractEntries(servers), path, nil
}

func toClaudeServer(def Definition) map[string]any {
	if def.Transport == "http" {
		server := map[string]any{
			"type": "http",
			"url":  def.URL,
		}
		if len(def.Headers) > 0 {
			server["headers"] = def.Headers
		}
		return server
	}
	server := map[string]any{
		"type":    "stdio",
		"command": def.Command,
		"args":    def.Args,
	}
	if len(def.Env) > 0 {
		server["env"] = def.Env
	}
	return server
}
