package menubar

import (
	"fmt"
	"os/exec"
	"strings"
)

// osaEscape 转义 AppleScript 字符串中的引号与换行。
func osaEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s)
}

// dialogInput 显示带默认值的输入对话框,返回输入文本;取消时 ok=false。
func dialogInput(title, prompt, def string) (string, bool) {
	script := fmt.Sprintf(`display dialog "%s" default answer "%s" with title "%s"`,
		osaEscape(prompt), osaEscape(def), osaEscape(title))
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", false
	}
	text := strings.TrimSpace(string(out))
	idx := strings.Index(text, "text returned:")
	if idx < 0 {
		return "", false
	}
	return strings.TrimSpace(text[idx+len("text returned:"):]), true
}

// dialogList 显示可选项列表,返回选中项;取消时 ok=false。
func dialogList(items []string, title, prompt, def string) (string, bool) {
	listItems := strings.Join(items, `", "`)
	script := fmt.Sprintf(`set chosen to choose from list {"%s"} with title "%s" with prompt "%s" default items {"%s"}
if chosen is false then return "cancel"
return item 1 of chosen`,
		listItems, osaEscape(title), osaEscape(prompt), osaEscape(def))
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil || strings.TrimSpace(string(out)) == "cancel" {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// dialogConfirm 显示确认对话框,返回是否点击了确认按钮。
func dialogConfirm(title, msg, okLabel string) bool {
	script := fmt.Sprintf(`display dialog "%s" with title "%s" buttons {"取消", "%s"} default button "%s"`,
		osaEscape(msg), osaEscape(title), okLabel, okLabel)
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "button returned:"+okLabel)
}

// dialogAlert 显示错误提示对话框。
func dialogAlert(title, msg string) {
	script := fmt.Sprintf(`display dialog "%s" with title "%s" buttons {"确定"} default button "确定" with icon stop`,
		osaEscape(msg), osaEscape(title))
	exec.Command("osascript", "-e", script).Run()
}
