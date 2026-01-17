package registryindex

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultRepo   = "https://github.com/wangxu-dev/mcp-skill-registry"
	defaultBranch = "main"
	skillIndex    = "index.skill.json"
	mcpIndex      = "index.mcp.json"
	metaFile      = "index.meta.json"
	syncTTL       = 5 * time.Minute
)

func registryRepo() string {
	if value := strings.TrimSpace(os.Getenv("MCP_REGISTRY_REPO")); value != "" {
		return value
	}
	return defaultRepo
}

func registryBranch() string {
	if value := strings.TrimSpace(os.Getenv("MCP_REGISTRY_BRANCH")); value != "" {
		return value
	}
	return defaultBranch
}

func rawBaseURL() (string, error) {
	repo := strings.TrimSpace(registryRepo())
	if repo == "" {
		return "", fmt.Errorf("registry repo is empty")
	}
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		repo = strings.TrimPrefix(repo, "https://github.com/")
		repo = strings.TrimPrefix(repo, "http://github.com/")
	}
	repo = strings.TrimSuffix(repo, ".git")
	if strings.Count(repo, "/") != 1 {
		return "", fmt.Errorf("invalid registry repo: %s", repo)
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/", repo, registryBranch()), nil
}
