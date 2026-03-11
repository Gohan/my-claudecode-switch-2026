package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfile_CanBeCreatedWithNameAndSettings(t *testing.T) {
	p := Profile{
		Name: "my-profile",
		Settings: map[string]interface{}{
			"model": "opus",
		},
	}

	assert.Equal(t, "my-profile", p.Name)
	assert.Equal(t, "opus", p.Settings["model"])
}

func TestProfile_IsEmptyWhenNameIsBlank(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"", true},
		{"   ", true},
		{"profile", false},
		{"  profile  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Profile{Name: tt.name}
			assert.Equal(t, tt.expected, p.IsEmpty())
		})
	}
}

func TestProfileName_Validation(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		errMsg  string
	}{
		{"valid-name", false, ""},
		{"valid_name", false, ""},
		{"ValidName123", false, ""},
		{"", true, "name cannot be empty"},
		{"   ", true, "name cannot be empty"},
		{"has/slash", true, "invalid character: /"},
		{"has\\backslash", true, "invalid character: \\"},
		{"has*star", true, "invalid character: *"},
		{"has?question", true, "invalid character: ?"},
		{"has:colon", true, "invalid character: :"},
		{"has\"quote", true, "invalid character: \""},
		{"has<less>", true, "invalid character: <"},
		{"has|pipe", true, "invalid character: |"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfileName(tt.name)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDiffEntry_RepresentsChanges(t *testing.T) {
	entry := DiffEntry{
		Key:      "model",
		OldValue: "opus",
		NewValue: "sonnet",
		Status:   DiffModified,
	}

	assert.Equal(t, "model", entry.Key)
	assert.Equal(t, DiffModified, entry.Status)
}

func TestDiffStatus_Values(t *testing.T) {
	assert.Equal(t, DiffStatus(0), DiffUnchanged)
	assert.Equal(t, DiffStatus(1), DiffModified)
	assert.Equal(t, DiffStatus(2), DiffAdded)
	assert.Equal(t, DiffStatus(3), DiffRemoved)
}

func TestFlatten_FlatMap(t *testing.T) {
	m := map[string]interface{}{
		"model": "opus",
		"key":   "value",
	}

	result := Flatten(m, "")

	assert.Equal(t, "opus", result["model"])
	assert.Equal(t, "value", result["key"])
	assert.Len(t, result, 2)
}

func TestFlatten_NestedMap(t *testing.T) {
	m := map[string]interface{}{
		"model": "opus",
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL": "https://api.example.com",
			"API_TIMEOUT_MS":      "3000000",
		},
	}

	result := Flatten(m, "")

	assert.Equal(t, "opus", result["model"])
	assert.Equal(t, "https://api.example.com", result["env.ANTHROPIC_BASE_URL"])
	assert.Equal(t, "3000000", result["env.API_TIMEOUT_MS"])
}

func TestFlatten_WithPrefix(t *testing.T) {
	m := map[string]interface{}{
		"key": "value",
	}

	result := Flatten(m, "prefix")

	assert.Equal(t, "value", result["prefix.key"])
}

func TestDiff_NoChange(t *testing.T) {
	current := map[string]interface{}{"model": "opus"}
	target := map[string]interface{}{"model": "opus"}

	entries := Diff(current, target)

	assert.Len(t, entries, 1)
	assert.Equal(t, DiffUnchanged, entries[0].Status)
}

func TestDiff_Modified(t *testing.T) {
	current := map[string]interface{}{"model": "opus"}
	target := map[string]interface{}{"model": "sonnet"}

	entries := Diff(current, target)

	assert.Len(t, entries, 1)
	assert.Equal(t, DiffModified, entries[0].Status)
	assert.Equal(t, "opus", entries[0].OldValue)
	assert.Equal(t, "sonnet", entries[0].NewValue)
}

func TestDiff_Added(t *testing.T) {
	current := map[string]interface{}{}
	target := map[string]interface{}{"model": "opus"}

	entries := Diff(current, target)

	assert.Len(t, entries, 1)
	assert.Equal(t, DiffAdded, entries[0].Status)
	assert.Equal(t, "opus", entries[0].NewValue)
}

func TestDiff_Removed(t *testing.T) {
	current := map[string]interface{}{"model": "opus"}
	target := map[string]interface{}{}

	entries := Diff(current, target)

	assert.Len(t, entries, 1)
	assert.Equal(t, DiffRemoved, entries[0].Status)
	assert.Equal(t, "opus", entries[0].OldValue)
}

func TestIsActive_Matching(t *testing.T) {
	current := map[string]interface{}{
		"model": "opus",
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL": "https://api.example.com",
		},
	}
	p := Profile{
		Name: "test",
		Settings: map[string]interface{}{
			"model": "opus",
			"env": map[string]interface{}{
				"ANTHROPIC_BASE_URL": "https://api.example.com",
			},
		},
	}

	assert.True(t, IsActive(current, p))
}

func TestIsActive_NotMatching(t *testing.T) {
	current := map[string]interface{}{"model": "opus"}
	p := Profile{
		Name:     "test",
		Settings: map[string]interface{}{"model": "sonnet"},
	}

	assert.False(t, IsActive(current, p))
}

func TestIsActive_DifferentKeys(t *testing.T) {
	current := map[string]interface{}{"model": "opus", "key1": "value1"}
	p := Profile{
		Name:     "test",
		Settings: map[string]interface{}{"model": "opus", "key2": "value2"},
	}

	assert.False(t, IsActive(current, p))
}

func TestIsActive_DifferentLengths(t *testing.T) {
	current := map[string]interface{}{"model": "opus", "extra": "value"}
	p := Profile{
		Name:     "test",
		Settings: map[string]interface{}{"model": "opus"},
	}

	assert.False(t, IsActive(current, p))
}

func TestMaskSensitive_Token(t *testing.T) {
	assert.Equal(t, "abcd****wxyz", MaskSensitive("api_token", "abcdefghiwxyz"))
	assert.Equal(t, "****", MaskSensitive("api_token", "short"))
}

func TestMaskSensitive_Key(t *testing.T) {
	assert.Equal(t, "abcd****wxyz", MaskSensitive("api_key", "abcdefghiwxyz"))
}

func TestMaskSensitive_NonSensitive(t *testing.T) {
	assert.Equal(t, "plainvalue", MaskSensitive("model", "plainvalue"))
}

func TestGetSummary_ModelOnly(t *testing.T) {
	settings := map[string]interface{}{
		"model": "opus",
	}

	model, baseURL := GetSummary(settings)

	assert.Equal(t, "opus", model)
	assert.Empty(t, baseURL)
}

func TestGetSummary_WithBaseURL(t *testing.T) {
	settings := map[string]interface{}{
		"model": "sonnet",
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL": "https://api.example.com",
		},
	}

	model, baseURL := GetSummary(settings)

	assert.Equal(t, "sonnet", model)
	assert.Equal(t, "https://api.example.com", baseURL)
}

func TestGetModelMapping_AllModels(t *testing.T) {
	settings := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "haiku-model",
			"ANTHROPIC_DEFAULT_SONNET_MODEL": "sonnet-model",
			"ANTHROPIC_DEFAULT_OPUS_MODEL":   "opus-model",
		},
	}

	mapping := GetModelMapping(settings)

	assert.Equal(t, "haiku-model", mapping["haiku"])
	assert.Equal(t, "sonnet-model", mapping["sonnet"])
	assert.Equal(t, "opus-model", mapping["opus"])
}

func TestGetModelMapping_NoEnv(t *testing.T) {
	settings := map[string]interface{}{
		"model": "opus",
	}

	mapping := GetModelMapping(settings)

	assert.Empty(t, mapping)
}
