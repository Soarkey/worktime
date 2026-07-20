package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func brewBinPath() string {
	brew, err := exec.LookPath("brew")
	if err != nil {
		return ""
	}
	out, err := exec.Command(brew, "--prefix", "worktime").Output()
	if err != nil {
		return ""
	}
	p := strings.TrimSpace(string(out))
	if p == "" {
		return ""
	}
	return filepath.Join(p, "bin", "worktime")
}

const bundleID = "com.soarkey.worktime"

func copyChangelog(appDir string) {
	exe, _ := os.Executable()
	candidates := []string{
		filepath.Join(filepath.Dir(appDir), "CHANGELOG.md"),
		filepath.Join(filepath.Dir(exe), "CHANGELOG.md"),
	}
	if bp := brewBinPath(); bp != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(bp), "..", "CHANGELOG.md"))
	}
	var data []byte
	for _, src := range candidates {
		if d, err := os.ReadFile(src); err == nil {
			data = d
			break
		}
	}
	if data == nil {
		return
	}
	resDir := filepath.Join(appDir, "Contents", "Resources")
	os.MkdirAll(resDir, 0755)
	os.WriteFile(filepath.Join(resDir, "CHANGELOG.md"), data, 0644)
}

// EnsureBundle creates a minimal .app bundle at <executable-dir>/worktime.app
// so that NSApplication has a proper CFBundleIdentifier for macOS notifications.
// Returns the path to the binary inside the bundle.
func EnsureBundle() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取可执行路径失败: %w", err)
	}

	dir := filepath.Dir(exe)
	appDir := filepath.Join(dir, "worktime.app")
	macosDir := filepath.Join(appDir, "Contents", "MacOS")
	binPath := filepath.Join(macosDir, filepath.Base(exe))
	plistPath := filepath.Join(appDir, "Contents", "Info.plist")

	target := exe
	if bp := brewBinPath(); bp != "" {
		target = bp
	}

	if err := os.MkdirAll(macosDir, 0755); err != nil {
		return "", fmt.Errorf("创建 bundle 目录失败: %w", err)
	}

	if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("清理旧 symlink 失败: %w", err)
	}
	if err := os.Symlink(target, binPath); err != nil {
		return "", fmt.Errorf("创建 symlink 失败: %w", err)
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>%s</string>
	<key>CFBundleIdentifier</key>
	<string>%s</string>
	<key>CFBundleName</key>
	<string>worktime</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>LSUIElement</key>
	<true/>
</dict>
</plist>
`, filepath.Base(exe), bundleID)

	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return "", fmt.Errorf("写入 Info.plist 失败: %w", err)
	}

	copyChangelog(appDir)

	return binPath, nil
}
