package parser

import (
	"testing"
	"time"
)

func TestParseLines(t *testing.T) {
	raw := `2026-05-29 10:12:35 +0800 Notification loginwindow logged in
2026-05-29 10:15:00 +0800 Notification com.apple.powermanagement.lidopen
2026-05-29 19:30:00 +0800 Notification        	Display is turned off
2026-05-29 22:14:00 +0800 Sleep               	Entering Sleep state due to 'Clamshell Sleep':TCPKeepAlive=active
2026-05-28 09:58:00 +0800 Notification loginwindow logged in
2026-05-28 19:05:00 +0800 Notification        	Display is turned off
`
	events := parseLines(raw)
	if len(events) != 2 {
		t.Fatalf("expected 2 dates, got %d", len(events))
	}

	// 2026-05-29: start at 10:12, leave at 22:14
	day29 := events["2026-05-29"]
	start := FindStartTime(day29)
	if start == nil {
		t.Fatal("expected start time for 2026-05-29")
	}
	if start.Hour() != 10 || start.Minute() != 12 {
		t.Errorf("expected 10:12, got %02d:%02d", start.Hour(), start.Minute())
	}

	leave := FindLeaveTime(day29)
	if leave == nil {
		t.Fatal("expected leave time for 2026-05-29")
	}
	if leave.Hour() != 22 || leave.Minute() != 14 {
		t.Errorf("expected 22:14, got %02d:%02d", leave.Hour(), leave.Minute())
	}

	// 2026-05-28: start at 09:58
	day28 := events["2026-05-28"]
	start28 := FindStartTime(day28)
	if start28 == nil {
		t.Fatal("expected start time for 2026-05-28")
	}
	if start28.Hour() != 9 || start28.Minute() != 58 {
		t.Errorf("expected 09:58, got %02d:%02d", start28.Hour(), start28.Minute())
	}
}

func TestFindStartTimeOutOfWindow(t *testing.T) {
	events := []Event{
		{Time: time.Date(2026, 5, 29, 8, 30, 0, 0, time.Local), Type: "start"},
		{Time: time.Date(2026, 5, 29, 11, 30, 0, 0, time.Local), Type: "start"},
	}
	start := FindStartTime(events)
	if start != nil {
		t.Errorf("expected nil (both out of 9-11 window), got %v", start)
	}
}
