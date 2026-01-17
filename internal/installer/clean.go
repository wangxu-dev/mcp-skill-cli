package installer

import (
	"os"
	"path/filepath"
)

func CleanLocalStore() error {
	skillRoot, err := LocalSkillStore()
	if err != nil {
		return err
	}
	if err := clearDir(skillRoot); err != nil {
		return err
	}

	mcpRoot, err := LocalMcpStore()
	if err != nil {
		return err
	}
	if err := clearDir(mcpRoot); err != nil {
		return err
	}

	return nil
}

func clearDir(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
