package menubar

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/energye/systray"
	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/brewservice"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/notify"
)

type MenuBar struct {
	mStatus *systray.MenuItem
	// today submenu
	mToday     *systray.MenuItem
	todayDate  *systray.MenuItem
	todayStart *systray.MenuItem
	todayEnd   *systray.MenuItem
	todayLate  *systray.MenuItem
	todayLeave *systray.MenuItem
	// week submenu
	mWeek     *systray.MenuItem
	weekItems []*systray.MenuItem
	weekSumm  *systray.MenuItem
	// export & quit
	mExport     *systray.MenuItem
	mConfig     *systray.MenuItem
	mTestNotify *systray.MenuItem
	mAutoStart  *systray.MenuItem
	mQuit       *systray.MenuItem
}

func New() *MenuBar {
	return &MenuBar{}
}

func (m *MenuBar) Run() {
	systray.Run(m.onReady, nil)
}

func (m *MenuBar) onReady() {
	systray.SetTitle("⏳")
	systray.SetTooltip("worktime")

	systray.SetOnClick(func(menu systray.IMenu) { menu.ShowMenu() })
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

	systray.AddSeparator()

	m.mExport = systray.AddMenuItem("导出 CSV...", "导出考勤记录")
	m.mExport.Click(func() { go exportCSV() })

	wh := config.Load()
	m.mConfig = systray.AddMenuItem(fmt.Sprintf("设置 (上班 %02d:%02d / 下班 %02d:%02d)", wh.StartHour, wh.StartMin, wh.EndHour, wh.EndMin), "设置上下班时间")
	m.mConfig.Click(func() { go m.showConfigDialog() })

	m.mTestNotify = systray.AddMenuItem("提醒测试", "发送测试通知")
	m.mTestNotify.Click(func() {
		go func() {
			if err := notify.Test(); err != nil {
				exec.Command("/usr/bin/osascript", "-e",
					fmt.Sprintf(`display dialog %q with title "通知失败" buttons {"确定"} default button "确定"`, err.Error())).Run()
			}
		}()
	})

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
	// output: "button returned:OK, text returned:10:00-19:00"
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
	wh = config.WorkHours{StartHour: sh, StartMin: sm, EndHour: eh, EndMin: em}
	if err := config.Save(wh); err != nil {
		return
	}
	m.mConfig.SetTitle(fmt.Sprintf("设置 (上班 %02d:%02d / 下班 %02d:%02d)", sh, sm, eh, em))
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
