package supervisor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	ErrProcessNotFound = errors.New("process not found")
)

// ProcessRecord holds information about a supervised Serve process.
type ProcessRecord struct {
	ID        string    `json:"id"`
	PID       int       `json:"pid"`
	Command   string    `json:"command"`
	Directory string    `json:"directory"`
	Port      int       `json:"port"`
	Runtime   string    `json:"runtime"`
	Framework string    `json:"framework"`
	StartedAt time.Time `json:"started_at"`
	LogFile   string    `json:"log_file"`
}

type processStore struct {
	Processes map[string]*ProcessRecord `json:"processes"`
}

var (
	storeMu sync.Mutex
)

func storePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "serve", "processes.json")
}

func logDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "serve", "logs")
}

func loadStore() (*processStore, error) {
	p := storePath()
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &processStore{Processes: make(map[string]*ProcessRecord)}, nil
		}
		return nil, err
	}
	var store processStore
	_ = json.Unmarshal(data, &store)
	if store.Processes == nil {
		store.Processes = make(map[string]*ProcessRecord)
	}
	return &store, nil
}

func saveStore(store *processStore) error {
	p := storePath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// IsProcessAlive checks if a process is actively running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), strconv.Itoa(pid))
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// FindAvailablePort tests ports sequentially starting at startPort.
func FindAvailablePort(startPort int, maxAttempts int) (int, error) {
	if startPort <= 0 {
		startPort = 3000
	}
	if maxAttempts <= 0 {
		maxAttempts = 50
	}

	for p := startPort; p < startPort+maxAttempts; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			_ = ln.Close()
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available port found between %d and %d", startPort, startPort+maxAttempts)
}

// KillProcessTree cleanly terminates a process and all its child subprocesses.
func KillProcessTree(pid int) error {
	if pid <= 0 {
		return nil
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
		return cmd.Run()
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

// RegisterProcess saves a process to the store.
func RegisterProcess(rec *ProcessRecord) error {
	storeMu.Lock()
	defer storeMu.Unlock()

	store, err := loadStore()
	if err != nil {
		return err
	}
	store.Processes[rec.ID] = rec
	return saveStore(store)
}

// UnregisterProcess removes a process from the store.
func UnregisterProcess(id string) error {
	storeMu.Lock()
	defer storeMu.Unlock()

	store, err := loadStore()
	if err != nil {
		return err
	}
	delete(store.Processes, id)
	return saveStore(store)
}

// ListProcesses returns active processes and purges dead ones.
func ListProcesses() ([]*ProcessRecord, error) {
	storeMu.Lock()
	defer storeMu.Unlock()

	store, err := loadStore()
	if err != nil {
		return nil, err
	}

	var active []*ProcessRecord
	changed := false

	for id, rec := range store.Processes {
		if IsProcessAlive(rec.PID) {
			active = append(active, rec)
		} else {
			delete(store.Processes, id)
			changed = true
		}
	}

	if changed {
		_ = saveStore(store)
	}

	return active, nil
}

// FindProcess finds by ID or PID string.
func FindProcess(query string) (*ProcessRecord, error) {
	list, err := ListProcesses()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	for _, p := range list {
		if strings.EqualFold(p.ID, queryLower) || strconv.Itoa(p.PID) == query {
			return p, nil
		}
	}
	return nil, ErrProcessNotFound
}

// StopProcess terminates a process by name or PID.
func StopProcess(query string) error {
	proc, err := FindProcess(query)
	if err != nil {
		return err
	}

	_ = KillProcessTree(proc.PID)
	return UnregisterProcess(proc.ID)
}

// WaitForHealth probes port / HTTP connection until server responds or timeout occurs.
func WaitForHealth(port int, host string, timeout time.Duration) bool {
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}

	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	for time.Now().Before(deadline) {
		// Try TCP probe first
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}

		// Try HTTP probe
		url := fmt.Sprintf("http://%s:%d/", host, port)
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return true
		}

		time.Sleep(150 * time.Millisecond)
	}
	return false
}

// SuperviseForeground runs a command in the foreground with full process management, logging, health check, and signal handling.
func SuperviseForeground(id, dir, commandStr string, port int, rt, framework string, extraEnv []string) error {
	_ = os.MkdirAll(logDir(), 0755)
	logFile := filepath.Join(logDir(), fmt.Sprintf("%s.log", id))
	logF, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer logF.Close()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", commandStr)
	} else {
		cmd = exec.Command("sh", "-c", commandStr)
	}

	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)

	// Memory buffer to capture recent output for health failure diagnostics
	var errorBuf bytes.Buffer
	multiOut := io.MultiWriter(os.Stdout, logF, &errorBuf)
	multiErr := io.MultiWriter(os.Stderr, logF, &errorBuf)

	cmd.Stdout = multiOut
	cmd.Stderr = multiErr
	cmd.Stdin = os.Stdin

	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %q: %w", commandStr, err)
	}

	rec := &ProcessRecord{
		ID:        id,
		PID:       cmd.Process.Pid,
		Command:   commandStr,
		Directory: dir,
		Port:      port,
		Runtime:   rt,
		Framework: framework,
		StartedAt: startTime,
		LogFile:   logFile,
	}

	_ = RegisterProcess(rec)
	defer func() {
		_ = UnregisterProcess(id)
	}()

	// Background health probe
	healthyChan := make(chan bool, 1)
	go func() {
		healthyChan <- WaitForHealth(port, "127.0.0.1", 10*time.Second)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Wait()
	}()

	bannerPrinted := false

	for {
		select {
		case isHealthy := <-healthyChan:
			if isHealthy && !bannerPrinted {
				bannerPrinted = true
				duration := time.Since(startTime).Round(10 * time.Millisecond)
				fmt.Println()
				fmt.Println("Serve")
				fmt.Println("────────────────────────")
				fmt.Printf("Project:   %s\n", id)
				fmt.Printf("Framework: %s\n", framework)
				fmt.Printf("Runtime:   %s\n", rt)
				fmt.Println("Status:    RUNNING")
				fmt.Println()
				fmt.Printf("Local:     http://localhost:%d\n", port)
				fmt.Printf("Process:   %d\n", cmd.Process.Pid)
				fmt.Printf("Startup:   %v\n", duration)
				fmt.Println()
				fmt.Println("Press Ctrl+C to stop.")
			}

		case <-sigChan:
			fmt.Printf("\nShutting down %s (PID %d)...\n", id, cmd.Process.Pid)
			_ = KillProcessTree(cmd.Process.Pid)
			return nil

		case err := <-errChan:
			if err != nil {
				if !bannerPrinted {
					fmt.Println()
					fmt.Println("Serve failed to start the project.")
					fmt.Println()
					recentOut := strings.TrimSpace(errorBuf.String())
					if recentOut != "" {
						fmt.Println("Reason:")
						fmt.Println(recentOut)
						fmt.Println()
					}
				}
				return err
			}
			return nil
		}
	}
}

// ReadLogs returns the trailing N lines from a process log file.
func ReadLogs(query string, linesCount int) ([]string, error) {
	proc, err := FindProcess(query)
	if err != nil {
		candidate := filepath.Join(logDir(), fmt.Sprintf("%s.log", query))
		if _, statErr := os.Stat(candidate); statErr == nil {
			proc = &ProcessRecord{LogFile: candidate}
		} else {
			return nil, err
		}
	}

	f, err := os.Open(proc.LogFile)
	if err != nil {
		return nil, fmt.Errorf("could not open log file: %w", err)
	}
	defer f.Close()

	if linesCount <= 0 {
		linesCount = 50
	}

	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if len(allLines) > linesCount {
		allLines = allLines[len(allLines)-linesCount:]
	}
	return allLines, nil
}
