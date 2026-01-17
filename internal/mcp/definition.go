package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Definition struct {
	Name      string
	Transport string
	URL       string
	Command   string
	Args      []string
	Env       map[string]string
}

type definitionFile struct {
	Transport string            `json:"transport"`
	Type      string            `json:"type"`
	URL       string            `json:"url"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
}

func LoadDefinitionFromFile(path string) (Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, err
	}

	var raw definitionFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return Definition{}, err
	}

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return normalizeDefinition(Definition{
		Name:      name,
		Transport: raw.Transport,
		URL:       raw.URL,
		Command:   raw.Command,
		Args:      raw.Args,
		Env:       raw.Env,
	}, raw.Type)
}

func DefinitionFromArgs(name, transport, url, command string, args []string) (Definition, error) {
	return normalizeDefinition(Definition{
		Name:      name,
		Transport: transport,
		URL:       url,
		Command:   command,
		Args:      args,
	}, "")
}

func normalizeDefinition(def Definition, typeValue string) (Definition, error) {
	def.Name = strings.TrimSpace(def.Name)
	if def.Name == "" {
		return Definition{}, fmt.Errorf("name is required")
	}

	transport := strings.ToLower(strings.TrimSpace(def.Transport))
	if transport == "" {
		transport = strings.ToLower(strings.TrimSpace(typeValue))
	}
	switch transport {
	case "http", "stdio":
		def.Transport = transport
	case "local":
		def.Transport = "stdio"
	case "remote":
		def.Transport = "http"
	default:
		return Definition{}, fmt.Errorf("unsupported transport: %s", transport)
	}

	switch def.Transport {
	case "http":
		if strings.TrimSpace(def.URL) == "" {
			return Definition{}, fmt.Errorf("url is required for http transport")
		}
	case "stdio":
		if strings.TrimSpace(def.Command) == "" {
			return Definition{}, fmt.Errorf("command is required for stdio transport")
		}
	}

	return def, nil
}
