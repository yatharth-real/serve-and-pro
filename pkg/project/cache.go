package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const cacheTTL = 5 * time.Minute

func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pro", "cache.json"), nil
}

// LoadCache reads the on-disk project cache. Missing/corrupt cache returns empty map.
func LoadCache() (map[string]*Project, error) {
	out := make(map[string]*Project)
	cp, err := cachePath()
	if err != nil {
		return out, nil
	}
	data, err := os.ReadFile(cp)
	if err != nil {
		return out, nil // cache miss is not an error
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return make(map[string]*Project), nil // corrupt → reset
	}
	return out, nil
}

// SaveCache persists the project cache atomically.
func SaveCache(m map[string]*Project) error {
	cp, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cp), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	tmp := cp + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	_ = os.Remove(cp)
	return os.Rename(tmp, cp)
}

func isFresh(t time.Time) bool { return time.Since(t) < cacheTTL }
