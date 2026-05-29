package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/menubar"
	"github.com/Soarkey/worktime/internal/notify"
	"github.com/energye/systray"
)

var pidFile string

func setPidFile() {
	if pidFile == "" {
		dir := os.TempDir()
		pidFile = filepath.Join(dir, "worktime.pid")
	}
}

func readPid() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func writePid() error {
	return os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func removePid() {
	os.Remove(pidFile)
}

func isProcessRunning(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func acquireSingleton() error {
	setPidFile()

	if pid, err := readPid(); err == nil && isProcessRunning(pid) {
		return fmt.Errorf("worktime 已在运行中 (pid %d)", pid)
	}

	return writePid()
}

// Run starts the daemon with menu bar and notification polling.
func Run() error {
	if err := acquireSingleton(); err != nil {
		return err
	}
	defer removePid()

	notifier := notify.New()
	mb := menubar.New()

	go pollLoop(notifier, mb)

	mb.Run()
	return nil
}

// Stop quits the systray, removes the pid file, and stops brew service.
func Stop() {
	systray.Quit()
}

func pollLoop(notifier *notify.Notifier, mb *menubar.MenuBar) {
	lastDate := ""
	tick := time.NewTicker(config.PollInterval)
	defer tick.Stop()

	poll(notifier, mb, &lastDate)
	for range tick.C {
		poll(notifier, mb, &lastDate)
	}
}

func poll(notifier *notify.Notifier, mb *menubar.MenuBar, lastDate *string) {
	today := time.Now().Format("2006-01-02")
	if *lastDate != today {
		notifier.ResetDaily()
		*lastDate = today
	}

	status, err := attendance.GetToday()
	if err != nil {
		return
	}

	mb.Update(status)

	if status == nil || status.State == "off" {
		return
	}

	if status.RemainingMinutes <= 0 {
		notifier.SendOnce("done", "worktime", "到点下班了！")
	}
}