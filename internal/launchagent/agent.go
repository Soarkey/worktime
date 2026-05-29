package launchagent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const plistLabel = "com.soarkey.worktime"

// LogDir is the log directory for stdout/stderr when running as daemon.
// Set via ldflags: -X main.logDir=/opt/homebrew/var/log/worktime
var LogDir string

var plistTmpl = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.Binary}}</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogDir}}/worktime.log</string>
    <key>StandardErrorPath</key>
    <string>{{.LogDir}}/worktime.err</string>
</dict>
</plist>
`))

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", plistLabel+".plist")
}

func Install() error {
	binary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	path := plistPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create plist: %w", err)
	}
	defer f.Close()

	data := struct {
		Label  string
		Binary string
		LogDir string
	}{plistLabel, binary, LogDir}

	if err := plistTmpl.Execute(f, data); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	if err := exec.Command("launchctl", "load", path).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}

	fmt.Printf("已安装 LaunchAgent: %s\n", path)
	return nil
}

func IsInstalled() bool {
	_, err := os.Stat(plistPath())
	return err == nil
}

func Uninstall(purge bool) error {
	path := plistPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("LaunchAgent 未安装")
	} else {
		exec.Command("launchctl", "unload", path).Run()

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove plist: %w", err)
		}
		fmt.Printf("已卸载 LaunchAgent: %s\n", path)
	}

	if purge {
		if err := os.RemoveAll(LogDir); err == nil {
			fmt.Printf("已清理日志: %s\n", LogDir)
		}
	}

	return nil
}