package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LiveReloader manages file change notifications and SSE connections.
type LiveReloader struct {
	rootDir   string
	clients   map[chan string]bool
	mu        sync.Mutex
	fileTimes map[string]time.Time
	stopChan  chan struct{}
}

// NewLiveReloader creates a new LiveReloader instance.
func NewLiveReloader(rootDir string) *LiveReloader {
	lr := &LiveReloader{
		rootDir:   rootDir,
		clients:   make(map[chan string]bool),
		fileTimes: make(map[string]time.Time),
		stopChan:  make(chan struct{}),
	}
	lr.snapshot()
	go lr.watchLoop()
	return lr
}

// Stop stops the watcher.
func (lr *LiveReloader) Stop() {
	close(lr.stopChan)
}

func (lr *LiveReloader) snapshot() map[string]time.Time {
	times := make(map[string]time.Time)
	_ = filepath.WalkDir(lr.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		if info, err := d.Info(); err == nil {
			times[path] = info.ModTime()
		}
		return nil
	})
	return times
}

func (lr *LiveReloader) watchLoop() {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	lr.mu.Lock()
	lr.fileTimes = lr.snapshot()
	lr.mu.Unlock()

	for {
		select {
		case <-lr.stopChan:
			return
		case <-ticker.C:
			current := lr.snapshot()
			lr.mu.Lock()
			changed := false
			if len(current) != len(lr.fileTimes) {
				changed = true
			} else {
				for p, t := range current {
					prev, ok := lr.fileTimes[p]
					if !ok || !prev.Equal(t) {
						changed = true
						break
					}
				}
			}
			if changed {
				lr.fileTimes = current
				lr.broadcast("reload")
			}
			lr.mu.Unlock()
		}
	}
}

func (lr *LiveReloader) broadcast(msg string) {
	for ch := range lr.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// ServeSSE handles the SSE endpoint for live reload.
func (lr *LiveReloader) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 10)
	lr.mu.Lock()
	lr.clients[ch] = true
	lr.mu.Unlock()

	defer func() {
		lr.mu.Lock()
		delete(lr.clients, ch)
		lr.mu.Unlock()
	}()

	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg := <-ch:
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg, msg)
			flusher.Flush()
		}
	}
}

const LiveReloadScript = `<script>
(() => {
  const es = new EventSource('/__serve/live-reload');
  es.onmessage = (e) => {
    if (e.data === 'reload') {
      window.location.reload();
    }
  };
  es.onerror = () => {
    setTimeout(() => window.location.reload(), 1500);
  };
})();
</script>`
