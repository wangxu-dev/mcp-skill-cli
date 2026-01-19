package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Tool string

const (
	ToolClaude      Tool = "claude"
	ToolCodex       Tool = "codex"
	ToolGemini      Tool = "gemini"
	ToolOpenCode    Tool = "opencode"
	ToolCursor      Tool = "cursor"
	ToolAmp         Tool = "amp"
	ToolKiloCode    Tool = "kilocode"
	ToolRooCode     Tool = "roo"
	ToolGoose       Tool = "goose"
	ToolAntigravity Tool = "antigravity"
	ToolCopilot     Tool = "copilot"
	ToolClawdbot    Tool = "clawdbot"
	ToolDroid       Tool = "droid"
	ToolWindsurf    Tool = "windsurf"
)

var allTools = []Tool{
	ToolClaude,
	ToolCodex,
	ToolGemini,
	ToolOpenCode,
	ToolCursor,
	ToolAmp,
	ToolKiloCode,
	ToolRooCode,
	ToolGoose,
	ToolAntigravity,
	ToolCopilot,
	ToolClawdbot,
	ToolDroid,
	ToolWindsurf,
}

func ParseTools(value string) ([]Tool, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("no tools specified")
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
	case ToolCursor:
		return filepath.Join(home, ".cursor", "skills"), nil
	case ToolAmp:
		return filepath.Join(home, ".config", "agents", "skills"), nil
	case ToolKiloCode:
		return filepath.Join(home, ".kilocode", "skills"), nil
	case ToolRooCode:
		return filepath.Join(home, ".roo", "skills"), nil
	case ToolGoose:
		return filepath.Join(home, ".config", "goose", "skills"), nil
	case ToolAntigravity:
		return filepath.Join(home, ".gemini", "antigravity", "skills"), nil
	case ToolCopilot:
		return filepath.Join(home, ".copilot", "skills"), nil
	case ToolClawdbot:
		return filepath.Join(home, ".clawdbot", "skills"), nil
	case ToolDroid:
		return filepath.Join(home, ".factory", "skills"), nil
	case ToolWindsurf:
		return filepath.Join(home, ".codeium", "windsurf", "skills"), nil
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
	case ToolCursor:
		return filepath.Join(cwd, ".cursor", "skills"), nil
	case ToolAmp:
		return filepath.Join(cwd, ".agents", "skills"), nil
	case ToolKiloCode:
		return filepath.Join(cwd, ".kilocode", "skills"), nil
	case ToolRooCode:
		return filepath.Join(cwd, ".roo", "skills"), nil
	case ToolGoose:
		return filepath.Join(cwd, ".goose", "skills"), nil
	case ToolAntigravity:
		return filepath.Join(cwd, ".agent", "skills"), nil
	case ToolCopilot:
		return filepath.Join(cwd, ".github", "skills"), nil
	case ToolClawdbot:
		return filepath.Join(cwd, "skills"), nil
	case ToolDroid:
		return filepath.Join(cwd, ".factory", "skills"), nil
	case ToolWindsurf:
		return filepath.Join(cwd, ".windsurf", "skills"), nil
	default:
		return "", fmt.Errorf("unsupported tool: %s", tool)
	}
}
