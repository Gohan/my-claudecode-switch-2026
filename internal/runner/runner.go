package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// RunDir 返回运行目录路径
func RunDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude-switch", "runs")
}

// PrepareRunDir 为指定 profile 准备运行目录
// 创建目录并将 profile 的 settings 复制到该目录
func PrepareRunDir(profileName string, settings map[string]interface{}) (string, error) {
	runDir := filepath.Join(RunDir(), profileName)

	// 创建目录
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create run directory: %w", err)
	}

	// 写入 settings.json
	settingsPath := filepath.Join(runDir, "settings.json")
	if err := writeSettings(settingsPath, settings); err != nil {
		return "", fmt.Errorf("failed to write settings: %w", err)
	}

	return runDir, nil
}

// Run 使用指定的 CLAUDE_CONFIG_DIR 启动 claude
func Run(configDir string) error {
	cmd := BuildCommand(configDir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// BuildCommand 返回用于启动 claude 的 exec.Cmd
// 适用于需要与 tea.ExecProcess 一起使用的场景
func BuildCommand(configDir string) *exec.Cmd {
	return buildCommand(configDir)
}

// buildCommand 根据操作系统构建启动命令
func buildCommand(configDir string) *exec.Cmd {
	env := append(os.Environ(), "CLAUDE_CONFIG_DIR="+configDir)

	switch runtime.GOOS {
	case "windows":
		return buildWindowsCommand(configDir, env)
	default:
		return buildUnixCommand(configDir, env)
	}
}

// buildUnixCommand 构建 Unix/Linux/macOS 命令
func buildUnixCommand(configDir string, env []string) *exec.Cmd {
	cmd := exec.Command("claude")
	cmd.Env = env
	return cmd
}

// buildWindowsCommand 构建 Windows 命令
func buildWindowsCommand(configDir string, env []string) *exec.Cmd {
	// 直接启动 claude，通过 cmd.Env 设置环境变量
	// 这是跨 Windows 版本和终端类型最可靠的方式
	cmd := exec.Command("claude")
	cmd.Env = env
	return cmd
}

// writeSettings 将 settings 写入文件
func writeSettings(path string, settings map[string]interface{}) error {
	data, err := marshalSettings(settings)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// marshalSettings 将 settings map 序列化为 JSON
func marshalSettings(settings map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(settings, "", "  ")
}
