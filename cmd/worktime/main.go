package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"

	"github.com/Soarkey/worktime/internal/attendance"
	"github.com/Soarkey/worktime/internal/config"
	"github.com/Soarkey/worktime/internal/daemon"
	"github.com/Soarkey/worktime/internal/launchagent"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "worktime",
		Short: "macOS 上下班时间监测菜单栏工具",
	}

	root.AddCommand(daemonCmd(), statusCmd(), todayCmd(), weekCmd(), exportCmd(), installCmd(), uninstallCmd(), configCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func daemonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "daemon",
		Short: "启动后台守护进程（菜单栏 + 通知）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return daemon.Run()
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看当前考勤状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := attendance.GetToday()
			if err != nil {
				return err
			}
			if status == nil {
				fmt.Println("未检测到今日上班时间")
				return nil
			}

			fmt.Printf("日期: %s\n", status.WorkDate)
			fmt.Printf("上班: %s\n", status.StartTime)
			fmt.Printf("预计下班: %s\n", status.ExpectedLeave)
			if status.LateMinutes > 0 {
				fmt.Printf("迟到: %d 分钟\n", status.LateMinutes)
			}
			if status.ActualLeave != "" {
				fmt.Printf("实际下班: %s\n", status.ActualLeave)
			}
			fmt.Printf("状态: %s\n", attendance.MenuBarTitle(*status))
			return nil
		},
	}
}

func todayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "today",
		Short: "查看今日考勤详情",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := attendance.GetToday()
			if err != nil {
				return err
			}
			if status == nil {
				fmt.Println("今日无考勤记录")
				return nil
			}

			fmt.Printf("日期: %s\n", status.WorkDate)
			fmt.Printf("上班: %s\n", status.StartTime)
			fmt.Printf("预计下班: %s\n", status.ExpectedLeave)
			if status.LateMinutes > 0 {
				fmt.Printf("迟到: %d 分钟\n", status.LateMinutes)
			}
			if status.ActualLeave != "" {
				fmt.Printf("实际下班: %s\n", status.ActualLeave)
			}
			return nil
		},
	}
}

func weekCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "week",
		Short: "查看本周考勤统计",
		RunE: func(cmd *cobra.Command, args []string) error {
			records, err := attendance.GetWeek()
			if err != nil {
				return err
			}
			if len(records) == 0 {
				fmt.Println("本周无考勤记录")
				return nil
			}

			fmt.Printf("%-12s %-6s %-6s %-6s %s\n", "日期", "上班", "预计", "实际", "迟到")
			fmt.Println("-----------------------------------------------")
			for _, r := range records {
				late := ""
				if r.LateMinutes > 0 {
					late = fmt.Sprintf("%d分钟", r.LateMinutes)
				}
				fmt.Printf("%-12s %-6s %-6s %-6s %s\n",
					r.WorkDate, r.StartTime, r.ExpectedLeave, r.ActualLeave, late)
			}
			return nil
		},
	}
}

func exportCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "导出考勤记录为 CSV",
		RunE: func(cmd *cobra.Command, args []string) error {
			records, err := attendance.GetAll()
			if err != nil {
				return err
			}
			if len(records) == 0 {
				fmt.Println("无考勤记录")
				return nil
			}

			sort.Slice(records, func(i, j int) bool {
				return records[i].WorkDate < records[j].WorkDate
			})

			f, err := os.Create(output)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			defer f.Close()

			w := csv.NewWriter(f)
			w.Write([]string{"日期", "上班时间", "预计下班", "实际下班", "迟到(分钟)"})
			for _, r := range records {
				w.Write([]string{r.WorkDate, r.StartTime, r.ExpectedLeave, r.ActualLeave, fmt.Sprintf("%d", r.LateMinutes)})
			}
			w.Flush()

			fmt.Printf("已导出 %d 条记录到 %s\n", len(records), output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "worktime.csv", "输出文件路径")
	return cmd
}

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "安装 LaunchAgent（开机自启）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchagent.Install()
		},
	}
}

func uninstallCmd() *cobra.Command {
	var purge bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "卸载 LaunchAgent",
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchagent.Uninstall(purge)
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "同时清理日志")
	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "查看或设置上下班时间",
		RunE: func(cmd *cobra.Command, args []string) error {
			wh := config.Load()
			startH, _ := cmd.Flags().GetInt("start-hour")
			startM, _ := cmd.Flags().GetInt("start-min")
			endH, _ := cmd.Flags().GetInt("end-hour")
			endM, _ := cmd.Flags().GetInt("end-min")

			changed := false
			if cmd.Flags().Changed("start-hour") {
				wh.StartHour = startH
				changed = true
			}
			if cmd.Flags().Changed("start-min") {
				wh.StartMin = startM
				changed = true
			}
			if cmd.Flags().Changed("end-hour") {
				wh.EndHour = endH
				changed = true
			}
			if cmd.Flags().Changed("end-min") {
				wh.EndMin = endM
				changed = true
			}

			if changed {
				if err := config.Save(wh); err != nil {
					return err
				}
				fmt.Println("已保存")
			}

			fmt.Printf("上班时间: %02d:%02d\n", wh.StartHour, wh.StartMin)
			fmt.Printf("下班时间: %02d:%02d\n", wh.EndHour, wh.EndMin)
			return nil
		},
	}
	cmd.Flags().Int("start-hour", 0, "上班小时 (0-23)")
	cmd.Flags().Int("start-min", 0, "上班分钟 (0-59)")
	cmd.Flags().Int("end-hour", 0, "下班小时 (0-23)")
	cmd.Flags().Int("end-min", 0, "下班分钟 (0-59)")
	return cmd
}
