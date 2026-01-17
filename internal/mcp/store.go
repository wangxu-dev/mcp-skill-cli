package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"

	"mcp-skill-manager/internal/installer"
)

func LocalDefinitionPath(name string) (string, error) {
	root, err := installer.LocalMcpStore()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name+".json"), nil
}

func LoadLocalDefinition(name string) (Definition, error) {
	path, err := LocalDefinitionPath(name)
	if err != nil {
		return Definition{}, err
	}
	return LoadDefinitionFromFile(path)
}

func SaveLocalDefinition(def Definition) (string, error) {
	path, err := LocalDefinitionPath(def.Name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(definitionFile{
		Transport: def.Transport,
		URL:       def.URL,
		Command:   def.Command,
		Args:      def.Args,
		Env:       def.Env,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func LoadDefinitionFromInput(input string) (Definition, error) {
	if isExistingFile(input) {
		return LoadDefinitionFromFile(input)
	}
	return LoadLocalDefinition(input)
}

func isExistingFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}
