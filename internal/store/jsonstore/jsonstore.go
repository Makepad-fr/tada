package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/idilsaglam/todo/internal/model"
)

// JSON-backed storage. Single file, human-readable, portable.
// No locking for v1; fine for a local single-user CLI.

const dataFileName = "todos.json"

func dataPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	return filepath.Join(wd, dataFileName), nil
}

func Load() ([]model.Item, error) {
	p, err := dataPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []model.Item{}, nil
		}
		return nil, fmt.Errorf("read file: %w", err)
	}
	var items []model.Item
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	return items, nil
}

func Save(items []model.Item) error {
	p, err := dataPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
