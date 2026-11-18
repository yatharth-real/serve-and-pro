package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var (
	ErrNoActionsConfigured = errors.New("no actions configured in project (.pro/actions.json)")
	ErrActionNotFound      = errors.New("action not found")
)

// Config represents the `.pro/actions.json` file content.
type Config struct {
	Actions map[string]string `json:"actions"`
}

func configPath(projectDir string) string {
	return filepath.Join(projectDir, ".pro", "actions.json")
}

// LoadConfig loads the actions configuration for a given project directory.
func LoadConfig(projectDir string) (*Config, error) {
	file := configPath(projectDir)
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			// Also check .pro/config.json fallback
			fallback := filepath.Join(projectDir, ".pro", "config.json")
			if fbData, fbErr := os.ReadFile(fallback); fbErr == nil {
				data = fbData
			} else {
				return &Config{Actions: make(map[string]string)}, nil
			}
		} else {
			return nil, err
		}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("corrupt config file %s: %w", file, err)
	}
	if cfg.Actions == nil {
		cfg.Actions = make(map[string]string)
	}
	return &cfg, nil
}

// SaveConfig saves the actions configuration for a given project directory.
func SaveConfig(projectDir string, cfg *Config) error {
	dir := filepath.Join(projectDir, ".pro")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath(projectDir), data, 0644)
}

// SetAction adds or updates an action.
func SetAction(projectDir, actionName, command string) error {
	actionName = strings.TrimSpace(actionName)
	command = strings.TrimSpace(command)
	if actionName == "" || command == "" {
		return errors.New("action name and command cannot be empty")
	}

	cfg, err := LoadConfig(projectDir)
	if err != nil {
		return err
	}

	cfg.Actions[actionName] = command
	return SaveConfig(projectDir, cfg)
}

// ListActions returns a list of sorted action names.
func ListActions(projectDir string) ([]string, map[string]string, error) {
	cfg, err := LoadConfig(projectDir)
	if err != nil {
		return nil, nil, err
	}

	keys := make([]string, 0, len(cfg.Actions))
	for k := range cfg.Actions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, cfg.Actions, nil
}

// Run executes a named action inside projectDir.
func Run(projectDir, actionName string, extraArgs []string) error {
	cfg, err := LoadConfig(projectDir)
	if err != nil {
		return err
	}

	if len(cfg.Actions) == 0 {
		return ErrNoActionsConfigured
	}

	// If no action specified, default to "dev" or "start" if present
	if actionName == "" {
		if _, ok := cfg.Actions["dev"]; ok {
			actionName = "dev"
		} else if _, ok := cfg.Actions["start"]; ok {
			actionName = "start"
		} else {
			var names []string
			for k := range cfg.Actions {
				names = append(names, k)
			}
			return fmt.Errorf("no action specified. Available actions: %s", strings.Join(names, ", "))
		}
	}

	cmdStr, exists := cfg.Actions[actionName]
	if !exists {
		return fmt.Errorf("%w: %q", ErrActionNotFound, actionName)
	}

	if len(extraArgs) > 0 {
		cmdStr += " " + strings.Join(extraArgs, " ")
	}

	fmt.Printf("Running action [%s]: %s\n\n", actionName, cmdStr)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}

	cmd.Dir = projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
