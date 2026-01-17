package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Tool string

const (
	ToolClaude   Tool = "claude"
	ToolCodex    Tool = "codex"
	ToolGemini   Tool = "gemini"
	ToolOpenCode Tool = "opencode"
)

var allTools = []Tool{ToolClaude, ToolCodex, ToolGemini, ToolOpenCode}

func ParseTools(value string) ([]Tool, error) {
	if strings.TrimSpace(value) == "" {
		return allTools, nil
	}

	raw := strings.Split(value, ",")
	seen := map[Tool]bool{}
	var tools []Tool
	for _, item := range raw {
		name := Tool(strings.ToLower(strings.TrimSpace(item)))
		if name == "" {
			continue
		}
		if name == "all" {
			return allTools, nil
		}
		if !isSupportedTool(name) {
			return nil, fmt.Errorf("unknown tool: %s", item)
		}
		if !seen[name] {
			seen[name] = true
			tools = append(tools, name)
		}
	}

	if len(tools) == 0 {
		return nil, fmt.Errorf("no tools specified")
	}
	return tools, nil
}

func isSupportedTool(tool Tool) bool {
	for _, candidate := range allTools {
		if candidate == tool {
			return true
		}
	}
	return false
}

func ResolveRoot(tool Tool, scope, cwd string) (string, error) {
	switch scope {
	case "", ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return userRoot(tool, home)
	case ScopeProject:
		if cwd == "" {
			return "", fmt.Errorf("project scope requires working directory")
		}
		return projectRoot(tool, cwd)
	default:
		return "", fmt.Errorf("invalid scope: %s", scope)
	}
}

func userRoot(tool Tool, home string) (string, error) {
	switch tool {
	case ToolClaude:
		return filepath.Join(home, ".claude", "skills"), nil
	case ToolCodex:
		return filepath.Join(home, ".codex", "skills"), nil
	case ToolGemini:
		return filepath.Join(home, ".gemini", "skills"), nil
	case ToolOpenCode:
		return filepath.Join(home, ".config", "opencode", "skill"), nil
	default:
		return "", fmt.Errorf("unsupported tool: %s", tool)
	}
}

func projectRoot(tool Tool, cwd string) (string, error) {
	switch tool {
	case ToolClaude:
		return filepath.Join(cwd, ".claude", "skills"), nil
	case ToolCodex:
		return filepath.Join(cwd, ".codex", "skills"), nil
	case ToolGemini:
		return filepath.Join(cwd, ".gemini", "skills"), nil
	case ToolOpenCode:
		return filepath.Join(cwd, ".opencode", "skill"), nil
	default:
		return "", fmt.Errorf("unsupported tool: %s", tool)
	}
}
