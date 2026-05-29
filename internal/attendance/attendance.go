package attendance

import (
	"fmt"
	"time"

	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/parser"
)

type Status struct {
	WorkDate         string
	StartTime        string
	ExpectedLeave    string
	ActualLeave      string
	LateMinutes      int
	RemainingMinutes int
	State            string // "working", "soon", "off"
}

func Calculate(startTime time.Time) Status {
	wh := config.Load()
	standardStart := time.Date(startTime.Year(), startTime.Month(), startTime.Day(),
		wh.StartHour, wh.StartMin, 0, 0, time.Local)
	standardEnd := time.Date(startTime.Year(), startTime.Month(), startTime.Day(),
		wh.EndHour, wh.EndMin, 0, 0, time.Local)

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
		return "下班"
	case "soon":
		h := s.RemainingMinutes / 60
		m := s.RemainingMinutes % 60
		return fmt.Sprintf("%02d:%02d", h, m)
	default:
		return s.ExpectedLeave
	}
}

func GetToday() (*Status, error) {
	events, err := parser.ParsePmsetLog()
	if err != nil {
		return nil, fmt.Errorf("parse pmset: %w", err)
	}

	wh := config.Load()
	today := time.Now().Format("2006-01-02")
	todayEvents := events[today]
	startTime := parser.FindStartTime(todayEvents, wh.StartHour)
	if startTime == nil {
		return nil, nil
	}

	status := Calculate(*startTime)

	leaveTime := parser.FindLeaveTime(todayEvents)
	if leaveTime != nil {
		status.ActualLeave = leaveTime.Format("15:04")
	}

	return &status, nil
}

func GetByDate(date string, events map[string][]parser.Event, startHour int) *Status {
	dayEvents := events[date]
	startTime := parser.FindStartTime(dayEvents, startHour)
	if startTime == nil {
		return nil
	}

	status := Calculate(*startTime)

	leaveTime := parser.FindLeaveTime(dayEvents)
	if leaveTime != nil {
		status.ActualLeave = leaveTime.Format("15:04")
	}

	return &status
}

func GetWeek() ([]Status, error) {
	events, err := parser.ParsePmsetLog()
	if err != nil {
		return nil, fmt.Errorf("parse pmset: %w", err)
	}

	wh := config.Load()
	now := time.Now()
	offset := int(now.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	monday := now.AddDate(0, 0, -offset)

	var results []Status
	for i := 0; i < 7; i++ {
		day := monday.AddDate(0, 0, i)
		if day.After(now) {
			break
		}
		dateStr := day.Format("2006-01-02")
		if s := GetByDate(dateStr, events, wh.StartHour); s != nil {
			results = append(results, *s)
		}
	}
	return results, nil
}

func GetAll() ([]Status, error) {
	events, err := parser.ParsePmsetLog()
	if err != nil {
		return nil, fmt.Errorf("parse pmset: %w", err)
	}

	wh := config.Load()
	var results []Status
	for date := range events {
		if s := GetByDate(date, events, wh.StartHour); s != nil {
			results = append(results, *s)
		}
	}
	return results, nil
}
