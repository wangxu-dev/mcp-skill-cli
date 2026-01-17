package installer

import (
	"fmt"
	"os"
	"path/filepath"
)

type RemoveRecord struct {
	SkillName string
	Tool      Tool
	Scope     string
	Path      string
}

func UninstallSkill(name string, scope string, tools []Tool, cwd string, force bool) ([]RemoveRecord, error) {
	var records []RemoveRecord
	for _, tool := range tools {
		root, err := ResolveRoot(tool, scope, cwd)
		if err != nil {
			return nil, err
		}

		dest := filepath.Join(root, name)
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			if force {
				continue
			}
			return nil, fmt.Errorf("skill not installed: %s (%s)", name, dest)
		} else if err != nil {
			return nil, err
		}

		if err := os.RemoveAll(dest); err != nil {
			return nil, err
		}

		records = append(records, RemoveRecord{
			SkillName: name,
			Tool:      tool,
			Scope:     scope,
			Path:      dest,
		})
	}

	return records, nil
}

func UninstallAll(scope string, tools []Tool, cwd string) ([]RemoveRecord, error) {
	items, err := ListInstalled(tools, []string{scope}, cwd)
	if err != nil {
		return nil, err
	}

	var records []RemoveRecord
	for _, item := range items {
		if err := os.RemoveAll(item.Path); err != nil {
			return nil, err
		}
		records = append(records, RemoveRecord{
			SkillName: item.SkillName,
			Tool:      item.Tool,
			Scope:     item.Scope,
			Path:      item.Path,
		})
	}

	return records, nil
}
