package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrProjectNotFound   = errors.New("project not found")
	ErrDuplicateName     = errors.New("a project with this name already exists")
	ErrDuplicatePath     = errors.New("this directory is already registered under another name")
	ErrPathNotFound      = errors.New("directory does not exist on disk")
	ErrNoActiveProject   = errors.New("no active project is currently set")
	ErrInvalidName       = errors.New("invalid project name: must not be empty or contain path separators")
)

// Project represents a registered local project.
type Project struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Group        string    `json:"group,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// Exists checks if the project directory still exists on the filesystem.
func (p *Project) Exists() bool {
	info, err := os.Stat(p.Path)
	return err == nil && info.IsDir()
}

// StoreData represents the persisted registry JSON structure.
type StoreData struct {
	ActiveProject string              `json:"active_project,omitempty"`
	Projects      map[string]*Project `json:"projects"`
}

// Registry handles the persistence and manipulation of project records.
type Registry struct {
	filePath string
	mu       sync.RWMutex
	data     StoreData
}

// DefaultStoragePath returns the platform-standard config path for projects.json.
func DefaultStoragePath() string {
	if env := os.Getenv("SERVE_CONFIG_PATH"); env != "" {
		return env
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "serve", "projects.json")
}

// Open loads or initializes the Registry at the given path (or default path if empty).
func Open(customPath string) (*Registry, error) {
	path := customPath
	if path == "" {
		path = DefaultStoragePath()
	}

	reg := &Registry{
		filePath: path,
		data: StoreData{
			Projects: make(map[string]*Project),
		},
	}

	if err := reg.load(); err != nil {
		return nil, err
	}
	return reg, nil
}

func (r *Registry) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			r.data = StoreData{
				Projects: make(map[string]*Project),
			}
			return nil
		}
		return fmt.Errorf("failed to read registry file: %w", err)
	}

	var loaded StoreData
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("corrupt registry file %s: %w", r.filePath, err)
	}
	if loaded.Projects == nil {
		loaded.Projects = make(map[string]*Project)
	}
	r.data = loaded
	return nil
}

func (r *Registry) save() error {
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	bytesData, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize registry: %w", err)
	}

	tmpFile := r.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, bytesData, 0644); err != nil {
		return fmt.Errorf("failed to write temporary registry file: %w", err)
	}

	// Rename temp file to target for atomic update
	if err := os.Rename(tmpFile, r.filePath); err != nil {
		// On Windows, rename might fail if destination exists, so try remove then rename
		_ = os.Remove(r.filePath)
		if err := os.Rename(tmpFile, r.filePath); err != nil {
			return fmt.Errorf("failed to update registry file: %w", err)
		}
	}

	return nil
}

// ValidateName ensures the alias name has no illegal characters.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}
	if strings.ContainsAny(name, "/\\:*?\"<>| \t\n\r") {
		return fmt.Errorf("invalid project name %q: must not contain slashes, spaces, or wildcards", name)
	}
	return nil
}

// Add registers a new project. If name is empty, it derives from path's base name.
func (r *Registry) Add(dirPath, customName string) (*Project, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrPathNotFound, absPath)
	}

	name := strings.TrimSpace(customName)
	if name == "" {
		name = filepath.Base(absPath)
	}

	if err := ValidateName(name); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check duplicates
	if _, exists := r.data.Projects[name]; exists {
		return nil, fmt.Errorf("%w: %q", ErrDuplicateName, name)
	}

	cleanTarget := filepath.Clean(absPath)
	for _, p := range r.data.Projects {
		if filepath.Clean(p.Path) == cleanTarget {
			return nil, fmt.Errorf("%w (%q is already registered as %q)", ErrDuplicatePath, cleanTarget, p.Name)
		}
	}

	now := time.Now()
	proj := &Project{
		Name:         name,
		Path:         cleanTarget,
		RegisteredAt: now,
		LastAccessed: now,
		Tags:         []string{},
	}

	r.data.Projects[name] = proj
	if r.data.ActiveProject == "" {
		r.data.ActiveProject = name
	}

	if err := r.save(); err != nil {
		return nil, err
	}

	return proj, nil
}

// Get finds a project by name and updates its last accessed timestamp.
func (r *Registry) Get(name string) (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	proj, exists := r.data.Projects[name]
	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrProjectNotFound, name)
	}

	proj.LastAccessed = time.Now()
	_ = r.save()
	return proj, nil
}

// Peek finds a project by name without updating last accessed time.
func (r *Registry) Peek(name string) (*Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	proj, exists := r.data.Projects[name]
	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrProjectNotFound, name)
	}
	return proj, nil
}

// Remove deletes a project from the registry. Does NOT touch the disk.
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data.Projects[name]; !exists {
		return fmt.Errorf("%w: %q", ErrProjectNotFound, name)
	}

	delete(r.data.Projects, name)
	if r.data.ActiveProject == name {
		r.data.ActiveProject = ""
		// Pick another project as active if available
		for k := range r.data.Projects {
			r.data.ActiveProject = k
			break
		}
	}

	return r.save()
}

// Rename changes a project alias.
func (r *Registry) Rename(oldName, newName string) error {
	if err := ValidateName(newName); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	proj, exists := r.data.Projects[oldName]
	if !exists {
		return fmt.Errorf("%w: %q", ErrProjectNotFound, oldName)
	}

	if _, exists := r.data.Projects[newName]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateName, newName)
	}

	delete(r.data.Projects, oldName)
	proj.Name = newName
	r.data.Projects[newName] = proj

	if r.data.ActiveProject == oldName {
		r.data.ActiveProject = newName
	}

	return r.save()
}

// List returns all projects sorted alphabetically by name.
func (r *Registry) List() []*Project {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*Project, 0, len(r.data.Projects))
	for _, p := range r.data.Projects {
		res = append(res, p)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})

	return res
}

// SetActive sets the current active project.
func (r *Registry) SetActive(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	proj, exists := r.data.Projects[name]
	if !exists {
		return fmt.Errorf("%w: %q", ErrProjectNotFound, name)
	}

	r.data.ActiveProject = name
	proj.LastAccessed = time.Now()
	return r.save()
}

// GetActive returns the active project.
func (r *Registry) GetActive() (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.data.ActiveProject == "" {
		return nil, ErrNoActiveProject
	}

	proj, exists := r.data.Projects[r.data.ActiveProject]
	if !exists {
		r.data.ActiveProject = ""
		_ = r.save()
		return nil, ErrNoActiveProject
	}

	proj.LastAccessed = time.Now()
	_ = r.save()
	return proj, nil
}

// AddTag adds a tag to a project.
func (r *Registry) AddTag(name, tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return errors.New("tag cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	proj, exists := r.data.Projects[name]
	if !exists {
		return fmt.Errorf("%w: %q", ErrProjectNotFound, name)
	}

	for _, t := range proj.Tags {
		if strings.EqualFold(t, tag) {
			return nil // already has tag
		}
	}

	proj.Tags = append(proj.Tags, tag)
	return r.save()
}

// RemoveTag removes a tag from a project.
func (r *Registry) RemoveTag(name, tag string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	proj, exists := r.data.Projects[name]
	if !exists {
		return fmt.Errorf("%w: %q", ErrProjectNotFound, name)
	}

	var updated []string
	found := false
	for _, t := range proj.Tags {
		if strings.EqualFold(t, tag) {
			found = true
			continue
		}
		updated = append(updated, t)
	}

	if !found {
		return fmt.Errorf("tag %q not found on project %q", tag, name)
	}

	proj.Tags = updated
	return r.save()
}

// SetGroup assigns a group to a project.
func (r *Registry) SetGroup(name, group string) error {
	group = strings.TrimSpace(group)

	r.mu.Lock()
	defer r.mu.Unlock()

	proj, exists := r.data.Projects[name]
	if !exists {
		return fmt.Errorf("%w: %q", ErrProjectNotFound, name)
	}

	proj.Group = group
	return r.save()
}

// Recent returns the N most recently accessed projects.
func (r *Registry) Recent(limit int) []*Project {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*Project, 0, len(r.data.Projects))
	for _, p := range r.data.Projects {
		list = append(list, p)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].LastAccessed.After(list[j].LastAccessed)
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}

	return list
}
