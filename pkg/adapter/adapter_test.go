package adapter

import (
	"strings"
	"testing"

	"serve/pkg/detector"
)

func TestBuildFrameworkDevCommand(t *testing.T) {
	tests := []struct {
		name       string
		det        *detector.Result
		port       int
		host       string
		wantSubstr string
		wantEnv    string
	}{
		{
			name: "Vite with pnpm",
			det: &detector.Result{
				Framework:      "vite",
				PackageManager: "pnpm",
				DevCommand:     "pnpm run dev",
			},
			port:       5173,
			host:       "127.0.0.1",
			wantSubstr: "--port 5173",
		},
		{
			name: "Next.js with npm",
			det: &detector.Result{
				Framework:      "next",
				PackageManager: "npm",
				DevCommand:     "npm run dev",
			},
			port:       4000,
			host:       "0.0.0.0",
			wantSubstr: "-p 4000 -H 0.0.0.0",
		},
		{
			name: "Django with python",
			det: &detector.Result{
				Framework:      "django",
				PackageManager: "pip",
				DevCommand:     "python manage.py runserver",
			},
			port:       8000,
			host:       "127.0.0.1",
			wantSubstr: "127.0.0.1:8000",
		},
		{
			name: "FastAPI with uvicorn",
			det: &detector.Result{
				Framework:      "fastapi",
				PackageManager: "pip",
				DevCommand:     "uvicorn main:app --reload",
			},
			port:       8080,
			host:       "0.0.0.0",
			wantSubstr: "--port 8080 --host 0.0.0.0",
		},
		{
			name: "Go Gin server",
			det: &detector.Result{
				Framework:      "gin",
				PackageManager: "go",
				DevCommand:     "go run .",
			},
			port:       9000,
			host:       "127.0.0.1",
			wantSubstr: "go run .",
			wantEnv:    "PORT=9000",
		},
		{
			name: "ASP.NET Core",
			det: &detector.Result{
				Framework:      "aspnet",
				PackageManager: "dotnet",
				DevCommand:     "dotnet run",
			},
			port:       5000,
			host:       "localhost",
			wantSubstr: "dotnet run",
			wantEnv:    "ASPNETCORE_URLS=http://localhost:5000",
		},
		{
			name: "Laravel artisan",
			det: &detector.Result{
				Framework:      "laravel",
				PackageManager: "composer",
				DevCommand:     "php artisan serve",
			},
			port:       8000,
			host:       "127.0.0.1",
			wantSubstr: "--port=8000",
		},
		{
			name: "Ruby on Rails",
			det: &detector.Result{
				Framework:      "rails",
				PackageManager: "bundle",
				DevCommand:     "bin/rails server",
			},
			port:       3000,
			host:       "127.0.0.1",
			wantSubstr: "-p 3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, envs := BuildFrameworkDevCommand(tt.det, tt.port, tt.host)
			if !strings.Contains(cmd, tt.wantSubstr) {
				t.Errorf("BuildFrameworkDevCommand() cmd = %q, want substring %q", cmd, tt.wantSubstr)
			}
			if tt.wantEnv != "" {
				found := false
				for _, e := range envs {
					if e == tt.wantEnv {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("BuildFrameworkDevCommand() envs = %v, want env %q", envs, tt.wantEnv)
				}
			}
		})
	}
}

func TestCheckRuntime(t *testing.T) {
	// "node" or "python" or "go" should be checked gracefully
	res := CheckRuntime("go")
	if !res.Installed {
		t.Logf("Go runtime is unexpectedly not installed in test environment")
	} else if res.Version == "" {
		t.Errorf("CheckRuntime('go') version should not be empty")
	}

	unknown := CheckRuntime("some_non_existent_runtime_xyz")
	if unknown.Installed {
		t.Errorf("expected unknown runtime to not be installed")
	}
}
