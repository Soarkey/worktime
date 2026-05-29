package config

import (
	"os"
	"path/filepath"
	"time"
)

const (
	StandardStartHour = 10
	StandardStartMin  = 0
	StandardEndHour   = 19
	StandardEndMin    = 0

	StartWindowBeginHour = 9
	StartWindowEndHour   = 11

	LeaveWindowBeginHour = 18

	NotifyBeforeMinutes = 10

	PollInterval = 1 * time.Minute
)

func DBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "worktime", "worktime.db")
}

func DBDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "worktime")
}
