# worktime

macOS 自动考勤菜单栏工具。通过解析 `pmset -g log` 系统日志，自动识别上下班时间，在菜单栏实时显示状态，并在下班前发送通知提醒。

## 功能

- 自动识别上班时间（9:00-11:00 窗口内首次唤醒/登录/开盖事件）
- 自动识别下班时间（18:00 后最后一次休眠/关盖/关机事件）
- 菜单栏实时显示：🟢 预计下班时间 / 🟡 剩余时间 / 🔴 已下班
- 下班前 10 分钟提醒 + 到点提醒（macOS 原生通知）
- CLI 查看今日/本周考勤统计
- 导出 CSV 考勤记录
- LaunchAgent 开机自启

## 安装

### Homebrew（推荐）

```bash
brew install Soarkey/tap/worktime
```

### 从源码编译

```bash
git clone https://github.com/Soarkey/worktime.git
cd worktime
make build
```

编译产物在 `build/worktime`。

## 使用

### 启动守护进程

```bash
worktime daemon
```

启动后菜单栏会显示考勤状态，每分钟自动刷新。

### 设置开机自启

```bash
worktime install
```

### CLI 命令

```bash
worktime status    # 查看当前状态
worktime today     # 今日考勤详情
worktime week      # 本周统计
worktime export    # 导出 CSV（默认 worktime.csv）
worktime export -o ~/Desktop/attendance.csv
```

### 卸载

```bash
worktime uninstall          # 仅卸载 LaunchAgent
worktime uninstall --purge  # 同时清理日志和数据库
```

## 配置

默认参数（`internal/config/config.go`）：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| 标准上班时间 | 10:00 | 用于计算迟到和预计下班 |
| 标准下班时间 | 19:00 | 标准工时 9 小时 |
| 上班检测窗口 | 9:00-11:00 | 此窗口内首个事件视为上班 |
| 下班检测起始 | 18:00 | 此时间后的事件视为下班 |
| 提前提醒 | 10 分钟 | 下班前通知 |
| 轮询间隔 | 1 分钟 | pmset 日志刷新频率 |

## 数据存储

- 数据库：`~/Library/Application Support/worktime/worktime.db`（SQLite）
- 日志：`~/Library/Logs/worktime/`

## 技术栈

- Go 1.22+
- [energye/systray](https://github.com/energye/systray) — 菜单栏
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — 纯 Go SQLite
- [spf13/cobra](https://github.com/spf13/cobra) — CLI 框架
- macOS `osascript` — 原生通知

## License

MIT
