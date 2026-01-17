package registryindex

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/mcp"
)

func SyncSkill(entry SkillEntry) error {
	if strings.TrimSpace(entry.Name) == "" || strings.TrimSpace(entry.Repo) == "" || strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("invalid skill entry: missing name/repo/path")
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

	if err := gitClone(entry.Repo, tempDir); err != nil {
		return err
	}

	path := filepath.Join(tempDir, filepath.FromSlash(entry.Path))
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("skill path not found: %s", entry.Path)
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
	if strings.TrimSpace(entry.Name) == "" || strings.TrimSpace(entry.Repo) == "" || strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("invalid mcp entry: missing name/repo/path")
	}

	needs, err := needsUpdate("mcp", entry.Name, entry.Head)
	if err != nil {
		return err
	}
	if !needs {
		return nil
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
	return record.Head != head, nil
}

func gitClone(repoURL, dest string) error {
	url, err := normalizeRepoURL(repoURL)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", "--depth", "1", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
