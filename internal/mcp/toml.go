package mcp

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type tomlBlock struct {
	kind  string
	name  string
	lines []string
}

func parseTomlBlocks(path string) ([]tomlBlock, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []tomlBlock{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var blocks []tomlBlock
	current := tomlBlock{kind: "other"}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		table := parseTomlTable(line)
		if table != "" {
			if current.kind == "mcp" && isMcpSubtable(current.name, table) {
				current.lines = append(current.lines, line)
				continue
			}

			if len(current.lines) > 0 {
				blocks = append(blocks, current)
			}
			if isMcpTable(table) {
				name := strings.TrimPrefix(table, "mcp_servers.")
				current = tomlBlock{kind: "mcp", name: name, lines: []string{line}}
			} else {
				current = tomlBlock{kind: "other", lines: []string{line}}
			}
			continue
		}

		current.lines = append(current.lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(current.lines) > 0 {
		blocks = append(blocks, current)
	}
	return blocks, nil
}

func writeTomlBlocks(path string, blocks []tomlBlock) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var lines []string
	for _, block := range blocks {
		lines = append(lines, block.lines...)
	}
	data := strings.Join(lines, "\n")
	if len(data) > 0 && !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	return os.WriteFile(path, []byte(data), 0o644)
}

func parseTomlTable(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") {
		return ""
	}
	end := strings.Index(trimmed, "]")
	if end == -1 {
		return ""
	}
	table := strings.TrimSpace(trimmed[1:end])
	return table
}

func isMcpTable(table string) bool {
	return strings.HasPrefix(table, "mcp_servers.")
}

func isMcpSubtable(name, table string) bool {
	prefix := "mcp_servers." + name + "."
	return strings.HasPrefix(table, prefix)
}

func formatTomlEntry(def Definition) []string {
	lines := []string{fmt.Sprintf("[mcp_servers.%s]", def.Name)}
	if def.Transport == "http" {
		lines = append(lines, fmt.Sprintf("url = %q", def.URL))
		return lines
	}

	lines = append(lines, fmt.Sprintf("command = %q", def.Command))
	if len(def.Args) > 0 {
		lines = append(lines, fmt.Sprintf("args = [%s]", joinTomlArray(def.Args)))
	}
	if len(def.Env) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("[mcp_servers.%s.env]", def.Name))
		for key, value := range def.Env {
			lines = append(lines, fmt.Sprintf("%s = %q", key, value))
		}
	}
	return lines
}

func joinTomlArray(values []string) string {
	escaped := make([]string, 0, len(values))
	for _, value := range values {
		escaped = append(escaped, fmt.Sprintf("%q", value))
	}
	return strings.Join(escaped, ", ")
}
