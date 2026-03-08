package service

import (
	"fmt"
	"reflect"

	"claude-switch/internal/domain"
	"claude-switch/internal/repository"
)

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
}

// ProfileRunner defines the interface for running a profile
type ProfileRunner interface {
	Run(p domain.Profile) error
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