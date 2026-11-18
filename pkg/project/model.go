package project

import "time"

// Project is the unified PRO project intelligence model.
type Project struct {
	Name           string            `json:"name"`
	Path           string            `json:"path"`
	Group          string            `json:"group,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	// Git
	GitBranch      string            `json:"git_branch,omitempty"`
	GitDirty       bool              `json:"git_dirty,omitempty"`
	GitAhead       int               `json:"git_ahead,omitempty"`
	GitBehind      int               `json:"git_behind,omitempty"`
	GitCommit      string            `json:"git_commit,omitempty"`
	// Ecosystem
	Runtime        string            `json:"runtime,omitempty"`
	Language       string            `json:"language,omitempty"`
	Framework      string            `json:"framework,omitempty"`
	PackageManager string            `json:"package_manager,omitempty"`
	BuildSystem    string            `json:"build_system,omitempty"`
	// Commands
	DevCommand     string            `json:"dev_command,omitempty"`
	BuildCommand   string            `json:"build_command,omitempty"`
	PreviewCommand string            `json:"preview_command,omitempty"`
	Scripts        map[string]string `json:"scripts,omitempty"`
	// Runtime state
	Port           int               `json:"port,omitempty"`
	ProcessStatus  string            `json:"process_status,omitempty"`
	// Timestamps
	DiscoveredAt   time.Time         `json:"discovered_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func (p *Project) IsRunning() bool { return p.ProcessStatus == "running" }
