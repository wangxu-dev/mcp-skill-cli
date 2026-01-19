package mcp

import (
	"fmt"
	"os"
	"path/filepath"

	"mcp-skill-manager/internal/installer"
)

func OpenCodeConfigPath(scope, cwd string) (string, error) {
	switch scope {
	case installer.ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "opencode", "opencode.json"), nil
	case installer.ScopeProject:
		if cwd == "" {
			return "", fmt.Errorf("project scope requires working directory")
		}
		return filepath.Join(cwd, ".opencode", "opencode.json"), nil
	default:
		return "", fmt.Errorf("invalid scope: %s", scope)
	}
}

func InstallOpenCode(def Definition, scope, cwd string, force bool) (string, error) {
	path, err := OpenCodeConfigPath(scope, cwd)
	if err != nil {
		return "", err
	}
	config, err := loadJSONConfig(path)
	if err != nil {
		return "", err
	}
	mcpSection := ensureMap(config, "mcp")
	if _, exists := mcpSection[def.Name]; exists && !force {
		return "", fmt.Errorf("server already exists: %s", def.Name)
	}
	mcpSection[def.Name] = toOpenCodeServer(def)
	if _, ok := config["$schema"]; !ok {
		config["$schema"] = "https://opencode.ai/config.json"
	}
	if err := writeJSONConfig(path, config); err != nil {
		return "", err
	}
	return path, nil
}

func UninstallOpenCode(name, scope, cwd string, force bool) (string, error) {
	path, err := OpenCodeConfigPath(scope, cwd)
	if err != nil {
		return "", err
	}
	config, err := loadJSONConfig(path)
	if err != nil {
		return "", err
	}
	mcpSection := ensureMap(config, "mcp")
	if _, exists := mcpSection[name]; !exists {
		if force {
			return path, nil
		}
		return "", fmt.Errorf("server not found: %s", name)
	}
	delete(mcpSection, name)
	if err := writeJSONConfig(path, config); err != nil {
		return "", err
	}
	return path, nil
}

func ListOpenCode(scope, cwd string) ([]Entry, string, error) {
	path, err := OpenCodeConfigPath(scope, cwd)
	if err != nil {
		return nil, "", err
	}
	config, err := loadJSONConfig(path)
	if err != nil {
		return nil, "", err
	}
	servers := ensureMap(config, "mcp")
	return extractEntries(servers), path, nil
}

func toOpenCodeServer(def Definition) map[string]any {
	if def.Transport == "http" {
		return map[string]any{
			"type": "remote",
			"url":  def.URL,
		}
	}
	command := []string{def.Command}
	command = append(command, def.Args...)
	server := map[string]any{
		"type":    "local",
		"command": command,
	}
	if len(def.Env) > 0 {
		server["env"] = def.Env
	}
	return server
}
