package notify

import "sync"

type Notifier struct {
	mu   sync.Mutex
	sent map[string]bool
}

func New() *Notifier {
	return &Notifier{sent: make(map[string]bool)}
}

func (n *Notifier) ResetDaily() {
	n.mu.Lock()
	n.sent = make(map[string]bool)
	n.mu.Unlock()
}

func (n *Notifier) SendOnce(key, title, message string) error {
	n.mu.Lock()
	if n.sent[key] {
		n.mu.Unlock()
		return nil
	}
	n.sent[key] = true
	n.mu.Unlock()

	return send(title, message)
}

func Test() error {
	return send("worktime", "这是一条测试通知，提醒功能正常工作！")
}
