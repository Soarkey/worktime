package daemon

import (
	"fmt"
	"time"

	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/menubar"
	"github.com/Soarkey/worktime/internal/notify"
	"github.com/Soarkey/worktime/internal/storage"
)

func Run() error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	notifier := notify.New()
	mb := menubar.New(store, func() { store.Close() })

	go pollLoop(store, notifier, mb)

	mb.Run()
	return nil
}

func pollLoop(store *storage.Store, notifier *notify.Notifier, mb *menubar.MenuBar) {
	lastDate := ""
	tick := time.NewTicker(config.PollInterval)
	defer tick.Stop()

	poll(store, notifier, mb, &lastDate)
	for range tick.C {
		poll(store, notifier, mb, &lastDate)
	}
}

func poll(store *storage.Store, notifier *notify.Notifier, mb *menubar.MenuBar, lastDate *string) {
	today := time.Now().Format("2006-01-02")
	if *lastDate != today {
		notifier.ResetDaily()
		*lastDate = today
	}

	status, err := attendance.Sync(store)
	if err != nil {
		return
	}

	mb.Update(status)

	if status == nil || status.State == "off" {
		return
	}

	if status.RemainingMinutes <= config.NotifyBeforeMinutes && status.RemainingMinutes > 0 {
		notifier.SendOnce("before", "worktime", fmt.Sprintf("还有 %d 分钟下班", status.RemainingMinutes))
	}
	if status.RemainingMinutes <= 0 {
		notifier.SendOnce("done", "worktime", "到点下班了！")
	}
}
