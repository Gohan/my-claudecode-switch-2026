package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Profile struct {
	Name     string
	Settings map[string]interface{}
}

type DiffStatus int

const (
	DiffUnchanged DiffStatus = iota
	DiffModified
	DiffAdded
	DiffRemoved
)

type DiffEntry struct {
	Key      string
	OldValue string
	NewValue string
	Status   DiffStatus
}

func SettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func ProfilesDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude-switch", "profiles")
}

func LoadCurrent() (map[string]interface{}, error) {
	return loadJSON(SettingsPath())
}

func ApplyProfile(p Profile) error {
	return saveJSON(SettingsPath(), p.Settings)
}

func List() ([]Profile, error) {
	dir := ProfilesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var profiles []Profile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		settings, err := loadJSON(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		profiles = append(profiles, Profile{Name: name, Settings: settings})
	}
	return profiles, nil
}

func Save(name string, settings map[string]interface{}) error {
	dir := ProfilesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return saveJSON(filepath.Join(dir, name+".json"), settings)
}

func Delete(name string) error {
	return os.Remove(filepath.Join(ProfilesDir(), name+".json"))
}

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

func GetSummary(settings map[string]interface{}) (model, baseURL string) {
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

func loadJSON(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func saveJSON(path string, data map[string]interface{}) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(bytes, '\n'), 0644)
}
