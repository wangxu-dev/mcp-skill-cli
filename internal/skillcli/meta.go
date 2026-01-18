package skillcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mcp-skill-manager/internal/registryindex"
)

type SkillMeta struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Head        string `json:"head,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	CheckedAt   string `json:"checkedAt,omitempty"`
}

func loadSkillMeta(skillPath string) (SkillMeta, error) {
	data, err := os.ReadFile(filepath.Join(skillPath, "skill.meta.json"))
	if err != nil {
		return SkillMeta{}, err
	}
	var meta SkillMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return SkillMeta{}, err
	}
	return meta, nil
}

func readSkillVersion(skillPath string) (string, error) {
	version, _, err := readFrontmatterFromSkill(filepath.Join(skillPath, "SKILL.md"))
	return version, err
}

func readSkillDescription(skillPath string) (string, error) {
	_, description, err := readFrontmatterFromSkill(filepath.Join(skillPath, "SKILL.md"))
	return description, err
}

type remoteMetaError struct {
	StatusCode int
	Body       string
}

func (e remoteMetaError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("remote fetch failed: status %d", e.StatusCode)
	}
	return fmt.Sprintf("remote fetch failed: status %d (%s)", e.StatusCode, e.Body)
}

func isRemoteNotFound(err error) bool {
	var remoteErr remoteMetaError
	if !errors.As(err, &remoteErr) {
		return false
	}
	return remoteErr.StatusCode == http.StatusNotFound
}

func fetchRemoteSkillMeta(name string) (SkillMeta, error) {
	base, err := registryindex.RawBaseURL()
	if err != nil {
		return SkillMeta{}, err
	}
	url := fmt.Sprintf("%sskill/%s/skill.meta.json", base, name)
	resp, err := http.Get(url)
	if err != nil {
		return SkillMeta{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		return SkillMeta{}, remoteMetaError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	var meta SkillMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return SkillMeta{}, err
	}
	return meta, nil
}

func readFrontmatterFromSkill(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(string(data), "\n")
	body := extractFrontmatter(lines)
	if len(body) == 0 {
		limit := 40
		if len(lines) < limit {
			limit = len(lines)
		}
		body = lines[:limit]
	}
	var version string
	var description string
	for _, line := range body {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "version:") && version == "" {
			value := strings.TrimSpace(trimmed[len("version:"):])
			version = trimQuoted(value)
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "description:") && description == "" {
			value := strings.TrimSpace(trimmed[len("description:"):])
			description = trimQuoted(value)
		}
	}
	return version, description, nil
}

func extractFrontmatter(lines []string) []string {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return lines[1:i]
		}
	}
	return nil
}

func trimQuoted(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
