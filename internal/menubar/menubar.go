package menubar

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/energye/systray"
	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/brewservice"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/parser"
)

type MenuBar struct {
	version string

	mStatus *systray.MenuItem
	mToday     *systray.MenuItem
	todayDate  *systray.MenuItem
	todayStart *systray.MenuItem
	todayEnd   *systray.MenuItem
	todayLate  *systray.MenuItem
	todayLeave *systray.MenuItem
	todayModify *systray.MenuItem
	mWeek     *systray.MenuItem
	weekItems []*systray.MenuItem
	weekSumm  *systray.MenuItem
	mExport      *systray.MenuItem
	mConfig      *systray.MenuItem
	mConfigRange *systray.MenuItem
	mAutoStart   *systray.MenuItem
	mQuit        *systray.MenuItem
}

func New(version string) *MenuBar {
	return &MenuBar{version: version}
}

func (m *MenuBar) Run() {
	systray.Run(m.onReady, nil)
}

func (m *MenuBar) onReady() {
	systray.SetTitle("⏳")
	systray.SetTooltip("worktime")

	systray.SetOnClick(func(menu systray.IMenu) {
		go m.refresh()
		menu.ShowMenu()
	})
	systray.SetOnRClick(func(menu systray.IMenu) { menu.ShowMenu() })

	m.mStatus = systray.AddMenuItem("加载中...", "当前状态")
	m.mStatus.Disable()

	systray.AddSeparator()

	m.mToday = systray.AddMenuItem("今日详情", "")
	m.todayDate = m.mToday.AddSubMenuItem("日期: --", "")
	m.todayDate.Disable()
	m.todayStart = m.mToday.AddSubMenuItem("上班: --", "")
	m.todayStart.Disable()
	m.todayEnd = m.mToday.AddSubMenuItem("预计下班: --", "")
	m.todayEnd.Disable()
	m.todayLate = m.mToday.AddSubMenuItem("延迟: --", "")
	m.todayLate.Disable()
	m.todayLeave = m.mToday.AddSubMenuItem("实际下班: --", "")
	m.todayLeave.Disable()

	m.mWeek = systray.AddMenuItem("本周统计", "")
	m.weekItems = make([]*systray.MenuItem, 7)
	for i := range m.weekItems {
		m.weekItems[i] = m.mWeek.AddSubMenuItem("--", "")
		m.weekItems[i].Disable()
	}
	m.weekSumm = m.mWeek.AddSubMenuItem("--", "")
	m.weekSumm.Disable()

	m.todayModify = systray.AddMenuItem("修改今日上班时间...", "手动修正今日上班开始时间")
	m.todayModify.Click(func() { go m.showModifyTodayDialog() })

	systray.AddSeparator()

	m.mExport = systray.AddMenuItem("导出 CSV...", "导出考勤记录")
	m.mExport.Click(func() { go exportCSV() })

	wh := config.Load()
	m.mConfig = systray.AddMenuItem(fmt.Sprintf("设置 (上班 %02d:%02d / 下班 %02d:%02d)", wh.StartHour, wh.StartMin, wh.EndHour, wh.EndMin), "设置上下班时间")
	m.mConfig.Click(func() { go m.showConfigDialog() })

	rbH, rbM := wh.RangeBegin()/60, wh.RangeBegin()%60
	reH, reM := wh.RangeEnd()/60, wh.RangeEnd()%60
	m.mConfigRange = systray.AddMenuItem(fmt.Sprintf("设置检测时间段 (%02d:%02d-%02d:%02d)", rbH, rbM, reH, reM), "设置上班检测时间段")
	m.mConfigRange.Click(func() { go m.showConfigRangeDialog() })

	if brewservice.IsRunning() {
		m.mAutoStart = systray.AddMenuItem("开机启动: 已开启", "点击关闭开机启动")
	} else {
		m.mAutoStart = systray.AddMenuItem("开机启动: 已关闭", "点击开启开机启动")
	}
	m.mAutoStart.Click(func() { go m.toggleAutoStart() })

	systray.AddSeparator()

	m.mQuit = systray.AddMenuItem("退出", "退出 worktime")
	m.mQuit.Click(func() {
		brewservice.Stop()
		systray.Quit()
	})

	m.buildVersionSubmenu()
}

func (m *MenuBar) refresh() {
	attendance.ClearCache()
	if status, err := attendance.GetToday(); err == nil {
		m.Update(status)
	}
}

func (m *MenuBar) Update(status *attendance.Status) {
	if status == nil {
		systray.SetTitle("⏳ 未检测")
		m.mStatus.SetTitle("未检测到上班时间")
		return
	}

	systray.SetTitle(attendance.MenuBarTitle(*status))

	detail := fmt.Sprintf("上班: %s | 预计下班: %s", status.StartTime, status.ExpectedLeave)
	if status.LateMinutes > 0 {
		detail += fmt.Sprintf(" | 延迟 %d 分钟", status.LateMinutes)
	}
	m.mStatus.SetTitle(detail)

	m.todayDate.SetTitle(fmt.Sprintf("日期: %s", status.WorkDate))
	m.todayStart.SetTitle(fmt.Sprintf("上班: %s", status.StartTime))
	m.todayEnd.SetTitle(fmt.Sprintf("预计下班: %s", status.ExpectedLeave))
	m.todayLate.SetTitle(fmt.Sprintf("延迟: %d 分钟", status.LateMinutes))
	leave := status.ActualLeave
	if leave == "" {
		leave = "--"
	}
	m.todayLeave.SetTitle(fmt.Sprintf("实际下班: %s", leave))

	m.refreshWeek()
}

var weekdayNames = [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

func (m *MenuBar) refreshWeek() {
	records, err := attendance.GetWeek()
	if err != nil {
		return
	}

	recordMap := make(map[string]attendance.Status, len(records))
	for _, r := range records {
		recordMap[r.WorkDate] = r
	}

	now := time.Now()
	offset := int(now.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	mondayTime := now.AddDate(0, 0, -offset)

	lateCount := 0
	dayCount := 0

	for i := 0; i < 7; i++ {
		day := mondayTime.AddDate(0, 0, i)
		dateStr := day.Format("2006-01-02")
		wdName := weekdayNames[day.Weekday()]

		if r, ok := recordMap[dateStr]; ok {
			dayCount++
			if r.LateMinutes > 0 {
				lateCount++
			}
			actual := r.ActualLeave
			if actual == "" {
				actual = "--"
			}
			m.weekItems[i].SetTitle(fmt.Sprintf("%s %s  %s → %s", wdName, day.Format("01-02"), r.StartTime, actual))
			m.weekItems[i].Show()
		} else if day.After(now) {
			m.weekItems[i].Hide()
		} else {
			m.weekItems[i].SetTitle(fmt.Sprintf("%s %s  无记录", wdName, day.Format("01-02")))
			m.weekItems[i].Show()
		}
	}

	m.weekSumm.SetTitle(fmt.Sprintf("本周共 %d 天，延迟 %d 次", dayCount, lateCount))
}

func (m *MenuBar) showConfigDialog() {
	wh := config.Load()
	current := fmt.Sprintf("%02d:%02d-%02d:%02d", wh.StartHour, wh.StartMin, wh.EndHour, wh.EndMin)
	script := fmt.Sprintf(`display dialog "请输入上下班时间 (格式 HH:MM-HH:MM)" default answer "%s" with title "worktime 设置"`, current)
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return
	}
	text := strings.TrimSpace(string(out))
	idx := strings.Index(text, "text returned:")
	if idx < 0 {
		return
	}
	val := strings.TrimSpace(text[idx+len("text returned:"):])
	parts := strings.Split(val, "-")
	if len(parts) != 2 {
		return
	}
	start := strings.Split(strings.TrimSpace(parts[0]), ":")
	end := strings.Split(strings.TrimSpace(parts[1]), ":")
	if len(start) != 2 || len(end) != 2 {
		return
	}
	sh, _ := strconv.Atoi(start[0])
	sm, _ := strconv.Atoi(start[1])
	eh, _ := strconv.Atoi(end[0])
	em, _ := strconv.Atoi(end[1])
	wh.StartHour = sh
	wh.StartMin = sm
	wh.EndHour = eh
	wh.EndMin = em
	if err := config.Save(wh); err != nil {
		return
	}
	m.mConfig.SetTitle(fmt.Sprintf("设置 (上班 %02d:%02d / 下班 %02d:%02d)", sh, sm, eh, em))
}

func (m *MenuBar) showConfigRangeDialog() {
	wh := config.Load()
	rbH, rbM := wh.RangeBegin()/60, wh.RangeBegin()%60
	reH, reM := wh.RangeEnd()/60, wh.RangeEnd()%60
	current := fmt.Sprintf("%02d:%02d-%02d:%02d", rbH, rbM, reH, reM)
	script := fmt.Sprintf(`display dialog "请输入上班检测时间段 (格式 HH:MM-HH:MM)" default answer "%s" with title "worktime 设置"`, current)
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return
	}
	text := strings.TrimSpace(string(out))
	idx := strings.Index(text, "text returned:")
	if idx < 0 {
		return
	}
	val := strings.TrimSpace(text[idx+len("text returned:"):])
	parts := strings.Split(val, "-")
	if len(parts) != 2 {
		return
	}
	begin := strings.Split(strings.TrimSpace(parts[0]), ":")
	end := strings.Split(strings.TrimSpace(parts[1]), ":")
	if len(begin) != 2 || len(end) != 2 {
		return
	}
	bh, _ := strconv.Atoi(begin[0])
	bm, _ := strconv.Atoi(begin[1])
	eh, _ := strconv.Atoi(end[0])
	em, _ := strconv.Atoi(end[1])
	wh.RangeBeginHour = bh
	wh.RangeBeginMin = bm
	wh.RangeEndHour = eh
	wh.RangeEndMin = em
	if err := config.Save(wh); err != nil {
		return
	}
	m.mConfigRange.SetTitle(fmt.Sprintf("设置检测时间段 (%02d:%02d-%02d:%02d)", bh, bm, eh, em))
}

func (m *MenuBar) toggleAutoStart() {
	if brewservice.IsRunning() {
		brewservice.Stop()
		m.mAutoStart.SetTitle("开机启动: 已关闭")
	} else {
		brewservice.Start()
		m.mAutoStart.SetTitle("开机启动: 已开启")
	}
}

func (m *MenuBar) showModifyTodayDialog() {
	today := time.Now().Format("2006-01-02")

	current := ""
	if override := config.LoadOverride(today); override != nil {
		current = fmt.Sprintf("%02d:%02d", override.Hour, override.Min)
	} else if status, _ := attendance.GetToday(); status != nil {
		current = status.StartTime
	}

	wh := config.Load()
	rb := wh.RangeBegin()
	re := wh.RangeEnd()
	seen := make(map[string]bool)
	var opts []string
	if events, err := parser.GetParsedLog(); err == nil {
		for _, e := range events[today] {
			if e.Type != "start" {
				continue
			}
			mins := e.Time.Hour()*60 + e.Time.Minute()
			if mins >= rb && mins <= re {
				t := e.Time.Format("15:04")
				if !seen[t] {
					seen[t] = true
					opts = append(opts, t)
				}
			}
		}
	}

	if current != "" && !seen[current] {
		opts = append([]string{current}, opts...)
	}
	opts = append(opts, "手动输入...")

	listItems := strings.Join(opts, `", "`)
	script := fmt.Sprintf(
		`set chosen to choose from list {"%s"} with title "修改今日上班时间" with prompt "请选择今日的上班时间（根据亮屏记录）:" default items {"%s"}
if chosen is false then return "cancel"
return item 1 of chosen`, listItems, current)
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil || string(out) == "cancel" {
		return
	}
	val := strings.TrimSpace(string(out))

	if val == "手动输入..." {
		script := fmt.Sprintf(`display dialog "请输入今日上班时间 (格式 HH:MM)" default answer "%s" with title "修改今日上班时间"`, current)
		out, err := exec.Command("osascript", "-e", script).Output()
		if err != nil {
			return
		}
		text := strings.TrimSpace(string(out))
		idx := strings.Index(text, "text returned:")
		if idx < 0 {
			return
		}
		val = strings.TrimSpace(text[idx+len("text returned:"):])
	}

	parts := strings.Split(val, ":")
	if len(parts) != 2 {
		return
	}
	hh, _ := strconv.Atoi(parts[0])
	mm, _ := strconv.Atoi(parts[1])

	if err := config.SaveOverride(today, hh, mm); err != nil {
		return
	}

	attendance.ClearCache()
	if status, err := attendance.GetToday(); err == nil {
		m.Update(status)
	}
}

func loadChangelog() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	paths := []string{
		filepath.Join(dir, "..", "Resources", "CHANGELOG.md"),
		filepath.Join(dir, "CHANGELOG.md"),
	}
	wd, _ := os.Getwd()
	if wd != "" {
		paths = append(paths, filepath.Join(wd, "CHANGELOG.md"))
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data)
		}
	}
	return ""
}

func stripMarkdown(s string) string {
	s = strings.TrimPrefix(s, "- ")
	s = strings.TrimPrefix(s, "  - ")
	s = strings.TrimPrefix(s, "* ")
	s = strings.TrimLeft(s, "# ")
	return s
}

func extractVersionSection(content, version string) string {
	var buf strings.Builder
	marker := "## v" + version
	inSection := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## v") {
			if inSection {
				break
			}
			if trimmed == marker || strings.HasPrefix(trimmed, marker+" ") {
				inSection = true
				continue
			}
		}
		if inSection {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	return strings.TrimSpace(buf.String())
}

func (m *MenuBar) buildVersionSubmenu() {
	parent := systray.AddMenuItem(fmt.Sprintf("当前版本 v%s", m.version), "")
	content := loadChangelog()
	if content == "" {
		item := parent.AddSubMenuItem("未找到更新日志", "")
		item.Disable()
		return
	}
	section := extractVersionSection(content, m.version)
	if section == "" {
		item := parent.AddSubMenuItem("暂无更新记录", "")
		item.Disable()
		return
	}
	for _, line := range strings.Split(section, "\n") {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		text = stripMarkdown(text)
		if len(text) > 80 {
			text = text[:80] + "..."
		}
		item := parent.AddSubMenuItem(text, "")
		item.Disable()
	}
}
