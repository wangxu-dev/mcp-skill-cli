package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func loadJSONConfig(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return map[string]any{}, nil
	}

	clean := stripJSONComments(data)

	var config map[string]any
	if err := json.Unmarshal(clean, &config); err != nil {
		return nil, err
	}

	if config == nil {
		config = map[string]any{}
	}
	return config, nil
}

func writeJSONConfig(path string, config map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ensureMap(parent map[string]any, key string) map[string]any {
	raw, ok := parent[key]
	if !ok {
		child := map[string]any{}
		parent[key] = child
		return child
	}
	if cast, ok := raw.(map[string]any); ok {
		return cast
	}
	child := map[string]any{}
	parent[key] = child
	return child
}

func stripJSONComments(data []byte) []byte {
	var out []byte
	inString := false
	escape := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(data); i++ {
		ch := data[i]

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				out = append(out, ch)
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && i+1 < len(data) && data[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if inString {
			out = append(out, ch)
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			out = append(out, ch)
			continue
		}

		if ch == '/' && i+1 < len(data) {
			next := data[i+1]
			if next == '/' {
				inLineComment = true
				i++
				continue
			}
			if next == '*' {
				inBlockComment = true
				i++
				continue
			}
		}

		out = append(out, ch)
	}

	return out
}
