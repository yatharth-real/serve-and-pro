package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorDiagnostics(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Static</h1>"), 0644)

	output := RunDiagnostics(dir)
	if !strings.Contains(output, "Serve Doctor") {
		t.Errorf("expected header 'Serve Doctor' in output:\n%s", output)
	}
	if !strings.Contains(output, "READY") {
		t.Errorf("expected READY status in output:\n%s", output)
	}
}
