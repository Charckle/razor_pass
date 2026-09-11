package recents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	appName  = "rpass"
	fileName = "recents.json"
	maxKeep  = 15
)

// Store is the on-disk list of recently opened databases, most recent first.
type Store struct {
	Paths []string `json:"paths"`
	path  string
}

// DefaultPath is ~/.config/rpass/recents.json (or the OS equivalent).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName, fileName), nil
}

// Load reads a recents file. A missing file is an empty store, not an error.
func Load(path string) (*Store, error) {
	s := &Store{path: path, Paths: []string{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("recents: %w", err)
	}
	if s.Paths == nil {
		s.Paths = []string{}
	}
	return s, nil
}

// Save writes the store with mode 0600.
func (s *Store) Save() error {
	if s.path == "" {
		return fmt.Errorf("recents: no path")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.path, data, 0o600)
}

// PruneMissing drops paths that no longer exist and de-duplicates.
func (s *Store) PruneMissing() {
	seen := make(map[string]struct{}, len(s.Paths))
	out := make([]string, 0, len(s.Paths))
	for _, p := range s.Paths {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			out = append(out, p)
		}
	}
	s.Paths = out
}

// Touch moves abs to the front (most recent) and caps the list.
func (s *Store) Touch(abs string) {
	out := make([]string, 0, len(s.Paths)+1)
	out = append(out, abs)
	for _, p := range s.Paths {
		if p != abs {
			out = append(out, p)
		}
	}
	if len(out) > maxKeep {
		out = out[:maxKeep]
	}
	s.Paths = out
}

// Resolve turns a user-supplied (possibly relative) path into a cleaned
// absolute path and checks that it is an existing file.
func Resolve(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("%s: %w", p, err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("%s: is a directory", p)
	}
	return abs, nil
}
