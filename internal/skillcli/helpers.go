package skillcli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"mcp-skill-manager/internal/installer"
)

func resolveScope(scope string, global bool, local bool) (string, error) {
	if global && local {
		return "", fmt.Errorf("choose only one of global or local")
	}
	if global {
		return installer.ScopeUser, nil
	}
	if local {
		return installer.ScopeProject, nil
	}

	scope = strings.ToLower(strings.TrimSpace(scope))
	switch scope {
	case "", "local", "project":
		return installer.ScopeProject, nil
	case "global", "user":
		return installer.ScopeUser, nil
	default:
		return "", fmt.Errorf("unknown scope: %s", scope)
	}
}

func resolveListScopes(global bool, local bool) []string {
	if global && local {
		return []string{installer.ScopeUser, installer.ScopeProject}
	}
	if global {
		return []string{installer.ScopeUser}
	}
	if local {
		return []string{installer.ScopeProject}
	}
	return []string{installer.ScopeProject}
}

func resolveClientValue(clientFlag, clientShort, toolFlag string, all bool) (string, error) {
	if all {
		if clientFlag != "" || clientShort != "" || toolFlag != "" {
			return "", fmt.Errorf("cannot combine --all with --client")
		}
		return "all", nil
	}

	clientValue := clientFlag
	if clientShort != "" {
		clientValue = clientShort
	}
	if toolFlag != "" {
		if clientValue != "" && clientValue != toolFlag {
			return "", fmt.Errorf("conflicting client flags: use only one of --client/-c/--tool")
		}
		clientValue = toolFlag
	}
	if strings.TrimSpace(clientValue) == "" {
		return "", fmt.Errorf("choose a client with --client/-c or use --all")
	}
	return clientValue, nil
}

func resolveListClientValue(clientFlag, clientShort, toolFlag string) (string, error) {
	clientValue := clientFlag
	if clientShort != "" {
		clientValue = clientShort
	}
	if toolFlag != "" {
		if clientValue != "" && clientValue != toolFlag {
			return "", fmt.Errorf("conflicting client flags: use only one of --client/-c/--tool")
		}
		clientValue = toolFlag
	}
	if strings.TrimSpace(clientValue) == "" {
		return "all", nil
	}
	return clientValue, nil
}

func matchesSkillFilter(name, filter string) bool {
	if strings.TrimSpace(filter) == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
}

func truncateDescription(value string, limit int) string {
	value = strings.TrimSpace(value)
	if value == "" || limit <= 0 {
		return value
	}
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func confirmPrompt(out io.Writer, prompt string) bool {
	fmt.Fprint(out, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "yes"
}

func countEntries(path string) (int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return len(entries), nil
}

func splitArgs(args []string) ([]string, []string) {
	var flags []string
	var positionals []string
	valueFlags := map[string]bool{
		"--scope":  true,
		"--tool":   true,
		"--client": true,
		"-c":       true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if valueFlags[arg] && !strings.Contains(arg, "=") {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					flags = append(flags, args[i+1])
					i++
				}
			}
			continue
		}
		positionals = append(positionals, arg)
	}

	return flags, positionals
}
