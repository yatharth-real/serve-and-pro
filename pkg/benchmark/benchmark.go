package benchmark

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"serve/pkg/server"
)

// Report contains measured performance metrics.
type Report struct {
	TargetDirectory string
	StartupDuration time.Duration
	FirstResponse   time.Duration
	TotalRequests   int
	Concurrency     int
	Duration        time.Duration
	ReqPerSec       float64
	MemoryAllocMB   float64
	MemorySysMB     float64
}

// RunBenchmark measures local serving performance for a target directory.
func RunBenchmark(targetDir string, concurrency int, totalRequests int) (*Report, error) {
	if concurrency <= 0 {
		concurrency = 10
	}
	if totalRequests <= 0 {
		totalRequests = 500
	}

	// 1. Measure server startup
	startInit := time.Now()
	srv, err := server.NewServer(server.Config{
		Target: targetDir,
		Silent: true,
	})
	if err != nil {
		return nil, err
	}
	startupDur := time.Since(startInit)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 2. Measure first response time (TTFB)
	firstStart := time.Now()
	resp1, err := http.Get(ts.URL + "/")
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(io.Discard, resp1.Body)
	resp1.Body.Close()
	firstRespDur := time.Since(firstStart)

	// 3. Measure concurrent load throughput
	var wg sync.WaitGroup
	reqChan := make(chan struct{}, totalRequests)
	for i := 0; i < totalRequests; i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)

	var successCount int64
	benchStart := time.Now()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: concurrency * 2,
		},
	}

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range reqChan {
				r, err := client.Get(ts.URL + "/")
				if err == nil {
					_, _ = io.Copy(io.Discard, r.Body)
					r.Body.Close()
					if r.StatusCode < 500 {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}
		}()
	}

	wg.Wait()
	benchDur := time.Since(benchStart)

	// 4. Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	reqPerSec := float64(successCount) / benchDur.Seconds()

	return &Report{
		TargetDirectory: targetDir,
		StartupDuration: startupDur,
		FirstResponse:   firstRespDur,
		TotalRequests:   int(successCount),
		Concurrency:     concurrency,
		Duration:        benchDur,
		ReqPerSec:       reqPerSec,
		MemoryAllocMB:   float64(m.Alloc) / 1024 / 1024,
		MemorySysMB:     float64(m.Sys) / 1024 / 1024,
	}, nil
}

// Format formats the benchmark report for CLI printing.
func (r *Report) Format() string {
	var s string
	s += fmt.Sprintf("Serve Benchmark Results\n")
	s += fmt.Sprintf("────────────────────────────────────────\n")
	s += fmt.Sprintf("Target:            %s\n", r.TargetDirectory)
	s += fmt.Sprintf("Startup Time:      %v\n", r.StartupDuration)
	s += fmt.Sprintf("First Response:    %v\n", r.FirstResponse)
	s += fmt.Sprintf("Concurrency:       %d workers\n", r.Concurrency)
	s += fmt.Sprintf("Requests Completed:%d\n", r.TotalRequests)
	s += fmt.Sprintf("Bench Duration:    %v\n", r.Duration)
	s += fmt.Sprintf("Throughput:        %.1f req/sec\n", r.ReqPerSec)
	s += fmt.Sprintf("Memory Allocated:  %.2f MB\n", r.MemoryAllocMB)
	s += fmt.Sprintf("System Memory:     %.2f MB\n", r.MemorySysMB)
	return s
}
