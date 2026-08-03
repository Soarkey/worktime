package parser

import (
	"os"
	"testing"
	"time"
)

func mkEvent(date, t, typ string) Event {
	ts, _ := time.ParseInLocation(storeTimeFmt, date+" "+t, time.Local)
	return Event{Time: ts, Type: typ}
}

func TestMergeEvents(t *testing.T) {
	a := map[string][]Event{
		"2026-08-01": {mkEvent("2026-08-01", "09:00:00", "start")},
	}
	b := map[string][]Event{
		"2026-08-01": {mkEvent("2026-08-01", "09:00:00", "start"), mkEvent("2026-08-01", "19:00:00", "leave")},
		"2026-08-02": {mkEvent("2026-08-02", "09:30:00", "start")},
	}
	merged := mergeEvents(a, b)
	if len(merged) != 2 {
		t.Fatalf("expected 2 dates, got %d", len(merged))
	}
	if got := len(merged["2026-08-01"]); got != 2 {
		t.Errorf("2026-08-01 dedupe failed: got %d events, want 2", got)
	}
}

func TestMergeEventsSortsByTime(t *testing.T) {
	// 系统日志补充了更早的 start,合并后应按时间排序,FindStartTime 才能取到最早值
	stored := map[string][]Event{
		"2026-08-01": {mkEvent("2026-08-01", "10:05:00", "start")},
	}
	live := map[string][]Event{
		"2026-08-01": {mkEvent("2026-08-01", "09:30:00", "start"), mkEvent("2026-08-01", "19:00:00", "leave")},
	}
	merged := mergeEvents(stored, live)
	got := FindStartTime(merged["2026-08-01"], 8*60, 12*60)
	if got == nil || got.Hour() != 9 || got.Minute() != 30 {
		t.Errorf("expected start 09:30, got %v", got)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	orig := storePathFn
	dir := t.TempDir()
	storePathFn = func() string { return dir + "/events.json" }
	defer func() { storePathFn = orig }()

	events := map[string][]Event{
		"2026-08-01": {mkEvent("2026-08-01", "09:00:00", "start"), mkEvent("2026-08-01", "19:00:00", "leave")},
	}
	if err := SaveStore(events); err != nil {
		t.Fatalf("SaveStore: %v", err)
	}
	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore: %v", err)
	}
	if len(loaded["2026-08-01"]) != 2 {
		t.Errorf("round trip: got %d events", len(loaded["2026-08-01"]))
	}
	loaded["2026-08-01"][0].Type = "xxx"
	if events["2026-08-01"][0].Type != "start" {
		t.Error("LoadStore 应返回独立副本")
	}
}

func TestMergeStoreRepairsCorrupt(t *testing.T) {
	orig := storePathFn
	dir := t.TempDir()
	storePathFn = func() string { return dir + "/events.json" }
	defer func() { storePathFn = orig }()

	os.WriteFile(storePath(), []byte("{corrupt"), 0644)
	events := map[string][]Event{
		"2026-08-01": {mkEvent("2026-08-01", "09:00:00", "start")},
	}
	if err := MergeStore(events); err != nil {
		t.Fatalf("MergeStore: %v", err)
	}
	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("损坏文件未修复: %v", err)
	}
	if len(loaded["2026-08-01"]) != 1 {
		t.Errorf("修复后数据缺失: %d 条", len(loaded["2026-08-01"]))
	}
}

func TestMergeStoreIdempotent(t *testing.T) {
	orig := storePathFn
	dir := t.TempDir()
	storePathFn = func() string { return dir + "/events.json" }
	defer func() { storePathFn = orig }()

	day1 := map[string][]Event{
		"2026-08-01": {mkEvent("2026-08-01", "09:00:00", "start")},
	}
	day2 := map[string][]Event{
		"2026-08-01": {mkEvent("2026-08-01", "09:00:00", "start"), mkEvent("2026-08-01", "19:30:00", "leave")},
	}
	if err := MergeStore(day1); err != nil {
		t.Fatalf("MergeStore day1: %v", err)
	}
	if err := MergeStore(day2); err != nil {
		t.Fatalf("MergeStore day2: %v", err)
	}
	// 第三次相同数据合并不应增加记录
	if err := MergeStore(day2); err != nil {
		t.Fatalf("MergeStore day2 again: %v", err)
	}
	loaded, _ := LoadStore()
	if got := len(loaded["2026-08-01"]); got != 2 {
		t.Errorf("idempotent merge failed: got %d events, want 2", got)
	}
}
