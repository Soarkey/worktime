package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

const bundleID = "com.soarkey.worktime"

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

	return binPath, nil
}
