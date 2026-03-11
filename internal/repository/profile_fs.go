package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-switch/internal/domain"
)

// ProfileRepositoryFS implements ProfileRepository using the file system
type ProfileRepositoryFS struct {
	profilesDir  string
	settingsPath string
}

// NewProfileRepositoryFS creates a new file system repository
func NewProfileRepositoryFS(profilesDir, settingsPath string) *ProfileRepositoryFS {
	return &ProfileRepositoryFS{
		profilesDir:  profilesDir,
		settingsPath: settingsPath,
	}
}

// Save stores a profile with the given name and settings
func (r *ProfileRepositoryFS) Save(name string, settings map[string]interface{}) error {
	if err := domain.ValidateProfileName(name); err != nil {
		return err
	}

	if err := os.MkdirAll(r.profilesDir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	path := filepath.Join(r.profilesDir, name+".json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	return os.WriteFile(path, append(data, '\n'), 0644)
}

// GetByName loads a single profile by name
func (r *ProfileRepositoryFS) GetByName(name string) (*domain.Profile, error) {
	path := filepath.Join(r.profilesDir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("unmarshal settings: %w", err)
	}

	return &domain.Profile{Name: name, Settings: settings}, nil
}

// List returns all profiles and any errors encountered during loading
func (r *ProfileRepositoryFS) List() ([]domain.Profile, []ListError) {
	entries, err := os.ReadDir(r.profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []ListError{{Name: "", Err: err}}
	}

	var profiles []domain.Profile
	var errs []ListError

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".json")
		p, err := r.GetByName(name)
		if err != nil {
			errs = append(errs, ListError{Name: name, Err: err})
			continue
		}
		profiles = append(profiles, *p)
	}

	return profiles, errs
}

// Delete removes a profile by name
func (r *ProfileRepositoryFS) Delete(name string) error {
	path := filepath.Join(r.profilesDir, name+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return domain.ErrProfileNotFound
		}
		return fmt.Errorf("remove profile %s: %w", name, err)
	}
	return nil
}

// LoadCurrent loads the current Claude settings
func (r *ProfileRepositoryFS) LoadCurrent() (map[string]interface{}, error) {
	data, err := os.ReadFile(r.settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// Apply writes a profile's settings to the current settings file
func (r *ProfileRepositoryFS) Apply(p domain.Profile) error {
	data, err := json.MarshalIndent(p.Settings, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: write to temp file first, then rename
	tmpPath := r.settingsPath + ".tmp"
	if err := os.WriteFile(tmpPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// os.Rename is atomic on the same file system
	if err := os.Rename(tmpPath, r.settingsPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("rename to settings: %w", err)
	}

	return nil
}
