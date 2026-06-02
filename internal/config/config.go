package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type WorkHours struct {
	StartHour int `json:"start_hour"`
	StartMin  int `json:"start_min"`
	EndHour   int `json:"end_hour"`
	EndMin    int `json:"end_min"`

	RangeBeginHour int `json:"range_begin_hour,omitempty"`
	RangeBeginMin  int `json:"range_begin_min,omitempty"`
	RangeEndHour   int `json:"range_end_hour,omitempty"`
	RangeEndMin    int `json:"range_end_min,omitempty"`
}

var defaultWorkHours = WorkHours{
	StartHour: 10,
	StartMin:  0,
	EndHour:   19,
	EndMin:    0,
}

func (wh WorkHours) RangeBegin() int {
	if wh.RangeBeginHour == 0 && wh.RangeBeginMin == 0 {
		return (wh.StartHour - 2) * 60
	}
	return wh.RangeBeginHour*60 + wh.RangeBeginMin
}

func (wh WorkHours) RangeEnd() int {
	if wh.RangeEndHour == 0 && wh.RangeEndMin == 0 {
		return (wh.StartHour + 2) * 60
	}
	return wh.RangeEndHour*60 + wh.RangeEndMin
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
