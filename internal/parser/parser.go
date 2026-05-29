package parser

import (
	"bufio"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Soarkey/worktime/internal/config"
)

type Event struct {
	Time time.Time
	Type string
}

var startPatterns = regexp.MustCompile(`(?i)(loginwindow|com\.apple\.powermanagement\.lidopen|Wake from|wake from sleep)`)
var leavePatterns = regexp.MustCompile(`(?i)(Display is turned off|Clamshell Sleep)`)
var timestampRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2}:\d{2})`)

func ParsePmsetLog() (map[string][]Event, error) {
	out, err := exec.Command("pmset", "-g", "log").Output()
	if err != nil {
		return nil, err
	}
	return parseLines(string(out)), nil
}

func parseLines(raw string) map[string][]Event {
	events := make(map[string][]Event)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		matches := timestampRe.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}
		dateStr := matches[1]
		timeStr := matches[2]

		t, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr+" "+timeStr, time.Local)
		if err != nil {
			continue
		}

		if startPatterns.MatchString(line) {
			events[dateStr] = append(events[dateStr], Event{Time: t, Type: "start"})
		} else if leavePatterns.MatchString(line) {
			events[dateStr] = append(events[dateStr], Event{Time: t, Type: "leave"})
		}
	}
	return events
}

func FindStartTime(events []Event) *time.Time {
	for _, e := range events {
		if e.Type != "start" {
			continue
		}
		h := e.Time.Hour()
		if h >= config.StartWindowBeginHour && h < config.StartWindowEndHour {
			t := e.Time
			return &t
		}
	}
	return nil
}

func FindLeaveTime(events []Event) *time.Time {
	var last *time.Time
	for _, e := range events {
		if e.Type != "leave" {
			continue
		}
		if e.Time.Hour() >= config.LeaveWindowBeginHour {
			t := e.Time
			last = &t
		}
	}
	return last
}
