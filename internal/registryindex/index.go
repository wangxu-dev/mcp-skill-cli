package registryindex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mcp-skill-manager/internal/installer"
)

func LoadSkillIndex() (SkillIndex, error) {
	root, err := installer.LocalStoreRoot()
	if err != nil {
		return SkillIndex{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, skillIndex))
	if err != nil {
		return SkillIndex{}, err
	}
	var index SkillIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return SkillIndex{}, err
	}
	return index, nil
}

func LoadMCPIndex() (MCPIndex, error) {
	root, err := installer.LocalStoreRoot()
	if err != nil {
		return MCPIndex{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, mcpIndex))
	if err != nil {
		return MCPIndex{}, err
	}
	var index MCPIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return MCPIndex{}, err
	}
	return index, nil
}

func FindSkill(name string) (SkillEntry, bool, error) {
	index, err := LoadSkillIndex()
	if err != nil {
		return SkillEntry{}, false, err
	}
	for _, entry := range index.Skills {
		if strings.EqualFold(entry.Name, name) {
			return entry, true, nil
		}
	}
	return SkillEntry{}, false, nil
}

func FindMCP(name string) (MCPEntry, bool, error) {
	index, err := LoadMCPIndex()
	if err != nil {
		return MCPEntry{}, false, err
	}
	candidates := index.MCP
	if len(candidates) == 0 {
		candidates = index.Servers
	}
	for _, entry := range candidates {
		if strings.EqualFold(entry.Name, name) {
			return entry, true, nil
		}
	}
	return MCPEntry{}, false, nil
}

func SkillPathInStore(name string) (string, error) {
	root, err := installer.LocalSkillStore()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

func MCPPathInStore(name string) (string, error) {
	root, err := installer.LocalMcpStore()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name+".json"), nil
}

func EnsureIndexes() error {
	if err := SyncIfStale(); err != nil {
		return fmt.Errorf("registry sync failed: %w", err)
	}
	return nil
}
