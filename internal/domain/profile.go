package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Profile represents a Claude Code configuration
type Profile struct {
	Name     string
	Settings map[string]interface{}
}

// IsEmpty checks if the Profile is empty (name is blank)
func (p Profile) IsEmpty() bool {
	return strings.TrimSpace(p.Name) == ""
}

// DiffStatus represents diff status
type DiffStatus int

const (
	DiffUnchanged DiffStatus = iota
	DiffModified
	DiffAdded
	DiffRemoved
)

// DiffEntry represents a single configuration change
type DiffEntry struct {
	Key      string
	OldValue string
	NewValue string
	Status   DiffStatus
}

// invalidNameChars defines characters not allowed in profile names
var invalidNameChars = regexp.MustCompile(`[\\/:*?"<>|]`)

// ValidateProfileName validates a profile name
func ValidateProfileName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidName)
	}
	if match := invalidNameChars.FindString(name); match != "" {
		return fmt.Errorf("%w: invalid character: %s", ErrInvalidName, match)
	}
	return nil
}

// Flatten converts a nested map to a flat map with dot-separated keys
func Flatten(m map[string]interface{}, prefix string) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			for fk, fv := range Flatten(val, key) {
				result[fk] = fv
			}
		default:
			result[key] = fmt.Sprintf("%v", val)
		}
	}
	return result
}

// Diff compares two settings maps and returns the differences
func Diff(current, target map[string]interface{}) []DiffEntry {
	flatCurrent := Flatten(current, "")
	flatTarget := Flatten(target, "")

	allKeys := make(map[string]bool)
	for k := range flatCurrent {
		allKeys[k] = true
	}
	for k := range flatTarget {
		allKeys[k] = true
	}

	var keys []string
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var entries []DiffEntry
	for _, k := range keys {
		cv, inCurrent := flatCurrent[k]
		tv, inTarget := flatTarget[k]

		switch {
		case inCurrent && inTarget && cv == tv:
			entries = append(entries, DiffEntry{Key: k, OldValue: cv, NewValue: tv, Status: DiffUnchanged})
		case inCurrent && inTarget:
			entries = append(entries, DiffEntry{Key: k, OldValue: cv, NewValue: tv, Status: DiffModified})
		case inCurrent:
			entries = append(entries, DiffEntry{Key: k, OldValue: cv, Status: DiffRemoved})
		case inTarget:
			entries = append(entries, DiffEntry{Key: k, NewValue: tv, Status: DiffAdded})
		}
	}
	return entries
}

// IsActive checks if a profile's settings match the current settings
func IsActive(current map[string]interface{}, p Profile) bool {
	fc := Flatten(current, "")
	fp := Flatten(p.Settings, "")
	if len(fc) != len(fp) {
		return false
	}
	for k, v := range fc {
		if fp[k] != v {
			return false
		}
	}
	return true
}

// MaskSensitive masks sensitive values for display
func MaskSensitive(key, value string) string {
	lower := strings.ToLower(key)
	for _, s := range []string{"token", "key", "secret", "password", "credential"} {
		if strings.Contains(lower, s) {
			if len(value) <= 8 {
				return "****"
			}
			return value[:4] + "****" + value[len(value)-4:]
		}
	}
	return value
}

// GetSummary extracts model and baseURL from settings
func GetSummary(settings map[string]interface{}) (model, baseURL string) {
	if settings == nil {
		return
	}
	if m, ok := settings["model"]; ok {
		model = fmt.Sprintf("%v", m)
	}
	if env, ok := settings["env"].(map[string]interface{}); ok {
		if url, ok := env["ANTHROPIC_BASE_URL"]; ok {
			baseURL = fmt.Sprintf("%v", url)
		}
	}
	return
}

// GetModelMapping returns model mapping information from settings
func GetModelMapping(settings map[string]interface{}) map[string]string {
	mapping := make(map[string]string)
	if settings == nil {
		return mapping
	}
	if env, ok := settings["env"].(map[string]interface{}); ok {
		if v, ok := env["ANTHROPIC_DEFAULT_HAIKU_MODEL"]; ok {
			mapping["haiku"] = fmt.Sprintf("%v", v)
		}
		if v, ok := env["ANTHROPIC_DEFAULT_SONNET_MODEL"]; ok {
			mapping["sonnet"] = fmt.Sprintf("%v", v)
		}
		if v, ok := env["ANTHROPIC_DEFAULT_OPUS_MODEL"]; ok {
			mapping["opus"] = fmt.Sprintf("%v", v)
		}
	}
	return mapping
}
