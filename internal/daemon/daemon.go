package daemon

import (
	"os"
	"time"

	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/launchagent"
	"github.com/Soarkey/worktime/internal/menubar"
	"github.com/Soarkey/worktime/internal/notify"
)

func ensureLogDir() {
	os.MkdirAll(launchagent.LogDir, 0755)
}

func Run() error {
	ensureLogDir()
	notifier := notify.New()
	mb := menubar.New(nil)

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

	if status == nil || status.State == "off" {
		return
	}

	if status.RemainingMinutes <= 0 {
		notifier.SendOnce("done", "worktime", "到点下班了！")
	}
}