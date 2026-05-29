package attendance

import (
	"fmt"
	"time"

	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/parser"
	"github.com/Soarkey/worktime/internal/storage"
)

type Status struct {
	WorkDate          string
	StartTime         string
	ExpectedLeave     string
	ActualLeave       string
	LateMinutes       int
	RemainingMinutes  int
	State             string // "working", "soon", "off"
}

func Calculate(startTime time.Time) Status {
	standardStart := time.Date(startTime.Year(), startTime.Month(), startTime.Day(),
		config.StandardStartHour, config.StandardStartMin, 0, 0, time.Local)
	standardEnd := time.Date(startTime.Year(), startTime.Month(), startTime.Day(),
		config.StandardEndHour, config.StandardEndMin, 0, 0, time.Local)

	lateDur := startTime.Sub(standardStart)
	if lateDur < 0 {
		lateDur = 0
	}
	lateMinutes := int((lateDur + 59*time.Second) / time.Minute)

	expectedLeave := standardEnd.Add(lateDur)

	now := time.Now()
	remaining := expectedLeave.Sub(now)
	remainingMin := int(remaining.Minutes())

	state := "working"
	if remainingMin <= 0 {
		state = "off"
	} else if remainingMin <= 30 {
		state = "soon"
	}

	return Status{
		WorkDate:         startTime.Format("2006-01-02"),
		StartTime:        startTime.Format("15:04"),
		ExpectedLeave:    expectedLeave.Format("15:04"),
		LateMinutes:      lateMinutes,
		RemainingMinutes: remainingMin,
		State:            state,
	}
}

func MenuBarTitle(s Status) string {
	switch s.State {
	case "off":
		return "🔴 下班"
	case "soon":
		h := s.RemainingMinutes / 60
		m := s.RemainingMinutes % 60
		return fmt.Sprintf("🟡 %02d:%02d", h, m)
	default:
		return fmt.Sprintf("🟢 %s", s.ExpectedLeave)
	}
}

func Sync(store *storage.Store) (*Status, error) {
	today := time.Now().Format("2006-01-02")

	events, err := parser.ParsePmsetLog()
	if err != nil {
		return nil, fmt.Errorf("parse pmset: %w", err)
	}

	todayEvents := events[today]
	startTime := parser.FindStartTime(todayEvents)
	if startTime == nil {
		existing, _ := store.GetByDate(today)
		if existing != nil && existing.StartTime != "" {
			t, _ := time.ParseInLocation("2006-01-02 15:04", today+" "+existing.StartTime, time.Local)
			status := Calculate(t)
			status.ActualLeave = existing.ActualLeaveTime
			return &status, nil
		}
		return nil, nil
	}

	status := Calculate(*startTime)

	if err := store.UpsertStart(today, status.StartTime, status.ExpectedLeave, status.LateMinutes); err != nil {
		return nil, fmt.Errorf("upsert start: %w", err)
	}

	leaveTime := parser.FindLeaveTime(todayEvents)
	if leaveTime != nil {
		status.ActualLeave = leaveTime.Format("15:04")
		if err := store.UpdateActualLeave(today, status.ActualLeave); err != nil {
			return nil, fmt.Errorf("update leave: %w", err)
		}
	}

	return &status, nil
}
