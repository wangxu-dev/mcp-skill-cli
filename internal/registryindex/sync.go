package registryindex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"mcp-skill-manager/internal/installer"
)

type Meta struct {
	Repo      string `json:"repo"`
	Branch    string `json:"branch"`
	LastSync  string `json:"lastSync"`
	SkillFile string `json:"skillIndex"`
	MCPFile   string `json:"mcpIndex"`
}

func SyncIfStale() error {
	root, err := installer.LocalStoreRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	meta, _ := loadMeta(root)
	if !shouldSync(meta, root) {
		return nil
	}

	base, err := rawBaseURL()
	if err != nil {
		return err
	}

	skillURL := base + skillIndex
	mcpURL := base + mcpIndex

	if err := downloadFile(skillURL, filepath.Join(root, skillIndex)); err != nil {
		return err
	}
	if err := downloadFile(mcpURL, filepath.Join(root, mcpIndex)); err != nil {
		return err
	}

	meta = Meta{
		Repo:      registryRepo(),
		Branch:    registryBranch(),
		LastSync:  time.Now().UTC().Format(time.RFC3339),
		SkillFile: skillIndex,
		MCPFile:   mcpIndex,
	}
	return saveMeta(root, meta)
}

func shouldSync(meta Meta, root string) bool {
	if meta.LastSync == "" {
		return true
	}
	if !fileExists(filepath.Join(root, skillIndex)) || !fileExists(filepath.Join(root, mcpIndex)) {
		return true
	}
	lastSync, err := time.Parse(time.RFC3339, meta.LastSync)
	if err != nil {
		return true
	}
	return time.Since(lastSync) > syncTTL
}

func loadMeta(root string) (Meta, error) {
	path := filepath.Join(root, metaFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

func saveMeta(root string, meta Meta) error {
	path := filepath.Join(root, metaFile)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		return fmt.Errorf("registry fetch failed: %s (%s)", resp.Status, string(body))
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), "index-*")
	if err != nil {
		return err
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dest)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}
