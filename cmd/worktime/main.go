package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/brewservice"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/daemon"
)

var version = "dev"

func init() {
	version = strings.TrimPrefix(version, "v")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "today":
		runToday()
	case "week":
		runWeek()
	case "export":
		runExport(args)
	case "start":
		runStart()
	case "stop":
		runStop()
	case "daemon":
		runDaemon()
	case "config":
		runConfig(args)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`用法: worktime <命令>

命令:
  today         查看今日考勤详情
  week          查看本周考勤统计
  export        导出考勤记录为 CSV
  start         启动菜单栏应用
  stop          停止菜单栏应用
  daemon        后台守护进程（由 start 或 launchd 调用）
  config        查看或设置上下班时间`)
}

func runToday() {
	status, err := attendance.GetToday()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	if status == nil {
		fmt.Println("今日无考勤记录")
		return
	}

	fmt.Printf("日期: %s\n", status.WorkDate)
	fmt.Printf("上班: %s\n", status.StartTime)
	fmt.Printf("预计下班: %s\n", status.ExpectedLeave)
	if status.LateMinutes > 0 {
		fmt.Printf("延迟: %d 分钟\n", status.LateMinutes)
	}
	if status.ActualLeave != "" {
		fmt.Printf("实际下班: %s\n", status.ActualLeave)
	}
}

func runWeek() {
	records, err := attendance.GetWeek()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	if len(records) == 0 {
		fmt.Println("本周无考勤记录")
		return
	}

	fmt.Printf("%-12s %-6s %-6s %-6s %s\n", "日期", "上班", "预计", "实际", "延迟")
	fmt.Println("-----------------------------------------------")
	for _, r := range records {
		late := ""
		if r.LateMinutes > 0 {
			late = fmt.Sprintf("%d分钟", r.LateMinutes)
		}
		fmt.Printf("%-12s %-6s %-6s %-6s %s\n",
			r.WorkDate, r.StartTime, r.ExpectedLeave, r.ActualLeave, late)
	}
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	output := fs.String("output", "worktime.csv", "输出文件路径")
	fs.StringVar(output, "o", "worktime.csv", "输出文件路径（简写）")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	records, err := attendance.GetAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	if len(records) == 0 {
		fmt.Println("无考勤记录")
		return
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].WorkDate < records[j].WorkDate
	})

	f, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"日期", "上班时间", "预计下班", "实际下班", "延迟(分钟)"})
	for _, r := range records {
		w.Write([]string{r.WorkDate, r.StartTime, r.ExpectedLeave, r.ActualLeave, fmt.Sprintf("%d", r.LateMinutes)})
	}
	w.Flush()

	fmt.Printf("已导出 %d 条记录到 %s\n", len(records), *output)
}

func runStart() {
	if _, err := daemon.EnsureBundle(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	if err := brewservice.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	if err := daemon.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("worktime 已在后台启动")
}

func runStop() {
	brewservice.Stop()
	killDaemon()
}

func runDaemon() {
	if err := daemon.Run(version); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func runConfig(args []string) {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	startHour := fs.Int("start-hour", 0, "上班小时 (0-23)")
	startMin := fs.Int("start-min", 0, "上班分钟 (0-59)")
	endHour := fs.Int("end-hour", 0, "下班小时 (0-23)")
	endMin := fs.Int("end-min", 0, "下班分钟 (0-59)")
	rangeBegin := fs.String("range-begin", "", "上班统计开始时间 (HH:MM)")
	rangeEnd := fs.String("range-end", "", "上班统计结束时间 (HH:MM)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	wh := config.Load()
	changed := false

	if v := *startHour; v != 0 || hasFlag(args, "start-hour") {
		wh.StartHour = v
		changed = true
	}
	if v := *startMin; v != 0 || hasFlag(args, "start-min") {
		wh.StartMin = v
		changed = true
	}
	if v := *endHour; v != 0 || hasFlag(args, "end-hour") {
		wh.EndHour = v
		changed = true
	}
	if v := *endMin; v != 0 || hasFlag(args, "end-min") {
		wh.EndMin = v
		changed = true
	}
	if v := *rangeBegin; v != "" || hasFlag(args, "range-begin") {
		if h, m, err := config.ParseHHMM(v); err == nil {
			wh.RangeBeginHour = h
			wh.RangeBeginMin = m
			changed = true
		}
	}
	if v := *rangeEnd; v != "" || hasFlag(args, "range-end") {
		if h, m, err := config.ParseHHMM(v); err == nil {
			wh.RangeEndHour = h
			wh.RangeEndMin = m
			changed = true
		}
	}

	if changed {
		if err := config.Save(wh); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("已保存")
	}

	fmt.Printf("上班时间:   %02d:%02d\n", wh.StartHour, wh.StartMin)
	fmt.Printf("下班时间:   %02d:%02d\n", wh.EndHour, wh.EndMin)
	rbH, rbM := wh.RangeBegin()/60, wh.RangeBegin()%60
	reH, reM := wh.RangeEnd()/60, wh.RangeEnd()%60
	fmt.Printf("上班统计时间段: %02d:%02d - %02d:%02d\n", rbH, rbM, reH, reM)
}

func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == "--"+name || a == "-"+name {
			return true
		}
	}
	return false
}

func killDaemon() {
	data, err := os.ReadFile(filepath.Join(os.TempDir(), "worktime.pid"))
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	syscall.Kill(pid, syscall.SIGTERM)
}
