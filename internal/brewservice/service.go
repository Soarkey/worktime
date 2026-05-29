package brewservice

import (
	"fmt"
	"os/exec"
	"strings"
)

const ServiceName = "worktime"

func Start() error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("未找到 Homebrew，请先安装: https://brew.sh")
	}
	return exec.Command("brew", "services", "start", ServiceName).Run()
}

func Stop() error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("未找到 Homebrew")
	}
	return exec.Command("brew", "services", "stop", ServiceName).Run()
}

func IsRunning() bool {
	out, err := exec.Command("brew", "services", "list").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, ServiceName) && strings.Contains(line, "started") {
			return true
		}
	}
	return false
}