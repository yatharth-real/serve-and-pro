package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
)

// Route maps a URL prefix to an upstream target URL.
type Route struct {
	Prefix  string
	Target  *url.URL
	Handler *httputil.ReverseProxy
}

// Manager manages a collection of reverse proxy routes.
type Manager struct {
	routes []Route
}

// NewManager creates a Manager from a map of prefix -> target URL strings.
func NewManager(ruleMap map[string]string) (*Manager, error) {
	var routes []Route
	for prefix, targetStr := range ruleMap {
		prefix = strings.TrimSpace(prefix)
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}

		u, err := url.Parse(strings.TrimSpace(targetStr))
		if err != nil {
			return nil, fmt.Errorf("invalid proxy target %q: %w", targetStr, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(u)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		}

		routes = append(routes, Route{
			Prefix:  prefix,
			Target:  u,
			Handler: proxy,
		})
	}

	// Sort longer prefixes first for most specific match
	sort.Slice(routes, func(i, j int) bool {
		return len(routes[i].Prefix) > len(routes[j].Prefix)
	})

	return &Manager{routes: routes}, nil
}

// ParseProxyRules parses CLI proxy strings e.g. ["/api=http://localhost:8000", "/auth=http://localhost:8081"].
func ParseProxyRules(rawRules []string) (map[string]string, error) {
	res := make(map[string]string)
	for _, rule := range rawRules {
		parts := strings.SplitN(rule, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid proxy rule %q (expected format: /prefix=http://target)", rule)
		}
		prefix := strings.TrimSpace(parts[0])
		target := strings.TrimSpace(parts[1])
		res[prefix] = target
	}
	return res, nil
}

// RouteRequest checks if a request matches any proxy route. Returns the proxy handler if matched.
func (m *Manager) RouteRequest(r *http.Request) (http.Handler, bool) {
	if m == nil || len(m.routes) == 0 {
		return nil, false
	}

	path := r.URL.Path
	for _, route := range m.routes {
		if strings.HasPrefix(path, route.Prefix) {
			return route.Handler, true
		}
	}
	return nil, false
}

// Routes returns the configured routes list.
func (m *Manager) Routes() []Route {
	if m == nil {
		return nil
	}
	return m.routes
}
