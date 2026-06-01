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
	debug.SetGCPercent(5)
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

func scheduleUpdates(mb *menubar.MenuBar) {
	status, err := attendance.GetToday()
	if err != nil {
		return
	}
	mb.Update(status)

	if status == nil || status.State == "off" {
		return
	}

	time.AfterFunc(time.Duration(status.RemainingMinutes)*time.Minute, func() {
		mb.SetOffTitle()
	})
}
