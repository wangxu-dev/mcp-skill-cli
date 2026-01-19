package mcpcli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"mcp-skill-manager/internal/mcp"
	"mcp-skill-manager/internal/registryindex"
)

func printMcpEntry(out io.Writer, entry registryindex.MCPEntry) {
	fmt.Fprintf(out, "name: %s\n", entry.Name)
	if entry.Type != "" {
		fmt.Fprintf(out, "type: %s\n", entry.Type)
	}
	if entry.Description != "" {
		fmt.Fprintf(out, "description: %s\n", entry.Description)
	}
	if entry.UpdatedAt != "" {
		fmt.Fprintf(out, "updatedAt: %s\n", entry.UpdatedAt)
	}
	if entry.Type == "http" && entry.URL != "" {
		fmt.Fprintf(out, "url: %s\n", entry.URL)
	}
	if entry.Type == "stdio" && entry.Repo != "" {
		fmt.Fprintf(out, "repo: %s\n", entry.Repo)
	}
}

func printDefinition(out io.Writer, def mcp.Definition) {
	if def.Transport != "" {
		fmt.Fprintf(out, "transport: %s\n", def.Transport)
	}
	if def.URL != "" {
		fmt.Fprintf(out, "url: %s\n", def.URL)
	}
	if def.Command != "" {
		fmt.Fprintf(out, "command: %s\n", def.Command)
	}
	if len(def.Args) > 0 {
		fmt.Fprintf(out, "args: %s\n", strings.Join(def.Args, " "))
	}
	if len(def.Env) > 0 {
		fmt.Fprintln(out, "env:")
		keys := make([]string, 0, len(def.Env))
		for key := range def.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(out, "  %s=%s\n", key, def.Env[key])
		}
	}
	if len(def.Headers) > 0 {
		fmt.Fprintln(out, "headers:")
		keys := make([]string, 0, len(def.Headers))
		for key := range def.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(out, "  %s: %s\n", key, def.Headers[key])
		}
	}
}
