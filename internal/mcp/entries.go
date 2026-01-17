package mcp

import (
	"sort"
	"strings"
)

type Entry struct {
	Name      string
	Transport string
}

func extractEntries(servers map[string]any) []Entry {
	entries := make([]Entry, 0, len(servers))
	for name, raw := range servers {
		entry := Entry{Name: name}
		if server, ok := raw.(map[string]any); ok {
			entry.Transport = detectTransport(server)
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries
}

func detectTransport(server map[string]any) string {
	if value, ok := server["type"].(string); ok {
		switch strings.ToLower(value) {
		case "http", "stdio":
			return strings.ToLower(value)
		case "local":
			return "stdio"
		case "remote":
			return "http"
		}
	}
	if _, ok := server["url"]; ok {
		return "http"
	}
	if _, ok := server["command"]; ok {
		return "stdio"
	}
	return ""
}
