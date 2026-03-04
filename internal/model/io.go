package model

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func ReadSnapshot(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{}, err
	}

	return snap, nil
}

func WriteSnapshot(path string, snap Snapshot, pretty bool) error {
	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(snap, "", "  ")
	} else {
		data, err = json.Marshal(snap)
	}
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	return os.WriteFile(path, data, 0o644)
}
