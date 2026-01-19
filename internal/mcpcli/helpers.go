package mcpcli

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

func splitArgs(args []string) ([]string, []string) {
	var flags []string
	var positionals []string
	valueFlags := map[string]bool{
		"--scope":     true,
		"--client":    true,
		"--tool":      true,
		"-c":          true,
		"--name":      true,
		"--transport": true,
		"--url":       true,
		"--command":   true,
		"--args":      true,
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

func splitArgsCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func matchesFilter(name, filter string) bool {
	if strings.TrimSpace(filter) == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
}

func displayTransport(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func truncateDescription(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if value == "" || maxLen <= 0 {
		return ""
	}
	if len(value) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return value[:maxLen]
	}
	return value[:maxLen-3] + "..."
}

func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "server already exists:")
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

func containsClient(clients []installer.Tool, target installer.Tool) bool {
	for _, client := range clients {
		if client == target {
			return true
		}
	}
	return false
}

func containsScope(scopes []string, target string) bool {
	for _, scope := range scopes {
		if scope == target {
			return true
		}
	}
	return false
}

func normalizeMcpClients(clients []installer.Tool, clientValue string) ([]installer.Tool, error) {
	allowed := map[installer.Tool]bool{
		installer.ToolClaude:   true,
		installer.ToolCodex:    true,
		installer.ToolGemini:   true,
		installer.ToolOpenCode: true,
	}
	var filtered []installer.Tool
	var unsupported []string
	for _, client := range clients {
		if allowed[client] {
			filtered = append(filtered, client)
			continue
		}
		unsupported = append(unsupported, string(client))
	}
	if len(unsupported) > 0 && strings.TrimSpace(clientValue) != "" && clientValue != "all" {
		return nil, fmt.Errorf("unsupported MCP clients: %s", strings.Join(unsupported, ", "))
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no supported MCP clients selected")
	}
	return filtered, nil
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

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}
