package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-switch/internal/domain"
)

// ProfileRepositoryTestSuite tests the Repository behavior
// This can be used to test any implementation of ProfileRepository
func ProfileRepositoryTestSuite(t *testing.T, factory func() ProfileRepository) {
	t.Run("SaveAndGet", func(t *testing.T) {
		repo := factory()
		settings := map[string]interface{}{"model": "opus"}

		err := repo.Save("test", settings)
		require.NoError(t, err)

		p, err := repo.GetByName("test")
		require.NoError(t, err)
		assert.Equal(t, "test", p.Name)
		assert.Equal(t, "opus", p.Settings["model"])
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		repo := factory()

		_, err := repo.GetByName("nonexistent")
		assert.ErrorIs(t, err, domain.ErrProfileNotFound)
	})

	t.Run("ListEmpty", func(t *testing.T) {
		repo := factory()

		profiles, errs := repo.List()
		assert.Empty(t, profiles)
		assert.Empty(t, errs)
	})

	t.Run("ListMultiple", func(t *testing.T) {
		repo := factory()
		repo.Save("p1", map[string]interface{}{"model": "opus"})
		repo.Save("p2", map[string]interface{}{"model": "sonnet"})

		profiles, errs := repo.List()
		assert.Len(t, profiles, 2)
		assert.Empty(t, errs)
	})

	t.Run("Delete", func(t *testing.T) {
		repo := factory()
		repo.Save("todelete", map[string]interface{}{})

		err := repo.Delete("todelete")
		require.NoError(t, err)

		_, err = repo.GetByName("todelete")
		assert.ErrorIs(t, err, domain.ErrProfileNotFound)
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		repo := factory()

		err := repo.Delete("nonexistent")
		assert.Error(t, err)
	})

	t.Run("LoadCurrentEmpty", func(t *testing.T) {
		repo := factory()

		current, err := repo.LoadCurrent()
		require.NoError(t, err)
		assert.NotNil(t, current)
	})

	t.Run("ApplyAndLoadCurrent", func(t *testing.T) {
		repo := factory()

		p := domain.Profile{
			Name: "test",
			Settings: map[string]interface{}{
				"model": "opus",
			},
		}

		err := repo.Apply(p)
		require.NoError(t, err)

		current, err := repo.LoadCurrent()
		require.NoError(t, err)
		assert.Equal(t, "opus", current["model"])
	})

	t.Run("SaveInvalidName", func(t *testing.T) {
		repo := factory()

		err := repo.Save("", map[string]interface{}{})
		assert.Error(t, err)

		err = repo.Save("has/slash", map[string]interface{}{})
		assert.Error(t, err)
	})
}

func TestProfileRepositoryFS(t *testing.T) {
	factory := func() ProfileRepository {
		tmpDir := t.TempDir()
		return NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))
	}

	ProfileRepositoryTestSuite(t, factory)

	t.Run("PersistAcrossInstances", func(t *testing.T) {
		tmpDir := t.TempDir()
		settingsPath := filepath.Join(tmpDir, "settings.json")

		repo1 := NewProfileRepositoryFS(tmpDir, settingsPath)
		repo1.Save("persisted", map[string]interface{}{"model": "opus"})

		repo2 := NewProfileRepositoryFS(tmpDir, settingsPath)
		p, err := repo2.GetByName("persisted")

		require.NoError(t, err)
		assert.Equal(t, "opus", p.Settings["model"])
	})

	t.Run("ListSkipsInvalidFiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))

		// Create valid file
		repo.Save("valid", map[string]interface{}{})

		// Create invalid file
		os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("not json"), 0644)

		profiles, errs := repo.List()
		assert.Len(t, profiles, 1)
		assert.Len(t, errs, 1)
		assert.Equal(t, "invalid", errs[0].Name)
	})

	t.Run("ListSkipsNonJSONFiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))

		repo.Save("valid", map[string]interface{}{})

		// Create non-JSON file
		os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hello"), 0644)

		profiles, errs := repo.List()
		assert.Len(t, profiles, 1)
		assert.Empty(t, errs)
	})

	t.Run("ListSkipsDirectories", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))

		repo.Save("valid", map[string]interface{}{})

		// Create directory with .json name
		os.Mkdir(filepath.Join(tmpDir, "subdir.json"), 0755)

		profiles, errs := repo.List()
		assert.Len(t, profiles, 1)
		assert.Empty(t, errs)
	})

	t.Run("ApplyCreatesFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		settingsPath := filepath.Join(tmpDir, "settings.json")

		// Ensure file doesn't exist
		_, err := os.Stat(settingsPath)
		assert.True(t, os.IsNotExist(err))

		repo := NewProfileRepositoryFS(tmpDir, settingsPath)
		p := domain.Profile{
			Name: "test",
			Settings: map[string]interface{}{
				"model": "opus",
			},
		}

		err = repo.Apply(p)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(settingsPath)
		assert.NoError(t, err)
	})

	t.Run("ApplyOverwrites", func(t *testing.T) {
		tmpDir := t.TempDir()
		settingsPath := filepath.Join(tmpDir, "settings.json")

		repo := NewProfileRepositoryFS(tmpDir, settingsPath)

		// First apply
		p1 := domain.Profile{Name: "test", Settings: map[string]interface{}{"model": "opus"}}
		err := repo.Apply(p1)
		require.NoError(t, err)

		// Second apply
		p2 := domain.Profile{Name: "test", Settings: map[string]interface{}{"model": "sonnet"}}
		err = repo.Apply(p2)
		require.NoError(t, err)

		current, err := repo.LoadCurrent()
		require.NoError(t, err)
		assert.Equal(t, "sonnet", current["model"])
	})

	t.Run("SaveCreatesDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		nestedDir := filepath.Join(tmpDir, "nested", "profiles")
		repo := NewProfileRepositoryFS(nestedDir, filepath.Join(tmpDir, "settings.json"))

		err := repo.Save("test", map[string]interface{}{"model": "opus"})
		require.NoError(t, err)

		// Verify directory was created
		_, err = os.Stat(nestedDir)
		assert.NoError(t, err)
	})

	t.Run("ListWithDirectoryError", func(t *testing.T) {
		// Create a file where directory should be
		tmpDir := t.TempDir()
		repo := NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))

		// Create a profile first
		repo.Save("test", map[string]interface{}{})

		profiles, errs := repo.List()
		assert.Len(t, profiles, 1)
		assert.Empty(t, errs)
	})

	t.Run("LoadCurrentInvalidJSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		settingsPath := filepath.Join(tmpDir, "settings.json")

		// Write invalid JSON
		os.WriteFile(settingsPath, []byte("not json"), 0644)

		repo := NewProfileRepositoryFS(tmpDir, settingsPath)
		_, err := repo.LoadCurrent()
		assert.Error(t, err)
	})

	t.Run("SaveWithNestedSettings", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))

		settings := map[string]interface{}{
			"model": "opus",
			"env": map[string]interface{}{
				"ANTHROPIC_BASE_URL": "https://api.example.com",
			},
		}

		err := repo.Save("nested", settings)
		require.NoError(t, err)

		p, err := repo.GetByName("nested")
		require.NoError(t, err)
		assert.Equal(t, "opus", p.Settings["model"])

		env, ok := p.Settings["env"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "https://api.example.com", env["ANTHROPIC_BASE_URL"])
	})

	t.Run("ApplyWithNestedSettings", func(t *testing.T) {
		tmpDir := t.TempDir()
		settingsPath := filepath.Join(tmpDir, "settings.json")
		repo := NewProfileRepositoryFS(tmpDir, settingsPath)

		p := domain.Profile{
			Name: "test",
			Settings: map[string]interface{}{
				"model": "opus",
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL": "https://api.example.com",
				},
			},
		}

		err := repo.Apply(p)
		require.NoError(t, err)

		current, err := repo.LoadCurrent()
		require.NoError(t, err)
		assert.Equal(t, "opus", current["model"])
	})

	t.Run("DeleteNonExistentReturnsProfileNotFound", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))

		err := repo.Delete("nonexistent")
		assert.ErrorIs(t, err, domain.ErrProfileNotFound)
	})

	t.Run("GetByNameWithInvalidJSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))

		// Create invalid JSON file
		os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("not json"), 0644)

		_, err := repo.GetByName("invalid")
		assert.Error(t, err)
	})
}