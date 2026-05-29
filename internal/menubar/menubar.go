package menubar

import (
	"fmt"

	"github.com/energye/systray"
	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/storage"
)

type MenuBar struct {
	store       *storage.Store
	mStatus     *systray.MenuItem
	mToday      *systray.MenuItem
	mWeek       *systray.MenuItem
	mQuit       *systray.MenuItem
	onQuit      func()
}

func New(store *storage.Store, onQuit func()) *MenuBar {
	return &MenuBar{store: store, onQuit: onQuit}
}

func (m *MenuBar) Run() {
	systray.Run(m.onReady, m.onExit)
}

func (m *MenuBar) onReady() {
	systray.SetTitle("⏳")
	systray.SetTooltip("worktime")

	m.mStatus = systray.AddMenuItem("加载中...", "当前状态")
	m.mStatus.Disable()
	systray.AddSeparator()
	m.mToday = systray.AddMenuItem("今日详情", "查看今日考勤")
	m.mToday.Disable()
	m.mWeek = systray.AddMenuItem("本周统计", "查看本周考勤")
	m.mWeek.Disable()
	systray.AddSeparator()
	m.mQuit = systray.AddMenuItem("退出", "退出 worktime")

	m.mQuit.Click(func() {
		systray.Quit()
	})
}

func (m *MenuBar) onExit() {
	if m.onQuit != nil {
		m.onQuit()
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
		detail += fmt.Sprintf(" | 迟到 %d 分钟", status.LateMinutes)
	}
	m.mStatus.SetTitle(detail)
}
