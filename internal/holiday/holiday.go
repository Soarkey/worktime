package holiday

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/6tail/lunar-go/calendar"
	"github.com/Soarkey/worktime/internal/config"
)

const (
	DefaultURL    = "https://calendars.icloud.com/holidays/cn_zh.ics"
	cacheName     = "holiday.ics"
	refreshWindow = 24 * time.Hour
	expandYears   = 3
)

type DayInfo struct {
	Name    string // 节日/节气名称
	Holiday bool   // 放假
	Makeup  bool   // 调休上班
}

type Calendar struct {
	mu        sync.RWMutex
	holidays  map[string]bool
	makeup    map[string]bool
	names     map[string]string
	loadedAt  time.Time
	cachePath string
}

var defaultCalendar = &Calendar{cachePath: cacheFilePath()}

func Default() *Calendar { return defaultCalendar }

func cacheFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".worktime", cacheName)
}

// EnsureLoaded 加载缓存数据;过期或缺失时静默尝试拉取,失败则保留旧缓存。
func (c *Calendar) EnsureLoaded() {
	c.mu.Lock()
	cached, mtime, err := c.loadCacheLocked()
	if err == nil {
		c.holidays, c.makeup, c.names = cached.holidays, cached.makeup, cached.names
		c.loadedAt = mtime
	}
	needFetch := err != nil || c.loadedAt.IsZero() || time.Since(c.loadedAt) > refreshWindow
	c.mu.Unlock()

	if needFetch {
		go c.Refresh()
	}
}

// Refresh 从配置的接口拉取 ICS 并刷新缓存。
func (c *Calendar) Refresh() error {
	url := c.url()
	data, err := fetch(url)
	if err != nil {
		return fmt.Errorf("获取节假日日历失败: %w", err)
	}
	holidays, makeup, names, err := ParseICS(data)
	if err != nil {
		return fmt.Errorf("解析节假日日历失败: %w", err)
	}

	c.mu.Lock()
	c.holidays, c.makeup, c.names = holidays, makeup, names
	c.loadedAt = time.Now()
	if dir := filepath.Dir(c.cachePath); dir != "" {
		os.MkdirAll(dir, 0755)
		os.WriteFile(c.cachePath, data, 0644)
	}
	c.mu.Unlock()
	return nil
}

func (c *Calendar) url() string {
	wh := config.Load()
	if wh.HolidayURL != "" {
		return wh.HolidayURL
	}
	return DefaultURL
}

type cachedData struct {
	holidays map[string]bool
	makeup   map[string]bool
	names    map[string]string
}

func (c *Calendar) loadCacheLocked() (*cachedData, time.Time, error) {
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		return nil, time.Time{}, err
	}
	h, m, n, err := ParseICS(data)
	if err != nil {
		return nil, time.Time{}, err
	}
	info, err := os.Stat(c.cachePath)
	if err != nil {
		return nil, time.Time{}, err
	}
	return &cachedData{h, m, n}, info.ModTime(), nil
}

// DayInfo 返回某天的节假日/调休信息。
// 无可用数据时按常规周末判断,调用方可用 HasData 判断数据是否加载成功。
func (c *Calendar) DayInfo(d time.Time) DayInfo {
	key := d.Format("2006-01-02")

	c.mu.RLock()
	defer c.mu.RUnlock()

	info := DayInfo{Name: c.names[key]}
	if c.makeup[key] {
		info.Makeup = true
		info.Holiday = false
		return info
	}
	if c.holidays[key] {
		info.Holiday = true
		return info
	}
	info.Holiday = !c.HasDataLocked() && isWeekend(d)
	return info
}

// HasData 是否已加载到节假日数据。
func (c *Calendar) HasData() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.HasDataLocked()
}

func (c *Calendar) HasDataLocked() bool {
	return len(c.holidays) > 0 || len(c.makeup) > 0
}

// IsWorkday 判断某天是否为上班日(考虑调休)。
func (c *Calendar) IsWorkday(d time.Time) bool {
	info := c.DayInfo(d)
	if info.Makeup {
		return true
	}
	if info.Holiday {
		return false
	}
	return !isWeekend(d)
}

func isWeekend(d time.Time) bool {
	return d.Weekday() == time.Saturday || d.Weekday() == time.Sunday
}

// Lunar 返回农历月日,如 "六月廿一"。
func Lunar(d time.Time) string {
	solar := calendar.NewSolarFromYmd(d.Year(), int(d.Month()), d.Day())
	lunar := solar.GetLunar()
	return lunar.GetMonthInChinese() + "月" + lunar.GetDayInChinese()
}

var (
	reFolded   = regexp.MustCompile(`(?m)\r?\n[ \t]`)
	rePropLine = regexp.MustCompile(`^([A-Z\-]+)(;[^:]*)?:(.*)$`)
	reRrule    = regexp.MustCompile(`(?i)COUNT=(\d+)`)
)

func fetch(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("空响应")
	}
	return data, nil
}

// ParseICS 解析节假日 ICS,返回放假日期、调休上班日期、节日/节气名称。
func ParseICS(data []byte) (holidays, makeup map[string]bool, names map[string]string, err error) {
	holidays = make(map[string]bool)
	makeup = make(map[string]bool)
	names = make(map[string]string)

	text := string(data)
	text = reFolded.ReplaceAllString(text, "")

	for _, block := range splitBlocks(text, "BEGIN:VEVENT", "END:VEVENT") {
		props := parseProps(block)
		summary := props["SUMMARY"]
		dtstart := props["DTSTART"]
		dtend := props["DTEND"]
		if dtstart == "" {
			continue
		}
		start, ok := parseDate(dtstart)
		if !ok {
			continue
		}

		special := props["X-APPLE-SPECIAL-DAY"]
		isHoliday := special == "WORK-HOLIDAY" || strings.Contains(summary, "（休）")
		isMakeup := special == "ALTERNATE-WORKDAY" || strings.Contains(summary, "（班）")

		if isHoliday || isMakeup {
			end := start.AddDate(0, 0, 1)
			if d, ok := parseDate(dtend); ok {
				end = d
			}
			marks := holidayMarkers(start, end, props["RRULE"])
			if isHoliday {
				for _, d := range marks {
					holidays[d.Format("2006-01-02")] = true
				}
			}
			if isMakeup {
				for _, d := range marks {
					makeup[d.Format("2006-01-02")] = true
				}
			}
			name := cleanName(summary)
			if name != "" {
				for _, d := range marks {
					names[d.Format("2006-01-02")] = name
				}
			}
			continue
		}

		if summary != "" {
			name := cleanName(summary)
			if name != "" {
				marks := []time.Time{start}
				if rrule := props["RRULE"]; rrule != "" {
					marks = holidayMarkers(start, start.AddDate(0, 0, 1), rrule)
				}
				for _, d := range marks {
					names[d.Format("2006-01-02")] = name
				}
			}
		}
	}
	return holidays, makeup, names, nil
}

// holidayMarkers 展开多天/重复事件为日期列表。
func holidayMarkers(start, end time.Time, rrule string) []time.Time {
	var dates []time.Time
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}
	if m := reRrule.FindStringSubmatch(rrule); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 1 {
			limit := time.Now().AddDate(expandYears, 0, 0)
			base := make([]time.Time, len(dates))
			copy(base, dates)
			for i := 1; i < n; i++ {
				var next []time.Time
				for _, d := range base {
					nd := d.AddDate(i, 0, 0)
					if !nd.After(limit) {
						next = append(next, nd)
					}
				}
				dates = append(dates, next...)
			}
		}
	}
	return dates
}

func cleanName(s string) string {
	s = strings.ReplaceAll(s, "（休）", "")
	s = strings.ReplaceAll(s, "（班）", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if r := []rune(s); len(r) > 6 {
		s = string(r[:6]) + "…"
	}
	return s
}

func splitBlocks(text, begin, end string) []string {
	var blocks []string
	for {
		i := strings.Index(text, begin)
		if i < 0 {
			break
		}
		j := strings.Index(text[i:], end)
		if j < 0 {
			break
		}
		blocks = append(blocks, text[i:i+j])
		text = text[i+j+len(end):]
	}
	return blocks
}

func parseProps(block string) map[string]string {
	props := make(map[string]string)
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "BEGIN") || strings.HasPrefix(line, "END") {
			continue
		}
		m := rePropLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		props[m[1]] = m[3]
	}
	return props
}

func parseDate(s string) (time.Time, bool) {
	for _, layout := range []string{"20060102", "20060102T150405", "20060102T150405Z"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
