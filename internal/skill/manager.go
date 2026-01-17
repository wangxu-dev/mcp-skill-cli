package skill

import "mcp-skill-manager/internal/installer"

type Installed struct {
	Name   string
	Client installer.Tool
	Scope  string
	Path   string
}

func Install(source, scope, cwd string, clients []installer.Tool, force bool) ([]Installed, error) {
	records, err := installer.InstallFromInput(source, scope, clients, cwd, force)
	if err != nil {
		return nil, err
	}
	results := make([]Installed, 0, len(records))
	for _, record := range records {
		results = append(results, Installed{
			Name:   record.SkillName,
			Client: record.Tool,
			Scope:  scope,
			Path:   record.DestPath,
		})
	}
	return results, nil
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
