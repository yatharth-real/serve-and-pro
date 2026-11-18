package benchmark

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBenchmark(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Benchmark HTML</h1>"), 0644)

	rep, err := RunBenchmark(dir, 4, 20)
	if err != nil {
		t.Fatalf("RunBenchmark failed: %v", err)
	}

	if rep.TotalRequests != 20 {
		t.Errorf("expected 20 requests, got %d", rep.TotalRequests)
	}
	if rep.ReqPerSec <= 0 {
		t.Errorf("expected positive req/sec, got %f", rep.ReqPerSec)
	}
	formatted := rep.Format()
	if formatted == "" {
		t.Errorf("expected non-empty format string")
	}
}
