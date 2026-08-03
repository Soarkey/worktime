package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// 事件持久化存储:~/.worktime/events.json
// 格式:{"version":1,"events":{"2026-08-03":[{"time":"09:02:33","type":"start"},...]}}

const (
	storeName    = "events.json"
	storeVersion = 1
	eventTimeFmt = "15:04:05"
	storeTimeFmt = "2006-01-02 15:04:05"
)

type storedEvent struct {
	Time string `json:"time"`
	Type string `json:"type"`
}

type storeFile struct {
	Version int                      `json:"version"`
	Events  map[string][]storedEvent `json:"events"`
}

var storePathFn = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".worktime", storeName)
}

func storePath() string { return storePathFn() }

// LoadStore 读取本地持久化事件;文件不存在或损坏时返回空数据。
func LoadStore() (map[string][]Event, error) {
	data, err := os.ReadFile(storePath())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string][]Event), nil
		}
		return nil, err
	}
	var sf storeFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	events := make(map[string][]Event, len(sf.Events))
	for date, list := range sf.Events {
		for _, se := range list {
			t, err := time.ParseInLocation(storeTimeFmt, date+" "+se.Time, time.Local)
			if err != nil {
				continue
			}
			events[date] = append(events[date], Event{Time: t, Type: se.Type})
		}
	}
	return events, nil
}

// SaveStore 原子写入本地持久化文件。
func SaveStore(events map[string][]Event) error {
	sf := storeFile{Version: storeVersion, Events: make(map[string][]storedEvent, len(events))}
	for date, list := range events {
		sorted := append([]Event(nil), list...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Time.Before(sorted[j].Time) })
		for _, e := range sorted {
			sf.Events[date] = append(sf.Events[date], storedEvent{Time: e.Time.Format(eventTimeFmt), Type: e.Type})
		}
	}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(storePath()), 0755); err != nil {
		return err
	}
	tmp := storePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, storePath())
}

// MergeStore 将新解析的事件幂等合并到本地存储并写盘。
// 若无新事件且存储完好则跳过写盘,避免每分钟全量重写。
func MergeStore(events map[string][]Event) error {
	stored, err := LoadStore()
	corrupt := err != nil
	if err != nil {
		stored = make(map[string][]Event)
	}
	merged := mergeEvents(stored, events)
	if !corrupt && !storeChanged(stored, merged) {
		return nil
	}
	return SaveStore(merged)
}

// storeChanged 比较两个事件集合是否等价(忽略顺序差异)。
func storeChanged(a, b map[string][]Event) bool {
	if len(a) != len(b) {
		return true
	}
	for date, list := range a {
		if len(list) != len(b[date]) {
			return true
		}
	}
	for date, list := range b {
		if len(list) != len(a[date]) {
			return true
		}
	}
	// 数量一致时用键集合去重比对
	keySet := func(m map[string][]Event) map[eventKey]bool {
		ks := make(map[eventKey]bool)
		for date, list := range m {
			for _, e := range list {
				ks[eventKey{date, e.Time.Format(storeTimeFmt), e.Type}] = true
			}
		}
		return ks
	}
	ka, kb := keySet(a), keySet(b)
	if len(ka) != len(kb) {
		return true
	}
	for k := range ka {
		if !kb[k] {
			return true
		}
	}
	return false
}

// SyncStore 解析系统日志并与本地持久化合并写盘,保证历史全量留存。
func SyncStore() error {
	events, err := ParsePmsetLog()
	if err != nil {
		return err
	}
	return MergeStore(events)
}

type eventKey struct {
	date string
	time string
	typ  string
}

// mergeEvents 按 (日期, 时间, 类型) 去重合并,同一天事件按时间排序。
func mergeEvents(a, b map[string][]Event) map[string][]Event {
	merged := make(map[string][]Event, len(a)+len(b))
	seen := make(map[eventKey]bool)
	add := func(events map[string][]Event) {
		for date, list := range events {
			for _, e := range list {
				k := eventKey{date, e.Time.Format(storeTimeFmt), e.Type}
				if seen[k] {
					continue
				}
				seen[k] = true
				merged[date] = append(merged[date], e)
			}
		}
	}
	add(a)
	add(b)
	for date := range merged {
		sort.Slice(merged[date], func(i, j int) bool { return merged[date][i].Time.Before(merged[date][j].Time) })
	}
	return merged
}
