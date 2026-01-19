package registryindex

import (
	"encoding/json"
	"os"
	"path/filepath"

	"mcp-skill-manager/internal/installer"
)

type LocalRecord struct {
	Name      string `json:"name"`
	Repo      string `json:"repo"`
	Path      string `json:"path"`
	Head      string `json:"head"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

func LoadLocalRecord(kind, name string) (LocalRecord, bool, error) {
	path, err := localRecordPath(kind, name)
	if err != nil {
		return LocalRecord{}, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LocalRecord{}, false, nil
		}
		return LocalRecord{}, false, err
	}
	var record LocalRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return LocalRecord{}, false, err
	}
	return record, true, nil
}

func LocalRecordFor(kind, name string) (LocalRecord, bool, error) {
	return LoadLocalRecord(kind, name)
}

func SaveLocalRecord(kind string, record LocalRecord) error {
	path, err := localRecordPath(kind, record.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func localRecordPath(kind, name string) (string, error) {
	root, err := installer.LocalStoreRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".meta", kind, name+".json"), nil
}
