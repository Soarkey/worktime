package brewservice

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const ServiceName = "worktime"

// brewLabel is the launchd label used by Homebrew services.
const brewLabel = "homebrew.mxcl.worktime"

func plistPath() string {
	return filepath.Join(os.Getenv("HOME"), "Library/LaunchAgents", brewLabel+".plist")
}

// Start starts the brew service via launchctl directly,
// avoiding PATH issues when running in a GUI context.
func Start() error {
	uid := os.Getuid()
	domainTarget := fmt.Sprintf("gui/%d", uid)
	target := fmt.Sprintf("gui/%d/%s", uid, brewLabel)
	plist := plistPath()

	exe, err := exePath()
	if err != nil {
		return err
	}

	// Always write a fresh plist so the binary path is never stale.
	if err := writePlist(plist, exe); err != nil {
		return fmt.Errorf("无法生成 plist: %w", err)
	}

	if err := exec.Command("launchctl", "bootstrap", domainTarget, plist).Run(); err != nil {
		// Bootstrap fails when the service is already registered.
		// Unregister first, then try again.
		exec.Command("launchctl", "bootout", target).Run()
		if err := exec.Command("launchctl", "bootstrap", domainTarget, plist).Run(); err != nil {
			return fmt.Errorf("启动失败: %w", err)
		}
	}
	return nil
}

// Stop tries to stop the brew service via launchctl, with brew CLI as fallback.
// Errors are silently ignored since the service may simply not be running.
func Stop() {
	uid := os.Getuid()
	target := fmt.Sprintf("gui/%d/%s", uid, brewLabel)

	exec.Command("launchctl", "bootout", target).Run()

	if _, err := exec.LookPath("brew"); err == nil {
		exec.Command("brew", "services", "stop", ServiceName).Run()
	}
}

func exePath() (string, error) {
	// Prefer brew prefix so the plist points to the installed location.
	if brew, err := exec.LookPath("brew"); err == nil {
		out, err := exec.Command(brew, "--prefix", ServiceName).Output()
		if err == nil {
			p := strings.TrimSpace(string(out))
			if p != "" {
				bundlePath := filepath.Join(p, "bin", "worktime.app", "Contents", "MacOS", "worktime")
				if _, err := os.Stat(bundlePath); err == nil {
					return bundlePath, nil
				}
				return filepath.Join(p, "bin", ServiceName), nil
			}
		}
	}

	// Check for .app bundle next to the binary.
	exe, err := os.Executable()
	if err == nil {
		bundlePath := filepath.Join(filepath.Dir(exe), "worktime.app", "Contents", "MacOS", filepath.Base(exe))
		if _, err := os.Stat(bundlePath); err == nil {
			return bundlePath, nil
		}
	}
	return exe, nil
}

func writePlist(path, exe string) error {
	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>daemon</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`, brewLabel, exe)
	return os.WriteFile(path, []byte(content), 0644)
}

func IsRunning() bool {
	uid := os.Getuid()
	target := fmt.Sprintf("gui/%d/%s", uid, brewLabel)
	out, err := exec.Command("launchctl", "print", target).Output()
	if err != nil {
		return false
	}
	return !strings.Contains(string(out), "Could not find domain")
}