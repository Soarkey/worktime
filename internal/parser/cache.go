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

	events, err := ParsePmsetLog()
	if err != nil {
		return nil, err
	}

	cached = events
	cachedAt = time.Now()
	return cached, nil
}

func ClearCache() {
	mu.Lock()
	defer mu.Unlock()
	cached = nil
}
