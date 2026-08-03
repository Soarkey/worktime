package parser

import (
	"sync"
	"time"
)

var (
	mu       sync.Mutex
	cached   map[string][]Event
	cachedAt time.Time
	ttl      = 5 * time.Second
)

func GetParsedLog() (map[string][]Event, error) {
	mu.Lock()
	defer mu.Unlock()

	if cached != nil && time.Since(cachedAt) < ttl {
		return cached, nil
	}

	events, err := mergedEvents()
	if err != nil {
		return nil, err
	}

	cached = events
	cachedAt = time.Now()
	return cached, nil
}

// mergedEvents 以本地持久化(全量历史)为基础,合并系统日志中的最新事件。
func mergedEvents() (map[string][]Event, error) {
	stored, err := LoadStore()
	if err != nil {
		stored = make(map[string][]Event)
	}
	live, err := ParsePmsetLog()
	if err != nil {
		return stored, nil
	}
	return mergeEvents(stored, live), nil
}

func ClearCache() {
	mu.Lock()
	defer mu.Unlock()
	cached = nil
}
