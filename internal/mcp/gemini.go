package mcp

import (
	"fmt"
	"os"
	"path/filepath"

	"mcp-skill-manager/internal/installer"
)

func GeminiConfigPath(scope, cwd string) (string, error) {
	switch scope {
	case installer.ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".gemini", "mcp.json"), nil
	case installer.ScopeProject:
		if cwd == "" {
			return "", fmt.Errorf("project scope requires working directory")
		}
		return filepath.Join(cwd, ".gemini", "mcp.json"), nil
	default:
		return "", fmt.Errorf("invalid scope: %s", scope)
	}
}

func InstallGemini(def Definition, scope, cwd string, force bool) (string, error) {
	path, err := GeminiConfigPath(scope, cwd)
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

func UninstallGemini(name, scope, cwd string, force bool) (string, error) {
	path, err := GeminiConfigPath(scope, cwd)
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

func ListGemini(scope, cwd string) ([]Entry, string, error) {
	path, err := GeminiConfigPath(scope, cwd)
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
