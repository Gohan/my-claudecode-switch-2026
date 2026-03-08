package service

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"claude-switch/internal/domain"
)

// ProfileRunnerExec implements ProfileRunner using exec to run claude
type ProfileRunnerExec struct {
	claudePath string
}

// NewProfileRunnerExec creates a new ProfileRunnerExec
func NewProfileRunnerExec(claudePath string) *ProfileRunnerExec {
	return &ProfileRunnerExec{claudePath: claudePath}
}

// Run starts claude with the profile settings
func (r *ProfileRunnerExec) Run(p domain.Profile) error {
	cmd := r.buildCommand()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// buildCommand builds the exec command based on OS
func (r *ProfileRunnerExec) buildCommand() *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return exec.Command(r.claudePath)
	default:
		return exec.Command(r.claudePath)
	}
}

// ProfileRunnerWithDir implements ProfileRunner with a custom config directory
type ProfileRunnerWithDir struct {
	claudePath string
	configDir  string
}

// NewProfileRunnerWithDir creates a new ProfileRunnerWithDir
func NewProfileRunnerWithDir(claudePath, configDir string) *ProfileRunnerWithDir {
	return &ProfileRunnerWithDir{
		claudePath: claudePath,
		configDir:  configDir,
	}
}

// Run starts claude with CLAUDE_CONFIG_DIR set
func (r *ProfileRunnerWithDir) Run(p domain.Profile) error {
	cmd := r.buildCommand()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// BuildCommand returns the exec.Cmd for use with tea.ExecProcess
func (r *ProfileRunnerWithDir) BuildCommand() *exec.Cmd {
	return r.buildCommand()
}

func (r *ProfileRunnerWithDir) buildCommand() *exec.Cmd {
	env := append(os.Environ(), fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", r.configDir))

	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command(r.claudePath)
		cmd.Env = env
		return cmd
	default:
		cmd := exec.Command(r.claudePath)
		cmd.Env = env
		return cmd
	}
}