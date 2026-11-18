package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Workspace groups related projects together.
type Workspace struct {
	Name     string    `json:"name"`
	Projects []string  `json:"projects"` // project aliases
	Active   bool      `json:"active"`
	Created  time.Time `json:"created"`
}

// Store manages workspace persistence at ~/.pro/workspaces.json
type Store struct {
	filePath   string
	Workspaces map[string]*Workspace `json:"workspaces"`
	ActiveName string                `json:"active,omitempty"`
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".pro", "workspaces.json")
}

func Open(path string) (*Store, error) {
	if path == "" {
		path = DefaultPath()
	}
	s := &Store{filePath: path, Workspaces: make(map[string]*Workspace)}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	_ = json.Unmarshal(data, s)
	if s.Workspaces == nil {
		s.Workspaces = make(map[string]*Workspace)
	}
	return s, nil
}

func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	_ = os.Remove(s.filePath)
	return os.Rename(tmp, s.filePath)
}

func (s *Store) Create(name string) (*Workspace, error) {
	if _, exists := s.Workspaces[name]; exists {
		return s.Workspaces[name], nil
	}
	w := &Workspace{Name: name, Created: time.Now()}
	s.Workspaces[name] = w
	return w, s.Save()
}

func (s *Store) Add(wsName, projectAlias string) error {
	w, ok := s.Workspaces[wsName]
	if !ok {
		return os.ErrNotExist
	}
	for _, p := range w.Projects {
		if p == projectAlias {
			return nil // already member
		}
	}
	w.Projects = append(w.Projects, projectAlias)
	return s.Save()
}

func (s *Store) Remove(wsName, projectAlias string) error {
	w, ok := s.Workspaces[wsName]
	if !ok {
		return os.ErrNotExist
	}
	var updated []string
	for _, p := range w.Projects {
		if p != projectAlias {
			updated = append(updated, p)
		}
	}
	w.Projects = updated
	return s.Save()
}

func (s *Store) Use(name string) error {
	if _, ok := s.Workspaces[name]; !ok {
		return os.ErrNotExist
	}
	s.ActiveName = name
	return s.Save()
}

func (s *Store) List() []*Workspace {
	out := make([]*Workspace, 0, len(s.Workspaces))
	for _, w := range s.Workspaces {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Store) Active() *Workspace {
	if s.ActiveName == "" {
		return nil
	}
	return s.Workspaces[s.ActiveName]
}
