package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type OverrideEntry struct {
	Hour int `json:"hour"`
	Min  int `json:"min"`
}

type OverrideData map[string]OverrideEntry

func overridePath() string {
	return filepath.Join(configDir(), "override.json")
}

func LoadOverride(date string) *OverrideEntry {
	data, err := os.ReadFile(overridePath())
	if err != nil {
		return nil
	}
	var overrides OverrideData
	if err := json.Unmarshal(data, &overrides); err != nil {
		return nil
	}
	entry, ok := overrides[date]
	if !ok {
		return nil
	}
	return &entry
}

func SaveOverride(date string, hour, min int) error {
	var overrides OverrideData
	data, err := os.ReadFile(overridePath())
	if err == nil {
		json.Unmarshal(data, &overrides)
	}
	if overrides == nil {
		overrides = make(OverrideData)
	}
	overrides[date] = OverrideEntry{Hour: hour, Min: min}

	if err := os.MkdirAll(configDir(), 0755); err != nil {
		return err
	}
	data, err = json.MarshalIndent(overrides, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(overridePath(), data, 0644)
}
