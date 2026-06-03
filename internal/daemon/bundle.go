package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

const bundleID = "com.soarkey.worktime"

func copyChangelog(appDir string) {
	src := filepath.Join(filepath.Dir(appDir), "CHANGELOG.md")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		exe, _ := os.Executable()
		src = filepath.Join(filepath.Dir(exe), "CHANGELOG.md")
		if _, err := os.Stat(src); os.IsNotExist(err) {
			return
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
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

	if err := os.MkdirAll(macosDir, 0755); err != nil {
		return "", fmt.Errorf("创建 bundle 目录失败: %w", err)
	}

	if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("清理旧 symlink 失败: %w", err)
	}
	if err := os.Symlink(exe, binPath); err != nil {
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
