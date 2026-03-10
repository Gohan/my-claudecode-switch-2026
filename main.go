package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"claude-switch/internal/repository"
	"claude-switch/internal/service"
	"claude-switch/internal/tui"
)

func main() {
	// 支持命令行参数直接运行 profile
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "run":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: claude-switch run <profile-name>")
				os.Exit(1)
			}
			profileName := os.Args[2]
			if err := runProfile(profileName); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "help", "-h", "--help":
			printHelp()
			return
		}
	}

	// 默认启动 TUI
	// 初始化 Service 层
	home, _ := os.UserHomeDir()
	profilesDir := filepath.Join(home, ".claude-switch", "profiles")
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	repo := repository.NewProfileRepositoryFS(profilesDir, settingsPath)
	runnerExec := service.NewProfileRunnerExec("claude")
	svc := service.NewProfileService(repo, runnerExec)

	p := tea.NewProgram(tui.NewModel(svc), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runProfile(name string) error {
	home, _ := os.UserHomeDir()
	profilesDir := filepath.Join(home, ".claude-switch", "profiles")
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	repo := repository.NewProfileRepositoryFS(profilesDir, settingsPath)
	runnerExec := service.NewProfileRunnerExec("claude")
	svc := service.NewProfileService(repo, runnerExec)

	// 使用 service 运行 profile
	return svc.Run(name)
}

func printHelp() {
	home, _ := os.UserHomeDir()
	profilesDir := filepath.Join(home, ".claude-switch", "profiles")

	// 创建 runner 来获取 RunDir
	runnerExec := service.NewProfileRunnerExec("claude")

	fmt.Println("Claude Switch - Manage and switch between Claude Code profiles")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  claude-switch              Launch TUI to manage profiles")
	fmt.Println("  claude-switch run <name>   Run claude with specified profile")
	fmt.Println("  claude-switch help         Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  claude-switch run zai      Run claude with 'zai' profile")
	fmt.Println()
	fmt.Println("Profiles are stored in:")
	fmt.Printf("  %s\n", profilesDir)
	fmt.Println()
	fmt.Println("Run directories are created in:")
	fmt.Printf("  %s\n", runnerExec.RunDir())
}
