package installer

import (
	"os"
	"path/filepath"
)

func LocalStoreRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mcp-skill"), nil
}

func LocalSkillStore() (string, error) {
	root, err := LocalStoreRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "skill"), nil
}

func LocalMcpStore() (string, error) {
	root, err := LocalStoreRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "mcp"), nil
}
