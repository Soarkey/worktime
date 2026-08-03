package menubar

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/holiday"
	"github.com/Soarkey/worktime/internal/parser"
	"github.com/energye/systray"
)

const maxDaysInMonth = 31

// buildCalendarMenu 构建日历子菜单:月份标题、翻月导航、每天一个菜单项。
func (m *MenuBar) buildCalendarMenu() {
	m.mCalendar = systray.AddMenuItem("日历", "查看农历与节假日")

	m.calTitle = m.mCalendar.AddSubMenuItem("--", "")
	m.calTitle.Disable()
	m.calPrev = m.mCalendar.AddSubMenuItem("◀ 上月", "")
	m.calPrev.Click(func() {
		m.calMonth = m.calMonth.AddDate(0, -1, 0)
		m.renderCalendar()
	})
	m.calNext = m.mCalendar.AddSubMenuItem("下月 ▶", "")
	m.calNext.Click(func() {
		m.calMonth = m.calMonth.AddDate(0, 1, 0)
		m.renderCalendar()
	})

	m.calDays = make([]*systray.MenuItem, maxDaysInMonth)
	for i := range m.calDays {
		item := m.mCalendar.AddSubMenuItem("--", "")
		item.Hide()
		item.Click(func(idx int) func() {
			return func() { go m.showDayDetailDialog(idx) }
		}(i))
		m.calDays[i] = item
	}

	setting := m.mCalendar.AddSubMenuItem("设置节假日接口...", "配置节假日日历 (ICS) 地址")
	setting.Click(func() { go m.showHolidayURLDialog() })

	m.calMonth = time.Now()
	m.renderCalendar()
	holiday.Default().EnsureLoaded()
}

func (m *MenuBar) renderCalendar() {
	cal := holiday.Default()
	month := m.calMonth
	now := time.Now()

	m.calTitle.SetTitle(fmt.Sprintf("%d年%d月", month.Year(), int(month.Month())))

	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local)
	daysInMonth := first.AddDate(0, 1, 0).AddDate(0, 0, -1).Day()

	for i := 0; i < maxDaysInMonth; i++ {
		item := m.calDays[i]
		if i >= daysInMonth {
			item.Hide()
			continue
		}
		day := first.AddDate(0, 0, i)
		info := cal.DayInfo(day)

		mark := " 班"
		if info.Holiday {
			mark = " 休"
		} else if info.Makeup {
			mark = " 班"
		} else if !cal.IsWorkday(day) {
			mark = " 休"
		}
		if info.Name != "" && !info.Holiday && !info.Makeup {
			mark = " " + info.Name
		}

		title := fmt.Sprintf("%s %s %s%s", day.Format("01-02"), weekdayNames[day.Weekday()], holiday.Lunar(day), mark)
		if sameDay(day, now) {
			title = "● " + title
		}
		item.SetTitle(title)
		item.SetTooltip("点击查看当天详情")
		item.Show()
	}
}

func (m *MenuBar) showDayDetailDialog(idx int) {
	if idx < 0 || idx >= maxDaysInMonth {
		return
	}
	first := time.Date(m.calMonth.Year(), m.calMonth.Month(), 1, 0, 0, 0, 0, time.Local)
	day := first.AddDate(0, 0, idx)
	cal := holiday.Default()
	info := cal.DayInfo(day)

	var lines []string
	lines = append(lines, fmt.Sprintf("%s %s", day.Format("2006-01-02"), weekdayNames[day.Weekday()]))
	lines = append(lines, "农历: "+holiday.Lunar(day))

	switch {
	case info.Makeup:
		lines = append(lines, "类型: 工作日（调休上班）")
	case info.Holiday:
		lines = append(lines, "类型: 休息日")
		if info.Name != "" {
			lines = append(lines, "节日: "+info.Name)
		}
	case info.Name != "":
		lines = append(lines, "节日: "+info.Name)
	default:
		lines = append(lines, "类型: 工作日")
	}

	lines = append(lines, m.dayWorkTime(day))
	msg := strings.Join(lines, "\n")

	script := fmt.Sprintf(
		`display dialog "%s" with title "%s" buttons {"确定"} default button "确定"`,
		osaEscape(msg), osaEscape(fmt.Sprintf("worktime 日历 · %d月%d日", m.calMonth.Month(), day.Day())),
	)
	exec.Command("osascript", "-e", script).Run()
}

// dayWorkTime 返回某天的上下班时间描述。
func (m *MenuBar) dayWorkTime(day time.Time) string {
	wh := config.Load()
	standard := fmt.Sprintf("上班: %02d:%02d\n预计下班: %02d:%02d", wh.StartHour, wh.StartMin, wh.EndHour, wh.EndMin)

	if events, err := parser.GetParsedLog(); err == nil {
		dateStr := day.Format("2006-01-02")
		if s := attendance.GetByDate(dateStr, events, wh.RangeBegin(), wh.RangeEnd()); s != nil {
			line := fmt.Sprintf("上班: %s\n预计下班: %s", s.StartTime, s.ExpectedLeave)
			if s.ActualLeave != "" {
				line += "\n实际下班: " + s.ActualLeave
			}
			if s.LateMinutes > 0 {
				line += fmt.Sprintf("\n延迟: %d 分钟", s.LateMinutes)
			}
			return line
		}
		if _, ok := events[dateStr]; !ok {
			return standard + "\n（无考勤记录）"
		}
	}
	return standard
}

func (m *MenuBar) showHolidayURLDialog() {
	wh := config.Load()
	current := wh.HolidayURL
	if current == "" {
		current = holiday.DefaultURL
	}
	url, ok := dialogInput("worktime 设置", "请输入节假日日历 (ICS) 接口地址:", current)
	if !ok {
		return
	}
	if url == "" || url == holiday.DefaultURL {
		wh.HolidayURL = ""
	} else {
		wh.HolidayURL = url
	}
	if err := config.Save(wh); err != nil {
		return
	}
	if err := holiday.Default().Refresh(); err != nil {
		return
	}
	m.renderCalendar()
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
