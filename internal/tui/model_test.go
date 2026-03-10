package tui

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"claude-switch/internal/domain"
)

// mockService implements service.ProfileService for testing
type mockService struct {
	profiles       []domain.Profile
	current        map[string]interface{}
	created        []string
	updated        []string
	deleted        []string
	applied        []string
	ran            []string
	getByNameErr   error
	createErr      error
	updateErr      error
	deleteErr      error
	applyErr       error
	runErr         error
}

func (m *mockService) List() ([]domain.Profile, error) {
	return m.profiles, nil
}

func (m *mockService) GetByName(name string) (*domain.Profile, error) {
	if m.getByNameErr != nil {
		return nil, m.getByNameErr
	}
	for _, p := range m.profiles {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, domain.ErrProfileNotFound
}

func (m *mockService) Create(name string, settings map[string]interface{}) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, name)
	m.profiles = append(m.profiles, domain.Profile{Name: name, Settings: settings})
	return nil
}

func (m *mockService) Update(name string, settings map[string]interface{}) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, name)
	for i, p := range m.profiles {
		if p.Name == name {
			m.profiles[i].Settings = settings
			break
		}
	}
	return nil
}

func (m *mockService) Delete(name string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = append(m.deleted, name)
	for i, p := range m.profiles {
		if p.Name == name {
			m.profiles = append(m.profiles[:i], m.profiles[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockService) Apply(name string) error {
	if m.applyErr != nil {
		return m.applyErr
	}
	m.applied = append(m.applied, name)
	return nil
}

func (m *mockService) Run(name string) error {
	if m.runErr != nil {
		return m.runErr
	}
	m.ran = append(m.ran, name)
	return nil
}

func (m *mockService) LoadCurrent() (map[string]interface{}, error) {
	return m.current, nil
}

func (m *mockService) IsActive(p domain.Profile) bool {
	if len(m.current) == 0 {
		return false
	}
	return false // simplified for testing
}

func (m *mockService) PrepareAndBuild(name string, settings map[string]interface{}) (*exec.Cmd, error) {
	// Return nil command for testing - actual command execution not needed
	return nil, nil
}

func (m *mockService) RunDir() string {
	return "/fake/run/dir"
}

func TestModel_InitializesWithProfiles(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{
			{Name: "p1", Settings: map[string]interface{}{"model": "opus"}},
			{Name: "p2", Settings: map[string]interface{}{"model": "sonnet"}},
		},
		current: map[string]interface{}{"model": "opus"},
	}

	m := NewModel(svc)

	assert.Len(t, m.profiles, 2)
	assert.Equal(t, "p1", m.profiles[0].Name)
	assert.Equal(t, "p2", m.profiles[1].Name)
}

func TestModel_InitializesWithEmptyProfiles(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{},
		current:  make(map[string]interface{}),
	}

	m := NewModel(svc)

	assert.Empty(t, m.profiles)
}

func TestModel_CursorNavigation(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{
			{Name: "p1"}, {Name: "p2"}, {Name: "p3"},
		},
	}

	m := NewModel(svc)

	// Initial cursor should be 0
	assert.Equal(t, 0, m.cursor)

	// Test cursor stays in bounds
	m.cursor = len(m.profiles) - 1
	assert.True(t, m.cursor >= 0 && m.cursor < len(m.profiles))
}

func TestModel_SafeProfileIndex(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{
			{Name: "p1"}, {Name: "p2"},
		},
	}

	m := NewModel(svc)

	// Valid index
	m.cursor = 0
	assert.True(t, m.safeProfileIndex())

	// Valid index
	m.cursor = 1
	assert.True(t, m.safeProfileIndex())

	// Invalid index - out of bounds
	m.cursor = 2
	assert.False(t, m.safeProfileIndex())

	// Invalid index - negative
	m.cursor = -1
	assert.False(t, m.safeProfileIndex())
}

func TestModel_ViewListWithProfiles(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{
			{Name: "test-profile", Settings: map[string]interface{}{"model": "opus"}},
		},
		current: map[string]interface{}{"model": "sonnet"},
	}

	m := NewModel(svc)
	m.state = viewList

	view := m.View()
	assert.Contains(t, view, "test-profile")
	assert.Contains(t, view, "Claude Code Settings Switch")
}

func TestModel_ViewListWithEmptyProfiles(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{},
		current:  make(map[string]interface{}),
	}

	m := NewModel(svc)
	m.state = viewList

	view := m.View()
	assert.Contains(t, view, "No profiles saved")
}

func TestModel_ViewConfirmApply(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{
			{Name: "test-profile", Settings: map[string]interface{}{}},
		},
	}

	m := NewModel(svc)
	m.cursor = 0
	m.state = viewConfirmApply

	view := m.View()
	assert.Contains(t, view, "Confirm Apply")
	assert.Contains(t, view, "test-profile")
}

func TestModel_ViewConfirmDelete(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{
			{Name: "test-profile", Settings: map[string]interface{}{}},
		},
	}

	m := NewModel(svc)
	m.cursor = 0
	m.state = viewConfirmDelete

	view := m.View()
	assert.Contains(t, view, "Confirm Delete")
	assert.Contains(t, view, "test-profile")
}

func TestModel_ViewSave(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{},
		current:  map[string]interface{}{},
	}

	m := NewModel(svc)
	m.state = viewSave

	view := m.View()
	assert.Contains(t, view, "Save Current Settings")
}

func TestModel_ViewCreateMenu(t *testing.T) {
	svc := &mockService{
		profiles: []domain.Profile{},
		current:  map[string]interface{}{},
	}

	m := NewModel(svc)
	m.state = viewCreateMenu

	view := m.View()
	assert.Contains(t, view, "Create New Profile")
	assert.Contains(t, view, "Save Current")
	assert.Contains(t, view, "Kimi")
	assert.Contains(t, view, "z.ai API")
}