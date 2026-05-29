package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	LeaveWindowBeginHour = 18
	PollInterval         = 1 * time.Minute
)

type WorkHours struct {
	StartHour int `json:"start_hour"`
	StartMin  int `json:"start_min"`
	EndHour   int `json:"end_hour"`
	EndMin    int `json:"end_min"`
}

var defaultWorkHours = WorkHours{
	StartHour: 10,
	StartMin:  0,
	EndHour:   19,
	EndMin:    0,
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".worktime")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func Load() WorkHours {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return defaultWorkHours
	}
	var wh WorkHours
	if err := json.Unmarshal(data, &wh); err != nil {
		return defaultWorkHours
	}
	return wh
}

func Save(wh WorkHours) error {
	if err := os.MkdirAll(configDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(wh, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}
