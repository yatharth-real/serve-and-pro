package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxyRouting(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"upstream":"backend-response","path":"` + r.URL.Path + `"}`))
	}))
	defer backend.Close()

	mgr, err := NewManager(map[string]string{
		"/api": backend.URL,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost:3000/api/users", nil)
	w := httptest.NewRecorder()

	handler, ok := mgr.RouteRequest(req)
	if !ok {
		t.Fatalf("expected route to match /api")
	}

	handler.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(body), "backend-response") {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestParseProxyRules(t *testing.T) {
	rules, err := ParseProxyRules([]string{"/api=http://localhost:8000", "/auth=http://localhost:8081"})
	if err != nil {
		t.Fatalf("ParseProxyRules failed: %v", err)
	}
	if rules["/api"] != "http://localhost:8000" {
		t.Errorf("expected /api -> http://localhost:8000, got %s", rules["/api"])
	}
	if rules["/auth"] != "http://localhost:8081" {
		t.Errorf("expected /auth -> http://localhost:8081, got %s", rules["/auth"])
	}
}
