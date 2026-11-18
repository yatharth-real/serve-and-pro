package supervisor

import (
	"os"
	"testing"
	"time"
)

func TestSupervisorRegistrationAndPort(t *testing.T) {
	// Find available port
	port, err := FindAvailablePort(35000, 20)
	if err != nil {
		t.Fatalf("FindAvailablePort failed: %v", err)
	}
	if port < 35000 {
		t.Errorf("expected port >= 35000, got %d", port)
	}

	rec := &ProcessRecord{
		ID:        "test-proc",
		PID:       os.Getpid(), // current running test process is guaranteed alive
		Command:   "test command",
		Directory: ".",
		Port:      port,
		Runtime:   "Go",
		Framework: "Test",
		StartedAt: time.Now(),
	}

	if err := RegisterProcess(rec); err != nil {
		t.Fatalf("RegisterProcess failed: %v", err)
	}

	found, err := FindProcess("test-proc")
	if err != nil {
		t.Fatalf("FindProcess failed: %v", err)
	}
	if found.PID != rec.PID || found.Port != rec.Port {
		t.Errorf("found record mismatch: %+v", found)
	}

	if err := UnregisterProcess("test-proc"); err != nil {
		t.Fatalf("UnregisterProcess failed: %v", err)
	}
}
