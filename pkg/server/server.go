package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"serve/pkg/proxy"
	"serve/pkg/supervisor"
	"serve/pkg/util"
)

func init() {
	_ = mime.AddExtensionType(".js", "application/javascript")
	_ = mime.AddExtensionType(".mjs", "application/javascript")
	_ = mime.AddExtensionType(".ts", "text/plain")
	_ = mime.AddExtensionType(".tsx", "text/plain")
	_ = mime.AddExtensionType(".jsx", "text/plain")
	_ = mime.AddExtensionType(".wasm", "application/wasm")
	_ = mime.AddExtensionType(".json", "application/json")
	_ = mime.AddExtensionType(".map", "application/json")
	_ = mime.AddExtensionType(".svg", "image/svg+xml")
	_ = mime.AddExtensionType(".webp", "image/webp")
	_ = mime.AddExtensionType(".avif", "image/avif")
	_ = mime.AddExtensionType(".ico", "image/x-icon")
	_ = mime.AddExtensionType(".woff2", "font/woff2")
	_ = mime.AddExtensionType(".woff", "font/woff")
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

// Config holds the server runtime options.
type Config struct {
	Target       string
	Host         string
	Port         int
	AutoPort     bool
	LAN          bool
	Open         bool
	HTTPS        bool
	Watch        bool
	Silent       bool
	ProxyManager *proxy.Manager
}

// Server encapsulates the HTTP file server.
type Server struct {
	config     Config
	absRoot    string
	singleFile bool
	reloader   *LiveReloader
	httpServer *http.Server
}

// NewServer initializes a new Server.
func NewServer(cfg Config) (*Server, error) {
	if cfg.Target == "" {
		cfg.Target = "."
	}
	abs, err := filepath.Abs(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("target does not exist: %w", err)
	}

	if cfg.Port <= 0 {
		cfg.Port = 3000
	}
	if cfg.Host == "" {
		if cfg.LAN {
			cfg.Host = "0.0.0.0"
		} else {
			cfg.Host = "127.0.0.1"
		}
	}

	s := &Server{
		config:     cfg,
		absRoot:    abs,
		singleFile: !info.IsDir(),
	}

	if cfg.Watch {
		watchDir := abs
		if s.singleFile {
			watchDir = filepath.Dir(abs)
		}
		s.reloader = NewLiveReloader(watchDir)
	}

	return s, nil
}

// Handler returns the HTTP handler with proxy, watch, security, and static serving.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Reverse proxy routing
		if s.config.ProxyManager != nil {
			if proxyHandler, ok := s.config.ProxyManager.RouteRequest(r); ok {
				proxyHandler.ServeHTTP(w, r)
				return
			}
		}

		// Only allow GET, HEAD, OPTIONS
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			w.Header().Set("Allow", "GET, HEAD, OPTIONS")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", "GET, HEAD, OPTIONS")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Live reload SSE endpoint
		if s.config.Watch && r.URL.Path == "/__serve/live-reload" {
			s.reloader.ServeSSE(w, r)
			return
		}

		s.serveFile(w, r)
	})
}

// isProtectedFile checks for secrets, private keys, environment files, and sensitive data.
func isProtectedFile(fileName string) bool {
	base := strings.ToLower(filepath.Base(fileName))
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	if strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") ||
		strings.HasSuffix(base, ".pfx") || strings.HasPrefix(base, "id_rsa") ||
		strings.HasSuffix(base, ".kdbx") {
		return true
	}
	return false
}

func isSafePath(rawPath string) bool {
	if strings.Contains(rawPath, "\x00") {
		return false
	}
	if strings.Contains(strings.ToUpper(rawPath), "::$DATA") {
		return false
	}
	cleaned := filepath.Clean(rawPath)
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, filepath.FromSlash("/../")) {
		return false
	}
	return true
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	// Single file serving
	if s.singleFile {
		if r.URL.Path == "/" || r.URL.Path == "/"+filepath.Base(s.absRoot) {
			s.respondFile(w, r, s.absRoot)
			return
		}
		parentDir := filepath.Dir(s.absRoot)
		reqPath, err := url.PathUnescape(r.URL.Path)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		target := filepath.Join(parentDir, filepath.Clean(reqPath))
		if !strings.HasPrefix(strings.ToLower(target), strings.ToLower(parentDir)) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if isProtectedFile(target) {
			http.Error(w, "Forbidden: Protected File", http.StatusForbidden)
			return
		}
		if _, err := os.Stat(target); err == nil {
			s.respondFile(w, r, target)
			return
		}
		http.NotFound(w, r)
		return
	}

	// Directory serving
	reqPath, err := url.PathUnescape(r.URL.Path)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if !isSafePath(reqPath) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	cleanRel := filepath.Clean(strings.TrimPrefix(filepath.FromSlash(reqPath), string(filepath.Separator)))
	targetPath := filepath.Join(s.absRoot, cleanRel)

	rootLower := strings.ToLower(s.absRoot)
	targetLower := strings.ToLower(targetPath)
	if !strings.HasPrefix(targetLower, rootLower) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Secret protection check
	if isProtectedFile(targetPath) {
		http.Error(w, "Forbidden: Protected File", http.StatusForbidden)
		return
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		if os.IsPermission(err) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if info.IsDir() {
		if !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
			return
		}

		indexHTML := filepath.Join(targetPath, "index.html")
		if idxInfo, err := os.Stat(indexHTML); err == nil && !idxInfo.IsDir() {
			s.respondFile(w, r, indexHTML)
			return
		}
		indexHTM := filepath.Join(targetPath, "index.htm")
		if idxInfo, err := os.Stat(indexHTM); err == nil && !idxInfo.IsDir() {
			s.respondFile(w, r, indexHTM)
			return
		}

		entries, err := os.ReadDir(targetPath)
		if err != nil {
			http.Error(w, "Unable to read directory", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = RenderDirectoryListing(w, r.URL.Path, targetPath, entries)
		return
	}

	s.respondFile(w, r, targetPath)
}

func (s *Server) respondFile(w http.ResponseWriter, r *http.Request, filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ctype := mime.TypeByExtension(filepath.Ext(filePath))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)

	// ETag generation
	etag := fmt.Sprintf(`W/"%x-%x"`, info.Size(), info.ModTime().UnixNano())
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if s.config.Watch && strings.Contains(ctype, "text/html") {
		content, err := io.ReadAll(f)
		if err == nil {
			bodyStr := string(content)
			if idx := strings.LastIndex(bodyStr, "</body>"); idx != -1 {
				bodyStr = bodyStr[:idx] + LiveReloadScript + bodyStr[idx:]
			} else if idx := strings.LastIndex(bodyStr, "</html>"); idx != -1 {
				bodyStr = bodyStr[:idx] + LiveReloadScript + bodyStr[idx:]
			} else {
				bodyStr += LiveReloadScript
			}
			injected := []byte(bodyStr)
			http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(injected))
			return
		}
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

// Start runs the server and blocks until SIGINT/SIGTERM or context cancellation.
func (s *Server) Start() error {
	// Auto port resolution if requested or default
	if s.config.AutoPort && !util.CheckPortAvailable(s.config.Host, s.config.Port) {
		nextPort, err := supervisor.FindAvailablePort(s.config.Port+1, 50)
		if err == nil {
			if !s.config.Silent {
				fmt.Printf("Port %d is already in use. Trying %d...\n", s.config.Port, nextPort)
			}
			s.config.Port = nextPort
		}
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	protocol := "http"
	if s.config.HTTPS {
		protocol = "https"
	}

	var tlsConf *tls.Config
	if s.config.HTTPS {
		cert, err := GenerateSelfSignedCert()
		if err != nil {
			return fmt.Errorf("failed to generate TLS certificate: %w", err)
		}
		tlsConf = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	s.httpServer = &http.Server{
		Addr:      addr,
		Handler:   s.Handler(),
		TLSConfig: tlsConf,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	localURL := fmt.Sprintf("%s://localhost:%d", protocol, s.config.Port)
	if s.config.Host != "127.0.0.1" && s.config.Host != "0.0.0.0" && s.config.Host != "localhost" {
		localURL = fmt.Sprintf("%s://%s:%d", protocol, s.config.Host, s.config.Port)
	}

	if !s.config.Silent {
		label := "Directory:"
		if s.singleFile {
			label = "File:     "
		}
		fmt.Println("Serve")
		fmt.Println("────────────────────────")
		fmt.Printf("%s %s\n", label, s.absRoot)
		fmt.Printf("Local:     %s\n", localURL)

		if s.config.LAN || s.config.Host == "0.0.0.0" {
			lanIP := util.GetLocalIP()
			fmt.Printf("Network:   %s://%s:%d\n", protocol, lanIP, s.config.Port)
			fmt.Println("Warning:   Server is exposed to the local area network.")
		}

		if s.config.HTTPS {
			fmt.Println("Mode:      HTTPS (Self-signed)")
		}
		if s.config.Watch {
			fmt.Println("Watch:     Enabled (Live reload)")
		}
		if s.config.ProxyManager != nil {
			routes := s.config.ProxyManager.Routes()
			for _, rt := range routes {
				fmt.Printf("Proxy:     %s -> %s\n", rt.Prefix, rt.Target.String())
			}
		}
		fmt.Println()
		fmt.Println("Press Ctrl+C to stop.")
	}

	if s.config.Open {
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = util.OpenURL(localURL)
		}()
	}

	errChan := make(chan error, 1)
	go func() {
		if s.config.HTTPS {
			errChan <- s.httpServer.ServeTLS(listener, "", "")
		} else {
			errChan <- s.httpServer.Serve(listener)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-sigChan:
		if !s.config.Silent {
			fmt.Println("\nStopping server...")
		}
		if s.reloader != nil {
			s.reloader.Stop()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
}

// Stop programmatically stops the running server.
func (s *Server) Stop() error {
	if s.reloader != nil {
		s.reloader.Stop()
	}
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
