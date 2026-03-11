package service

import (
	"fmt"
	"os/exec"
	"reflect"

	"claude-switch/internal/domain"
	"claude-switch/internal/repository"
)

// DefaultProfileConfig 定义预设配置的模板
type DefaultProfileConfig struct {
	Name        string
	DisplayName string
	BaseURL     string
	Timeout     string
	ModelMapping map[string]string
	EnvVars     map[string]string
}

// PredefinedProfiles 返回所有可用的预设配置列表
func PredefinedProfiles() []DefaultProfileConfig {
	return []DefaultProfileConfig{
		{
			Name:        "zai",
			DisplayName: "z.ai Coding Plan",
			BaseURL:     "https://api.z.ai/api/anthropic",
			Timeout:     "3000000",
			ModelMapping: map[string]string{
				"haiku":  "glm-4.5-air",
				"sonnet": "glm-4.7",
				"opus":   "glm-5",
			},
			EnvVars: map[string]string{
				"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
			},
		},
		{
			Name:        "tencent",
			DisplayName: "Tencent Cloud CodingPlan",
			BaseURL:     "https://api.lkeap.cloud.tencent.com/coding/anthropic",
			Timeout:     "3000000",
			ModelMapping: map[string]string{
				"haiku":  "tc-code-latest",
				"sonnet": "kimi-k2.5",
				"opus":   "minimax-m2.5",
			},
			EnvVars: map[string]string{
				"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
			},
		},
		{
			Name:        "kimi",
			DisplayName: "Kimi Official API",
			BaseURL:     "https://api.kimi.com/coding/",
			Timeout:     "3000000",
			ModelMapping: map[string]string{},
			EnvVars: map[string]string{
				"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
			},
		},
		{
			Name:        "ali",
			DisplayName: "Ali BaiLian CodingPlan",
			BaseURL:     "https://coding.dashscope.aliyuncs.com/apps/anthropic",
			Timeout:     "3000000",
			ModelMapping: map[string]string{
				"haiku":  "glm-5",
				"sonnet": "kimi-k2.5",
				"opus":   "MiniMax-M2.5",
			},
			EnvVars: map[string]string{
				"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
			},
		},
	}
}

// GetDefaultProfileSettings 根据预设名称生成 settings map
func GetDefaultProfileSettings(configName string) map[string]interface{} {
	var config *DefaultProfileConfig

	profiles := PredefinedProfiles()
	for i := range profiles {
		if profiles[i].Name == configName {
			config = &profiles[i]
			break
		}
	}

	if config == nil {
		return nil
	}

	// 构建 env 配置
	env := make(map[string]interface{})
	for k, v := range config.EnvVars {
		env[k] = v
	}
	env["ANTHROPIC_BASE_URL"] = config.BaseURL
	env["API_TIMEOUT_MS"] = config.Timeout

	// 添加模型映射
	for modelTier, modelName := range config.ModelMapping {
		switch modelTier {
		case "haiku":
			env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = modelName
		case "sonnet":
			env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = modelName
		case "opus":
			env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = modelName
		}
	}

	return map[string]interface{}{
		"includeCoAuthoredBy": false,
		"model":               "opus",
		"env":                 env,
	}
}

// ProfileService defines the business logic interface for profile management
type ProfileService interface {
	List() ([]domain.Profile, error)
	GetByName(name string) (*domain.Profile, error)
	Create(name string, settings map[string]interface{}) error
	Update(name string, settings map[string]interface{}) error
	Delete(name string) error
	Apply(name string) error
	Run(name string) error
	LoadCurrent() (map[string]interface{}, error)
	IsActive(p domain.Profile) bool
	PrepareAndBuild(name string, settings map[string]interface{}) (*exec.Cmd, error)
	RunDir() string
}

// profileService implements ProfileService
type profileService struct {
	repo   repository.ProfileRepository
	runner ProfileRunner
}

// NewProfileService creates a new ProfileService
func NewProfileService(repo repository.ProfileRepository, runner ProfileRunner) ProfileService {
	return &profileService{
		repo:   repo,
		runner: runner,
	}
}

// Create creates a new profile
func (s *profileService) Create(name string, settings map[string]interface{}) error {
	if _, err := s.repo.GetByName(name); err == nil {
		return domain.ErrProfileExists
	}
	if err := domain.ValidateProfileName(name); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidName, err)
	}
	return s.repo.Save(name, settings)
}

// List returns all profiles
func (s *profileService) List() ([]domain.Profile, error) {
	profiles, errs := s.repo.List()
	if len(errs) > 0 {
		return profiles, &ListWarningError{Errors: errs}
	}
	return profiles, nil
}

// ListWarningError represents a warning that some profiles failed to load
type ListWarningError struct {
	Errors []repository.ListError
}

func (e *ListWarningError) Error() string {
	return fmt.Sprintf("%d profiles failed to load", len(e.Errors))
}

func (e *ListWarningError) Is(target error) bool {
	_, ok := target.(*ListWarningError)
	return ok
}

// GetByName returns a profile by name
func (s *profileService) GetByName(name string) (*domain.Profile, error) {
	return s.repo.GetByName(name)
}

// Update updates an existing profile
func (s *profileService) Update(name string, settings map[string]interface{}) error {
	if _, err := s.repo.GetByName(name); err != nil {
		return err
	}
	return s.repo.Save(name, settings)
}

// Delete deletes a profile
func (s *profileService) Delete(name string) error {
	return s.repo.Delete(name)
}

// Apply applies a profile to current settings
func (s *profileService) Apply(name string) error {
	p, err := s.repo.GetByName(name)
	if err != nil {
		return err
	}
	return s.repo.Apply(*p)
}

// Run runs a profile
func (s *profileService) Run(name string) error {
	p, err := s.repo.GetByName(name)
	if err != nil {
		return err
	}
	return s.runner.Run(*p)
}

// LoadCurrent loads current settings
func (s *profileService) LoadCurrent() (map[string]interface{}, error) {
	return s.repo.LoadCurrent()
}

// IsActive checks if a profile is currently active
func (s *profileService) IsActive(p domain.Profile) bool {
	current, err := s.repo.LoadCurrent()
	if err != nil || len(current) == 0 {
		return false
	}

	// Deep comparison with normalized JSON types
	normalizedCurrent := normalizeSettings(current)
	normalizedProfile := normalizeSettings(p.Settings)

	return reflect.DeepEqual(normalizedCurrent, normalizedProfile)
}

// normalizeSettings normalizes settings to handle JSON deserialization type issues
func normalizeSettings(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = normalizeValue(v)
	}
	return result
}

func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return normalizeSettings(val)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = normalizeValue(item)
		}
		return result
	case float64:
		// JSON numbers are parsed as float64, convert to int64 for integer cases
		if val == float64(int64(val)) {
			return int64(val)
		}
		return val
	default:
		return v
	}
}

// DefaultZAIProfile returns a profile template with z.ai configuration
func DefaultZAIProfile() map[string]interface{} {
	return GetDefaultProfileSettings("zai")
}

// DefaultTencentCloudProfile returns a profile template with Tencent Cloud configuration
func DefaultTencentCloudProfile() map[string]interface{} {
	return GetDefaultProfileSettings("tencent")
}

// DefaultKimiProfile returns a profile template with Kimi configuration
func DefaultKimiProfile() map[string]interface{} {
	return GetDefaultProfileSettings("kimi")
}

// DefaultAliProfile returns a profile template with Ali BaiLian configuration
func DefaultAliProfile() map[string]interface{} {
	return GetDefaultProfileSettings("ali")
}

// PrepareAndBuild prepares the run directory and builds the command
func (s *profileService) PrepareAndBuild(name string, settings map[string]interface{}) (*exec.Cmd, error) {
	return s.runner.PrepareAndBuild(name, settings)
}

// RunDir returns the run directory path
func (s *profileService) RunDir() string {
	return s.runner.RunDir()
}
