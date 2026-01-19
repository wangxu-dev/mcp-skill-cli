package registryindex

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/mcp"
)

func SyncSkill(entry SkillEntry) error {
	if strings.TrimSpace(entry.Name) == "" {
		return fmt.Errorf("invalid skill entry: missing name")
	}

	needs, err := needsUpdate("skill", entry.Name, entry.Head)
	if err != nil {
		return err
	}
	if !needs {
		return nil
	}

	tempDir, err := os.MkdirTemp("", "mcp-skill-registry-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	repo := registryRepo()
	if strings.TrimSpace(repo) == "" {
		return fmt.Errorf("registry repo is empty")
	}
	if err := gitClone(repo, tempDir); err != nil {
		return err
	}

	path := filepath.Join(tempDir, "skill", entry.Name)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("skill path not found: skill/%s", entry.Name)
	}
	if _, err := installer.CacheSkillDir(path); err != nil {
		return err
	}
	return SaveLocalRecord("skill", LocalRecord{
		Name: entry.Name,
		Repo: entry.Repo,
		Path: entry.Path,
		Head: entry.Head,
	})
}

func SyncMCP(entry MCPEntry) error {
	if strings.TrimSpace(entry.Name) == "" {
		return fmt.Errorf("invalid mcp entry: missing name")
	}

	needs, err := needsUpdate("mcp", entry.Name, entry.Head)
	if err != nil {
		return err
	}
	if !needs {
		return nil
	}

	entryType := strings.ToLower(strings.TrimSpace(entry.Type))
	if entryType == "" && strings.TrimSpace(entry.URL) != "" {
		entryType = "http"
	}
	if entryType == "" && strings.TrimSpace(entry.Path) != "" {
		entryType = "stdio"
	}

	if entryType == "http" {
		def, err := mcp.DefinitionFromArgs(entry.Name, "http", entry.URL, "", nil)
		if err != nil {
			return err
		}
		def.Headers = entry.Headers
		if _, err := mcp.SaveLocalDefinition(def); err != nil {
			return err
		}
	} else {
		if strings.TrimSpace(entry.Repo) == "" || strings.TrimSpace(entry.Path) == "" {
			return fmt.Errorf("invalid mcp entry: missing repo/path")
		}
		tempDir, err := os.MkdirTemp("", "mcp-registry-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)

		if err := gitClone(entry.Repo, tempDir); err != nil {
			return err
		}

		path := filepath.Join(tempDir, filepath.FromSlash(entry.Path))
		def, err := mcp.LoadDefinitionFromFile(path)
		if err != nil {
			return err
		}
		def.Name = entry.Name
		if _, err := mcp.SaveLocalDefinition(def); err != nil {
			return err
		}
	}
	return SaveLocalRecord("mcp", LocalRecord{
		Name: entry.Name,
		Repo: entry.Repo,
		Path: entry.Path,
		Head: entry.Head,
	})
}

func needsUpdate(kind, name, head string) (bool, error) {
	record, ok, err := LoadLocalRecord(kind, name)
	if err != nil {
		return true, err
	}
	if !ok {
		return true, nil
	}
	if record.Head != head {
		return true, nil
	}
	if !cachedEntryExists(kind, name) {
		return true, nil
	}
	return false, nil
}

func gitClone(repoURL, dest string) error {
	url, err := normalizeRepoURL(repoURL)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", "--quiet", "--depth", "1", url, dest)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(output.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("git clone failed: %s", msg)
	}
	return nil
}

func normalizeRepoURL(repo string) (string, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", fmt.Errorf("repository is required")
	}
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "git@") {
		return repo, nil
	}
	if strings.Count(repo, "/") == 1 {
		return "https://github.com/" + repo + ".git", nil
	}
	return "", fmt.Errorf("unsupported repository format: %s", repo)
}

func cachedEntryExists(kind, name string) bool {
	switch kind {
	case "skill":
		path, err := SkillPathInStore(name)
		if err != nil {
			return false
		}
		info, err := os.Stat(path)
		return err == nil && info.IsDir()
	case "mcp":
		path, err := MCPPathInStore(name)
		if err != nil {
			return false
		}
		info, err := os.Stat(path)
		return err == nil && info.Mode().IsRegular()
	default:
		return false
	}
}
