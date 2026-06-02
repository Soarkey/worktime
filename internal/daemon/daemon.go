package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/menubar"
)

func Start() error {
	bundlePath, err := EnsureBundle()
	if err != nil {
		return err
	}
	cmd := exec.Command(bundlePath, "daemon")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}

var pidFile string

func setPidFile() {
	if pidFile == "" {
		pidFile = filepath.Join(os.TempDir(), "worktime.pid")
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

func tuneMemory() {
	runtime.GOMAXPROCS(1)
	debug.SetGCPercent(50)
	debug.SetMemoryLimit(6 * 1024 * 1024)
}

func releaseMemory() {
	runtime.GC()
	debug.FreeOSMemory()
}

func Run(version string) error {
	tuneMemory()

	if err := acquireSingleton(); err != nil {
		return err
	}
	defer removePid()

	mb := menubar.New(version)

	go scheduleUpdates(mb)

	releaseMemory()

	mb.Run()
	return nil
}

func todayAt(hour, min int) time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, time.Local)
}

func scheduleUpdates(mb *menubar.MenuBar) {
	var tick func()
	tick = func() {
		status, err := attendance.GetToday()
		if err == nil {
			mb.Update(status)
		}

		now := time.Now()
		wh := config.Load()
		var next time.Duration

		if status == nil {
			rb := todayAt(wh.RangeBegin()/60, wh.RangeBegin()%60)
			re := todayAt(wh.RangeEnd()/60, wh.RangeEnd()%60)
			switch {
			case now.Before(rb):
				next = rb.Sub(now)
			case now.After(re):
				next = rb.AddDate(0, 0, 1).Sub(now)
			default:
				next = 5 * time.Minute
			}
		} else if status.State == "off" {
			rb := todayAt(wh.RangeBegin()/60, wh.RangeBegin()%60)
			if now.After(rb) {
				rb = rb.AddDate(0, 0, 1)
			}
			next = rb.Sub(now)
		} else {
			if status.RemainingMinutes > 0 {
				next = time.Duration(status.RemainingMinutes+1) * time.Minute
			} else {
				next = 5 * time.Minute
			}
		}

		time.AfterFunc(next, tick)
	}

	tick()
}
