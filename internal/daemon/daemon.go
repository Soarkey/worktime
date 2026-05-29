package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/menubar"
	"github.com/Soarkey/worktime/internal/notify"
)

// Start ensures the .app bundle exists, then launches the daemon inside it
// in a detached background process and returns immediately.
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

	if status == nil {
		return
	}

	if status.RemainingMinutes <= 0 {
		notifier.SendOnce("done", "worktime", "到点下班了！")
	}

	if status.State == "off" {
		return
	}
}
