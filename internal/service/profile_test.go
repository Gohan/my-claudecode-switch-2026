package service

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-switch/internal/domain"
	"claude-switch/internal/repository"
)

// memoryRepo is an in-memory implementation for unit testing
type memoryRepo struct {
	profiles map[string]domain.Profile
	current  map[string]interface{}
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		profiles: make(map[string]domain.Profile),
		current:  make(map[string]interface{}),
	}
}

func (m *memoryRepo) Save(name string, settings map[string]interface{}) error {
	if err := domain.ValidateProfileName(name); err != nil {
		return err
	}
	m.profiles[name] = domain.Profile{Name: name, Settings: settings}
	return nil
}

func (m *memoryRepo) GetByName(name string) (*domain.Profile, error) {
	p, ok := m.profiles[name]
	if !ok {
		return nil, domain.ErrProfileNotFound
	}
	return &p, nil
}

func (m *memoryRepo) List() ([]domain.Profile, []repository.ListError) {
	var result []domain.Profile
	for _, p := range m.profiles {
		result = append(result, p)
	}
	return result, nil
}

func (m *memoryRepo) Delete(name string) error {
	if _, ok := m.profiles[name]; !ok {
		return domain.ErrProfileNotFound
	}
	delete(m.profiles, name)
	return nil
}

func (m *memoryRepo) LoadCurrent() (map[string]interface{}, error) {
	return m.current, nil
}

func (m *memoryRepo) Apply(p domain.Profile) error {
	m.current = p.Settings
	return nil
}

// memoryRunner is an in-memory implementation for testing
type memoryRunner struct {
	runs []string
}

func (m *memoryRunner) Run(p domain.Profile) error {
	m.runs = append(m.runs, p.Name)
	return nil
}

func (m *memoryRunner) PrepareAndBuild(name string, settings map[string]interface{}) (*exec.Cmd, error) {
	return nil, nil
}

func (m *memoryRunner) RunDir() string {
	return "/fake/run/dir"
}

func TestProfileService_CreatesProfileWithValidData(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	err := svc.Create("my-profile", map[string]interface{}{
		"model": "opus",
	})

	require.NoError(t, err)

	profiles, _ := svc.List()
	assert.Len(t, profiles, 1)
	assert.Equal(t, "my-profile", profiles[0].Name)
}

func TestProfileService_RejectsDuplicateName(t *testing.T) {
	repo := newMemoryRepo()
	repo.Save("existing", map[string]interface{}{})

	svc := NewProfileService(repo, nil)
	err := svc.Create("existing", map[string]interface{}{})

	assert.ErrorIs(t, err, domain.ErrProfileExists)
}

func TestProfileService_ValidatesName(t *testing.T) {
	tests := []struct {
		name string
	}{
		{""},
		{"has/slash"},
		{"has*star"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewProfileService(newMemoryRepo(), nil)
			err := svc.Create(tt.name, map[string]interface{}{})
			assert.ErrorIs(t, err, domain.ErrInvalidName)
		})
	}
}

func TestProfileService_RunCallsRunner(t *testing.T) {
	repo := newMemoryRepo()
	runner := &memoryRunner{}
	svc := NewProfileService(repo, runner)

	repo.Save("test", map[string]interface{}{"model": "opus"})

	err := svc.Run("test")

	require.NoError(t, err)
	assert.Equal(t, []string{"test"}, runner.runs)
}

func TestProfileService_ApplySavesToCurrent(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	repo.Save("prod", map[string]interface{}{"model": "opus"})

	err := svc.Apply("prod")

	require.NoError(t, err)
	current, _ := repo.LoadCurrent()
	assert.Equal(t, "opus", current["model"])
}

func TestProfileService_IsActive(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	repo.Save("active", map[string]interface{}{"model": "opus"})
	svc.Apply("active")

	p, _ := repo.GetByName("active")
	assert.True(t, svc.IsActive(*p))
}

func TestProfileService_IsActive_FalseWhenDifferent(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	repo.Save("profile1", map[string]interface{}{"model": "opus"})
	repo.Save("profile2", map[string]interface{}{"model": "sonnet"})
	svc.Apply("profile1")

	p, _ := repo.GetByName("profile2")
	assert.False(t, svc.IsActive(*p))
}

func TestProfileService_Delete(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	repo.Save("todelete", map[string]interface{}{})

	err := svc.Delete("todelete")
	require.NoError(t, err)

	_, err = svc.GetByName("todelete")
	assert.ErrorIs(t, err, domain.ErrProfileNotFound)
}

func TestProfileService_DeleteNonExistent(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	err := svc.Delete("nonexistent")
	assert.Error(t, err)
}

func TestProfileService_Update(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	repo.Save("test", map[string]interface{}{"model": "opus"})

	err := svc.Update("test", map[string]interface{}{"model": "sonnet"})
	require.NoError(t, err)

	p, err := svc.GetByName("test")
	require.NoError(t, err)
	assert.Equal(t, "sonnet", p.Settings["model"])
}

func TestProfileService_UpdateNonExistent(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	err := svc.Update("nonexistent", map[string]interface{}{})
	assert.Error(t, err)
}

func TestProfileService_ListWithErrors(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	repo.Save("p1", map[string]interface{}{"model": "opus"})
	repo.Save("p2", map[string]interface{}{"model": "sonnet"})

	profiles, err := svc.List()
	require.NoError(t, err)
	assert.Len(t, profiles, 2)
}

func TestProfileService_GetByName(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	repo.Save("test", map[string]interface{}{"model": "opus"})

	p, err := svc.GetByName("test")
	require.NoError(t, err)
	assert.Equal(t, "test", p.Name)
	assert.Equal(t, "opus", p.Settings["model"])
}

func TestProfileService_GetByNameNonExistent(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	_, err := svc.GetByName("nonexistent")
	assert.ErrorIs(t, err, domain.ErrProfileNotFound)
}

func TestProfileService_LoadCurrent(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	repo.Save("test", map[string]interface{}{"model": "opus"})
	svc.Apply("test")

	current, err := svc.LoadCurrent()
	require.NoError(t, err)
	assert.Equal(t, "opus", current["model"])
}

func TestProfileService_RunNonExistent(t *testing.T) {
	repo := newMemoryRepo()
	runner := &memoryRunner{}
	svc := NewProfileService(repo, runner)

	err := svc.Run("nonexistent")
	assert.Error(t, err)
	assert.Empty(t, runner.runs)
}

func TestProfileService_IsActive_EmptyCurrent(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	repo.Save("test", map[string]interface{}{"model": "opus"})

	p, _ := repo.GetByName("test")
	assert.False(t, svc.IsActive(*p))
}

func TestProfileService_IsActive_WithNestedSettings(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	settings := map[string]interface{}{
		"model": "opus",
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL": "https://api.example.com",
		},
	}

	repo.Save("nested", settings)
	svc.Apply("nested")

	p, _ := repo.GetByName("nested")
	assert.True(t, svc.IsActive(*p))
}

func TestProfileService_IsActive_WithArray(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	settings := map[string]interface{}{
		"model": "opus",
		"items": []interface{}{"a", "b", "c"},
	}

	repo.Save("with-array", settings)
	svc.Apply("with-array")

	p, _ := repo.GetByName("with-array")
	assert.True(t, svc.IsActive(*p))
}

func TestProfileService_IsActive_WithFloatAsInt(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	// JSON decodes numbers as float64
	settings := map[string]interface{}{
		"model": "opus",
		"count": float64(42),
	}

	repo.Save("with-int", settings)
	svc.Apply("with-int")

	p, _ := repo.GetByName("with-int")
	assert.True(t, svc.IsActive(*p))
}

func TestProfileService_IsActive_WithFloatValue(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	settings := map[string]interface{}{
		"model":  "opus",
		"ratio":  3.14159,
		"count":  float64(100),
		"nested": map[string]interface{}{"value": float64(50)},
	}

	repo.Save("with-float", settings)
	svc.Apply("with-float")

	p, _ := repo.GetByName("with-float")
	assert.True(t, svc.IsActive(*p))
}

func TestProfileService_ListWarningError(t *testing.T) {
	err := &ListWarningError{Errors: []repository.ListError{
		{Name: "bad", Err: fmt.Errorf("parse error")},
	}}

	assert.Contains(t, err.Error(), "1 profiles failed to load")
	assert.True(t, errors.Is(err, &ListWarningError{}))
}

func TestProfileService_ApplyNonExistent(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	err := svc.Apply("nonexistent")
	assert.Error(t, err)
}

func TestNewProfileService_NilRunner(t *testing.T) {
	repo := newMemoryRepo()
	svc := NewProfileService(repo, nil)

	assert.NotNil(t, svc)
}

func TestNewProfileRunnerExec(t *testing.T) {
	runner := NewProfileRunnerExec("claude")
	assert.NotNil(t, runner)
}

func TestNewProfileRunnerWithDir(t *testing.T) {
	runner := NewProfileRunnerWithDir("claude", "/tmp/config")
	assert.NotNil(t, runner)
	assert.NotNil(t, runner.RunDir())
}
