package skillcli

import (
	"bytes"
	"os"
	"path/filepath"

	"mcp-skill-manager/internal/installer"
)

func localStoreSkillPath(name string) (string, error) {
	root, err := installer.LocalSkillStore()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

func needsSkillUpdate(installedPath, cachedPath string) (bool, string, string, error) {
	installedVersion, installedErr := readSkillVersion(installedPath)
	if installedErr != nil && !os.IsNotExist(installedErr) {
		return false, "", "", installedErr
	}
	cachedVersion, cachedErr := readSkillVersion(cachedPath)
	if cachedErr != nil && !os.IsNotExist(cachedErr) {
		return false, "", "", cachedErr
	}
	if installedVersion != "" && cachedVersion != "" {
		return installedVersion != cachedVersion, installedVersion, cachedVersion, nil
	}
	if installedVersion == "" && cachedVersion == "" {
		installedData, err := os.ReadFile(filepath.Join(installedPath, "SKILL.md"))
		if err != nil && !os.IsNotExist(err) {
			return false, "", "", err
		}
		cachedData, err := os.ReadFile(filepath.Join(cachedPath, "SKILL.md"))
		if err != nil && !os.IsNotExist(err) {
			return false, "", "", err
		}
		if len(installedData) > 0 && len(cachedData) > 0 {
			return !bytes.Equal(installedData, cachedData), "", "", nil
		}
	}
	return true, installedVersion, cachedVersion, nil
}
