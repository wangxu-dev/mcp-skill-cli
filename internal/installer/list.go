package installer

import (
	"os"
	"path/filepath"
	"sort"
)

type InstalledSkill struct {
	SkillName string
	Tool      Tool
	Scope     string
	Path      string
}

func ListInstalled(tools []Tool, scopes []string, cwd string) ([]InstalledSkill, error) {
	var results []InstalledSkill
	for _, tool := range tools {
		for _, scope := range scopes {
			root, err := ResolveRoot(tool, scope, cwd)
			if err != nil {
				return nil, err
			}

			entries, err := os.ReadDir(root)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				name := entry.Name()
				results = append(results, InstalledSkill{
					SkillName: name,
					Tool:      tool,
					Scope:     scope,
					Path:      filepath.Join(root, name),
				})
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Tool != results[j].Tool {
			return results[i].Tool < results[j].Tool
		}
		if results[i].Scope != results[j].Scope {
			return results[i].Scope < results[j].Scope
		}
		return results[i].SkillName < results[j].SkillName
	})

	return results, nil
}
