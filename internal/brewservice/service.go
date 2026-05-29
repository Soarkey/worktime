package brewservice

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const ServiceName = "worktime"

// brewLabel is the launchd label used by Homebrew services.
const brewLabel = "homebrew.mxcl.worktime"

func Start() error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("未找到 Homebrew，请先安装: https://brew.sh")
	}
	return exec.Command("brew", "services", "start", ServiceName).Run()
}

// Stop stops the brew service via launchctl directly,
// avoiding PATH issues when running in a GUI context.
func Stop() error {
	uid := os.Getuid()
	target := fmt.Sprintf("gui/%d/%s", uid, brewLabel)

	if err := exec.Command("launchctl", "bootout", target).Run(); err == nil {
		return nil
	}

	// Fallback: try brew CLI if available.
	if _, err := exec.LookPath("brew"); err == nil {
		return exec.Command("brew", "services", "stop", ServiceName).Run()
	}

	return fmt.Errorf("无法停止后台服务，请手动执行: worktime stop")
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