package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"claude-switch/internal/profile"
	"claude-switch/internal/repository"
	"claude-switch/internal/runner"
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
	p, err := profile.GetByName(name)
	if err != nil {
		return fmt.Errorf("profile '%s' not found", name)
	}

	// 准备运行目录
	runDir, err := runner.PrepareRunDir(p.Name, p.Settings)
	if err != nil {
		return fmt.Errorf("failed to prepare run directory: %w", err)
	}

	// 启动 claude
	return runner.Run(runDir)
}

func printHelp() {
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
	fmt.Printf("  %s\n", profile.ProfilesDir())
	fmt.Println()
	fmt.Println("Run directories are created in:")
	fmt.Printf("  %s\n", runner.RunDir())
}
