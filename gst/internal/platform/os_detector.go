package platform

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// DetectOS 检测操作系统类型
// 返回: "ubuntu", "kylin", "uos", "other"
func DetectOS() string {
	// Read /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "other"
	}

	content := string(data)
	content = strings.ToLower(content)

	if strings.Contains(content, "ubuntu") {
		return "ubuntu"
	}
	if strings.Contains(content, "kylin") {
		return "kylin"
	}
	if strings.Contains(content, "uos") || strings.Contains(content, "uniontech") {
		return "uos"
	}
	if strings.Contains(content, "debian") {
		return "debian"
	}
	if strings.Contains(content, "fedora") || strings.Contains(content, "rhel") {
		return "rhel"
	}

	return "other"
}

// EnvInfo 环境变量信息
type EnvInfo struct {
	Display   string
	XDGConfig string
	Home      string
}

// GetEnvInfo 获取环境变量信息
func GetEnvInfo() *EnvInfo {
	return &EnvInfo{
		Display:   os.Getenv("DISPLAY"),
		XDGConfig: os.Getenv("XDG_CONFIG_HOME"),
		Home:      os.Getenv("HOME"),
	}
}

// CheckDesktopEnvironment 检查桌面环境
func CheckDesktopEnvironment() (bool, error) {
	// Check DISPLAY variable
	display := os.Getenv("DISPLAY")
	if display == "" {
		return false, nil
	}

	// Try to run xprop which is a common X11 utility
	cmd := exec.Command("xprop", "-root", "WM_NAME")
	cmd.Run()

	// Check if X server is accessible
	if runtime.GOOS == "linux" {
		// Try to execute xdpyinfo as a more reliable check
		cmd = exec.Command("xdpyinfo")
		err := cmd.Run()
		if err == nil {
			return true, nil
		}
	}

	// DISPLAY is set, assume desktop environment exists
	return display != "", nil
}

// IsSupportedOS 检查是否支持
func IsSupportedOS() bool {
	osType := DetectOS()
	return osType == "ubuntu" || osType == "kylin" || osType == "uos" || osType == "debian"
}

// GetOSVersion 获取操作系统版本详细信息
func GetOSVersion() (name string, version string) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown", "unknown"
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	name = "unknown"
	version = "unknown"

	for _, line := range lines {
		if strings.HasPrefix(line, "NAME=") {
			name = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		}
		if strings.HasPrefix(line, "VERSION=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION="), "\"")
		}
	}

	return
}

// IsKylinV10 检查是否为麒麟V10系统
func IsKylinV10() bool {
	data, err := os.ReadFile("/etc/os-version")
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "v10") || strings.Contains(content, "kylin")
}
