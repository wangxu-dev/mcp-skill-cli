package mcp

import (
	"fmt"

	"mcp-skill-manager/internal/installer"
)

type Installed struct {
	Name      string
	Client    installer.Tool
	Scope     string
	Path      string
	Transport string
}

func Install(def Definition, scope, cwd string, clients []installer.Tool, force bool) ([]Installed, error) {
	var results []Installed
	for _, client := range clients {
		path, err := installForClient(client, def, scope, cwd, force)
		if err != nil {
			return nil, err
		}
		results = append(results, Installed{
			Name:      def.Name,
			Client:    client,
			Scope:     scope,
			Path:      path,
			Transport: def.Transport,
		})
	}
	return results, nil
}

func Uninstall(name, scope, cwd string, clients []installer.Tool, force bool) ([]Installed, error) {
	var results []Installed
	for _, client := range clients {
		path, err := uninstallForClient(client, name, scope, cwd, force)
		if err != nil {
			return nil, err
		}
		results = append(results, Installed{
			Name:   name,
			Client: client,
			Scope:  scope,
			Path:   path,
		})
	}
	return results, nil
}

func List(scopes []string, cwd string, clients []installer.Tool) ([]Installed, error) {
	var results []Installed
	for _, scope := range scopes {
		for _, client := range clients {
			entries, path, err := listForClient(client, scope, cwd)
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				results = append(results, Installed{
					Name:      entry.Name,
					Client:    client,
					Scope:     scope,
					Path:      path,
					Transport: entry.Transport,
				})
			}
		}
	}
	return results, nil
}

func installForClient(client installer.Tool, def Definition, scope, cwd string, force bool) (string, error) {
	switch client {
	case installer.ToolClaude:
		return InstallClaude(def, scope, cwd, force)
	case installer.ToolCodex:
		return InstallCodex(def, scope, cwd, force)
	case installer.ToolGemini:
		return InstallGemini(def, scope, cwd, force)
	case installer.ToolOpenCode:
		return InstallOpenCode(def, scope, cwd, force)
	default:
		return "", fmt.Errorf("unsupported client: %s", client)
	}
}

func uninstallForClient(client installer.Tool, name, scope, cwd string, force bool) (string, error) {
	switch client {
	case installer.ToolClaude:
		return UninstallClaude(name, scope, cwd, force)
	case installer.ToolCodex:
		return UninstallCodex(name, scope, cwd, force)
	case installer.ToolGemini:
		return UninstallGemini(name, scope, cwd, force)
	case installer.ToolOpenCode:
		return UninstallOpenCode(name, scope, cwd, force)
	default:
		return "", fmt.Errorf("unsupported client: %s", client)
	}
}

func listForClient(client installer.Tool, scope, cwd string) ([]Entry, string, error) {
	switch client {
	case installer.ToolClaude:
		return ListClaude(scope, cwd)
	case installer.ToolCodex:
		return ListCodex(scope, cwd)
	case installer.ToolGemini:
		return ListGemini(scope, cwd)
	case installer.ToolOpenCode:
		return ListOpenCode(scope, cwd)
	default:
		return nil, "", fmt.Errorf("unsupported client: %s", client)
	}
}
