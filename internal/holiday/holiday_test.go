package holiday

import (
	"strings"
	"testing"
	"time"
)

const sampleICS = `BEGIN:VCALENDAR
VERSION:2.0
CALSCALE:GREGORIAN
X-WR-CALNAME:中国大陆节假日
BEGIN:VEVENT
DTSTAMP;VALUE=DATE:19760401
UID:holiday-newyear
DTSTART;VALUE=DATE:20260101
DTEND;VALUE=DATE:20260104
SUMMARY;LANGUAGE=zh_CN:元旦（休）
TRANSP:TRANSPARENT
X-APPLE-SPECIAL-DAY:WORK-HOLIDAY
END:VEVENT
BEGIN:VEVENT
DTSTAMP;VALUE=DATE:19760401
UID:work-chunjie
DTSTART;VALUE=DATE:20260214
SUMMARY;LANGUAGE=zh_CN:春节（班）
TRANSP:TRANSPARENT
X-APPLE-SPECIAL-DAY:ALTERNATE-WORKDAY
END:VEVENT
BEGIN:VEVENT
DTSTAMP;VALUE=DATE:19760401
UID:holiday-chunjie
DTSTART;VALUE=DATE:20260215
DTEND;VALUE=DATE:20260224
SUMMARY;LANGUAGE=zh_CN:春节（休）
TRANSP:TRANSPARENT
X-APPLE-SPECIAL-DAY:WORK-HOLIDAY
END:VEVENT
BEGIN:VEVENT
DTSTAMP;VALUE=DATE:19760401
UID:fest-qixi
DTSTART;VALUE=DATE:20260819
SUMMARY;LANGUAGE=zh_CN:七夕
END:VEVENT
BEGIN:VEVENT
DTSTAMP;VALUE=DATE:19760401
UID:rrule-newyear-2027
DTSTART;VALUE=DATE:20270101
SUMMARY;LANGUAGE=zh_CN:元旦
RRULE:FREQ=YEARLY;COUNT=3
END:VEVENT
END:VCALENDAR
`

func TestParseICS(t *testing.T) {
	holidays, makeup, names, err := ParseICS([]byte(sampleICS))
	if err != nil {
		t.Fatalf("ParseICS: %v", err)
	}

	cases := []struct {
		date      string
		holiday   bool
		makeup    bool
		wantNames string
	}{
		{"2026-01-01", true, false, "元旦"},
		{"2026-01-03", true, false, "元旦"},
		{"2026-01-04", false, false, ""}, // DTEND 排除
		{"2026-02-14", false, true, "春节"},
		{"2026-02-15", true, false, "春节"},
		{"2026-02-23", true, false, "春节"},
		{"2026-02-24", false, false, ""},
		{"2026-08-19", false, false, "七夕"},
	}
	for _, c := range cases {
		if holidays[c.date] != c.holiday {
			t.Errorf("%s holiday = %v, want %v", c.date, holidays[c.date], c.holiday)
		}
		if makeup[c.date] != c.makeup {
			t.Errorf("%s makeup = %v, want %v", c.date, makeup[c.date], c.makeup)
		}
		if names[c.date] != c.wantNames {
			t.Errorf("%s name = %q, want %q", c.date, names[c.date], c.wantNames)
		}
	}

	if names["2027-01-01"] != "元旦" {
		t.Errorf("RRULE 未展开到 2027-01-01: %q", names["2027-01-01"])
	}
	if names["2028-01-01"] != "元旦" {
		t.Errorf("RRULE 未展开到 2028-01-01: %q", names["2028-01-01"])
	}
	if names["2029-01-01"] != "元旦" {
		t.Error("RRULE COUNT=3 应展开到 2029")
	}
	if names["2030-01-01"] != "" {
		t.Error("RRULE COUNT=3 不应展开到 2030")
	}
}

func TestDayInfo(t *testing.T) {
	cal := &Calendar{cachePath: "/nonexistent"}
	h, m, n, _ := ParseICS([]byte(sampleICS))
	cal.holidays, cal.makeup, cal.names = h, m, n

	cases := []struct {
		date       string
		workday    bool
		holiday    bool
		makeup     bool
		weekendDay bool
	}{
		{"2026-01-01", false, true, false, false}, // 元旦放假(周四)
		{"2026-02-14", true, false, true, true},   // 春节调休上班(周六)
		{"2026-02-16", false, true, false, false}, // 春节放假(周一)
		{"2026-08-16", false, false, false, true}, // 普通周末(周日)
		{"2026-08-19", true, false, false, false}, // 普通工作日(周三)
	}
	for _, c := range cases {
		d, _ := time.Parse("2006-01-02", c.date)
		if got := cal.IsWorkday(d); got != c.workday {
			t.Errorf("%s IsWorkday = %v, want %v", c.date, got, c.workday)
		}
		info := cal.DayInfo(d)
		if info.Holiday != c.holiday {
			t.Errorf("%s Holiday = %v, want %v", c.date, info.Holiday, c.holiday)
		}
		if info.Makeup != c.makeup {
			t.Errorf("%s Makeup = %v, want %v", c.date, info.Makeup, c.makeup)
		}
	}
}

func TestLunar(t *testing.T) {
	d, _ := time.Parse("2006-01-02", "2026-08-19")
	got := Lunar(d)
	if !strings.Contains(got, "月") || !strings.Contains(got, "初") && !strings.Contains(got, "廿") {
		t.Errorf("Lunar(2026-08-19) = %q, 期望农历月日", got)
	}
}
