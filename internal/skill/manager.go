package skill

import (
	"fmt"
	"os"
	"strings"

	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/registryindex"
)

type Installed struct {
	Name   string
	Client installer.Tool
	Scope  string
	Path   string
}

func Install(source, scope, cwd string, clients []installer.Tool, force bool) ([]Installed, error) {
	if isLocalPath(source) || isRepoInput(source) {
		records, err := installer.InstallFromInput(source, scope, clients, cwd, force)
		if err != nil {
			return nil, err
		}
		return mapInstallRecords(records, scope), nil
	}

	if err := registryindex.EnsureIndexes(); err != nil {
		records, localErr := installer.InstallFromLocalStore(source, scope, clients, cwd, force)
		if localErr != nil {
			return nil, err
		}
		return mapInstallRecords(records, scope), nil
	}
	entry, ok, err := registryindex.FindSkill(source)
	if err != nil {
		return nil, err
	}
	if ok {
		if err := registryindex.SyncSkill(entry); err != nil {
			records, localErr := installer.InstallFromLocalStore(entry.Name, scope, clients, cwd, force)
			if localErr != nil {
				return nil, err
			}
			return mapInstallRecords(records, scope), nil
		}
		records, err := installer.InstallFromLocalStore(entry.Name, scope, clients, cwd, force)
		if err != nil {
			return nil, err
		}
		return mapInstallRecords(records, scope), nil
	}

	records, err := installer.InstallFromLocalStore(source, scope, clients, cwd, force)
	if err != nil {
		return nil, fmt.Errorf("skill not found in registry or local store: %s", source)
	}
	return mapInstallRecords(records, scope), nil
}

func List(scopes []string, cwd string, clients []installer.Tool) ([]Installed, error) {
	items, err := installer.ListInstalled(clients, scopes, cwd)
	if err != nil {
		return nil, err
	}
	results := make([]Installed, 0, len(items))
	for _, item := range items {
		results = append(results, Installed{
			Name:   item.SkillName,
			Client: item.Tool,
			Scope:  item.Scope,
			Path:   item.Path,
		})
	}
	return results, nil
}

func Uninstall(name, scope, cwd string, clients []installer.Tool, force bool) ([]Installed, error) {
	records, err := installer.UninstallSkill(name, scope, clients, cwd, force)
	if err != nil {
		return nil, err
	}
	results := make([]Installed, 0, len(records))
	for _, record := range records {
		results = append(results, Installed{
			Name:   record.SkillName,
			Client: record.Tool,
			Scope:  record.Scope,
			Path:   record.Path,
		})
	}
	return results, nil
}

func UninstallAll(scope, cwd string, clients []installer.Tool) ([]Installed, error) {
	records, err := installer.UninstallAll(scope, clients, cwd)
	if err != nil {
		return nil, err
	}
	results := make([]Installed, 0, len(records))
	for _, record := range records {
		results = append(results, Installed{
			Name:   record.SkillName,
			Client: record.Tool,
			Scope:  record.Scope,
			Path:   record.Path,
		})
	}
	return results, nil
}

func LocalStorePaths() (string, string, error) {
	skillRoot, err := installer.LocalSkillStore()
	if err != nil {
		return "", "", err
	}
	mcpRoot, err := installer.LocalMcpStore()
	if err != nil {
		return "", "", err
	}
	return skillRoot, mcpRoot, nil
}

func CleanLocalStore() error {
	return installer.CleanLocalStore()
}

func mapInstallRecords(records []installer.InstallRecord, scope string) []Installed {
	results := make([]Installed, 0, len(records))
	for _, record := range records {
		results = append(results, Installed{
			Name:   record.SkillName,
			Client: record.Tool,
			Scope:  scope,
			Path:   record.DestPath,
		})
	}
	return results
}

func isLocalPath(value string) bool {
	if value == "" {
		return false
	}
	_, err := os.Stat(value)
	return err == nil
}

func isRepoInput(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "git@") {
		return true
	}
	return strings.Count(value, "/") == 1
}
