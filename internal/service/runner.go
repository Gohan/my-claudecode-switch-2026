package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"claude-switch/internal/domain"
)

// ProfileRunner defines the interface for running a profile
type ProfileRunner interface {
	Run(p domain.Profile) error
	PrepareAndBuild(profileName string, settings map[string]interface{}) (*exec.Cmd, error)
	RunDir() string
}

// profileRunnerBase provides common functionality for profile runners
type profileRunnerBase struct {
	claudePath string
	runDir     string
}

// newProfileRunnerBase creates a new profileRunnerBase
func newProfileRunnerBase(claudePath string) *profileRunnerBase {
	home, _ := os.UserHomeDir()
	return &profileRunnerBase{
		claudePath: claudePath,
		runDir:    filepath.Join(home, ".claude-switch", "runs"),
	}
}

// RunDir returns the run directory path
func (r *profileRunnerBase) RunDir() string {
	return r.runDir
}

// prepareRunDir creates the run directory and writes settings
func (r *profileRunnerBase) prepareRunDir(profileName string, settings map[string]interface{}) (string, error) {
	runDir := filepath.Join(r.runDir, profileName)

	// Create directory
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create run directory: %w", err)
	}

	// Write settings.json
	settingsPath := filepath.Join(runDir, "settings.json")
	if err := writeSettings(settingsPath, settings); err != nil {
		return "", fmt.Errorf("failed to write settings: %w", err)
	}

	return runDir, nil
}

// writeSettings writes settings to file
func writeSettings(path string, settings map[string]interface{}) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// buildCommand builds the exec command with CLAUDE_CONFIG_DIR
func (r *profileRunnerBase) buildCommand(configDir string) *exec.Cmd {
	fmt.Printf("[DEBUG] Using CLAUDE_CONFIG_DIR=%s\n", configDir)
	env := append(os.Environ(), "CLAUDE_CONFIG_DIR="+configDir)

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

// ProfileRunnerExec implements ProfileRunner using exec to run claude
type ProfileRunnerExec struct {
	*profileRunnerBase
}

// NewProfileRunnerExec creates a new ProfileRunnerExec
func NewProfileRunnerExec(claudePath string) *ProfileRunnerExec {
	return &ProfileRunnerExec{
		profileRunnerBase: newProfileRunnerBase(claudePath),
	}
}

// Run starts claude with the profile settings
func (r *ProfileRunnerExec) Run(p domain.Profile) error {
	cmd, err := r.PrepareAndBuild(p.Name, p.Settings)
	if err != nil {
		return err
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PrepareAndBuild prepares the run directory and builds the command
func (r *ProfileRunnerExec) PrepareAndBuild(profileName string, settings map[string]interface{}) (*exec.Cmd, error) {
	runDir, err := r.prepareRunDir(profileName, settings)
	if err != nil {
		return nil, err
	}
	return r.buildCommand(runDir), nil
}

// ProfileRunnerWithDir implements ProfileRunner with a custom config directory
type ProfileRunnerWithDir struct {
	*profileRunnerBase
	configDir string
}

// NewProfileRunnerWithDir creates a new ProfileRunnerWithDir
func NewProfileRunnerWithDir(claudePath, configDir string) *ProfileRunnerWithDir {
	return &ProfileRunnerWithDir{
		profileRunnerBase: newProfileRunnerBase(claudePath),
		configDir:         configDir,
	}
}

// Run starts claude with CLAUDE_CONFIG_DIR set
func (r *ProfileRunnerWithDir) Run(p domain.Profile) error {
	cmd := r.buildCommand(r.configDir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PrepareAndBuild prepares the run directory and builds the command
func (r *ProfileRunnerWithDir) PrepareAndBuild(profileName string, settings map[string]interface{}) (*exec.Cmd, error) {
	runDir, err := r.prepareRunDir(profileName, settings)
	if err != nil {
		return nil, err
	}
	return r.buildCommand(runDir), nil
}