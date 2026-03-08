package repository

import "claude-switch/internal/domain"

// ListError records an error when loading a profile
type ListError struct {
	Name string
	Err  error
}

// ProfileRepository defines the data access interface for profiles
type ProfileRepository interface {
	// List returns all profiles and any errors encountered during loading
	List() ([]domain.Profile, []ListError)

	// GetByName loads a single profile by name
	GetByName(name string) (*domain.Profile, error)

	// Save stores a profile with the given name and settings
	Save(name string, settings map[string]interface{}) error

	// Delete removes a profile by name
	Delete(name string) error

	// LoadCurrent loads the current Claude settings
	LoadCurrent() (map[string]interface{}, error)

	// Apply writes a profile's settings to the current settings file
	Apply(p domain.Profile) error
}