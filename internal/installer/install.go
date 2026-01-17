package installer

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type InstallRecord struct {
	SkillName string
	Tool      Tool
	DestPath  string
}

func InstallFromInput(input string, scope string, tools []Tool, cwd string, force bool) ([]InstallRecord, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("source is required")
	}

	if isExistingPath(input) {
		return InstallFromPath(input, scope, tools, cwd, force)
	}

	if isRepoInput(input) {
		return InstallFromRepo(input, scope, tools, cwd, force)
	}

	return InstallFromLocalStore(input, scope, tools, cwd, force)
}

func InstallFromRepo(repo string, scope string, tools []Tool, cwd string, force bool) ([]InstallRecord, error) {
	tempDir, err := os.MkdirTemp("", "mcp-skill-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	url, err := normalizeRepoURL(repo)
	if err != nil {
		return nil, err
	}

	if err := gitClone(url, tempDir); err != nil {
		return nil, err
	}

	return InstallFromPath(tempDir, scope, tools, cwd, force)
}

func InstallFromPath(path string, scope string, tools []Tool, cwd string, force bool) ([]InstallRecord, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	skillDirs, err := findSkillDirs(path)
	if err != nil {
		return nil, err
	}
	if len(skillDirs) == 0 {
		return nil, fmt.Errorf("no SKILL.md found in %s", path)
	}

	cachedDirs, err := cacheSkillDirs(skillDirs)
	if err != nil {
		return nil, err
	}

	return installSkillDirs(cachedDirs, scope, tools, cwd, force)
}

func InstallFromLocalStore(name string, scope string, tools []Tool, cwd string, force bool) ([]InstallRecord, error) {
	storeRoot, err := LocalSkillStore()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(storeRoot, name)
	if !isExistingPath(path) {
		return nil, fmt.Errorf("skill not found in local store: %s", name)
	}
	return installSkillDirs([]string{path}, scope, tools, cwd, force)
}

func installSkillDirs(skillDirs []string, scope string, tools []Tool, cwd string, force bool) ([]InstallRecord, error) {
	var records []InstallRecord
	for _, skillDir := range skillDirs {
		skillName := filepath.Base(skillDir)
		for _, tool := range tools {
			root, err := ResolveRoot(tool, scope, cwd)
			if err != nil {
				return nil, err
			}

			dest := filepath.Join(root, skillName)
			if _, err := os.Stat(dest); err == nil {
				if !force {
					return nil, fmt.Errorf("skill already exists: %s (%s)", skillName, dest)
				}
				if err := os.RemoveAll(dest); err != nil {
					return nil, err
				}
			}

			if err := os.MkdirAll(root, 0o755); err != nil {
				return nil, err
			}

			if err := copyDir(skillDir, dest); err != nil {
				return nil, err
			}

			records = append(records, InstallRecord{
				SkillName: skillName,
				Tool:      tool,
				DestPath:  dest,
			})
		}
	}
	return records, nil
}

func cacheSkillDirs(skillDirs []string) ([]string, error) {
	storeRoot, err := LocalSkillStore()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(storeRoot, 0o755); err != nil {
		return nil, err
	}

	var cached []string
	for _, skillDir := range skillDirs {
		skillName := filepath.Base(skillDir)
		dest := filepath.Join(storeRoot, skillName)
		if isWithinRoot(storeRoot, skillDir) {
			cached = append(cached, filepath.Clean(skillDir))
			continue
		}

		if err := os.RemoveAll(dest); err != nil {
			return nil, err
		}
		if err := copyDir(skillDir, dest); err != nil {
			return nil, err
		}
		cached = append(cached, dest)
	}
	return cached, nil
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

func isRepoInput(input string) bool {
	_, err := normalizeRepoURL(input)
	return err == nil
}

func isExistingPath(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isWithinRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".."
}

func gitClone(repoURL, dest string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findSkillDirs(root string) ([]string, error) {
	var dirs []string
	seen := map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return fs.SkipDir
		}
		if d.Type().IsRegular() && d.Name() == "SKILL.md" {
			dir := filepath.Dir(path)
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return dirs, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}

		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink not supported: %s", path)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		return copyFile(path, filepath.Join(dst, rel), info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.ReadFrom(in); err != nil {
		return err
	}

	return out.Close()
}
