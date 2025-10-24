package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ===== TYPES.GO UNIT TESTS =====

func TestGetTargetBasic(t *testing.T) {
	// Setup test configuration
	original := cfg
	defer func() { cfg = original }()

	cfg = Config{
		Targets: map[string]Target{
			"simple": {
				Run: []string{"echo simple"},
			},
			"with-deps": {
				Run:  []string{"echo with-deps"},
				Deps: []string{"simple"},
			},
			"complex": {
				Run:  []string{"echo step1", "echo step2"},
				Deps: []string{"simple", "with-deps"},
			},
			"empty": {},
		},
		Vars: make(map[string]Var),
	}

	tests := []struct {
		name         string
		targetName   string
		expectedRun  []string
		expectedDeps []string
	}{
		{
			name:        "Simple target",
			targetName:  "simple",
			expectedRun: []string{"echo simple"},
		},
		{
			name:         "Target with dependencies",
			targetName:   "with-deps",
			expectedRun:  []string{"echo with-deps"},
			expectedDeps: []string{"simple"},
		},
		{
			name:         "Complex target",
			targetName:   "complex",
			expectedRun:  []string{"echo step1", "echo step2"},
			expectedDeps: []string{"simple", "with-deps"},
		},
		{
			name:        "Empty target",
			targetName:  "empty",
			expectedRun: nil,
		},
		{
			name:        "Non-existent target",
			targetName:  "nonexistent",
			expectedRun: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := GetTarget(tt.targetName)

			// Check run commands
			if len(target.Run) != len(tt.expectedRun) {
				t.Errorf("GetTarget(%v).Run length = %d, want %d", tt.targetName, len(target.Run), len(tt.expectedRun))
				return
			}

			for i, cmd := range target.Run {
				if cmd != tt.expectedRun[i] {
					t.Errorf("GetTarget(%v).Run[%d] = %v, want %v", tt.targetName, i, cmd, tt.expectedRun[i])
				}
			}

			// Check dependencies
			if len(target.Deps) != len(tt.expectedDeps) {
				t.Errorf("GetTarget(%v).Deps length = %d, want %d", tt.targetName, len(target.Deps), len(tt.expectedDeps))
				return
			}

			for i, dep := range target.Deps {
				if dep != tt.expectedDeps[i] {
					t.Errorf("GetTarget(%v).Deps[%d] = %v, want %v", tt.targetName, i, dep, tt.expectedDeps[i])
				}
			}
		})
	}
}

func TestConfigLoadFromFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		fileName     string
		fileContent  string
		expectError  bool
		validateFunc func(*testing.T, Config)
	}{
		{
			name:     "Valid minimal config",
			fileName: "minimal.yaml",
			fileContent: `targets:
  build:
    run:
      - "echo building"
`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg Config) {
				if len(cfg.Targets) != 1 {
					t.Errorf("Expected 1 target, got %d", len(cfg.Targets))
				}
				if target, exists := cfg.Targets["build"]; !exists {
					t.Error("Expected 'build' target to exist")
				} else if len(target.Run) != 1 || target.Run[0] != "echo building" {
					t.Errorf("Expected 'echo building', got %v", target.Run)
				}
			},
		},
		{
			name:     "Complete config with variables",
			fileName: "complete.yaml",
			fileContent: `vars:
  CC: "gcc"
  OUTPUT: "app.exe"

targets:
  build:
    deps:
      - "prepare"
    run:
      - "$CC -o $OUTPUT main.c"
  
  prepare:
    run:
      - "echo preparing"

prologue:
  run:
    - "echo starting build"

epilogue:
  run:
    - "echo build completed"
`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg Config) {
				// Check variables
				if len(cfg.Vars) != 2 {
					t.Errorf("Expected 2 variables, got %d", len(cfg.Vars))
				}
				if string(cfg.Vars["CC"]) != "gcc" {
					t.Errorf("Expected CC=gcc, got %v", cfg.Vars["CC"])
				}

				// Check targets
				if len(cfg.Targets) != 2 {
					t.Errorf("Expected 2 targets, got %d", len(cfg.Targets))
				}

				// Check prologue/epilogue
				if len(cfg.Prologue.Run) != 1 {
					t.Errorf("Expected 1 prologue command, got %d", len(cfg.Prologue.Run))
				}
				if len(cfg.Epilogue.Run) != 1 {
					t.Errorf("Expected 1 epilogue command, got %d", len(cfg.Epilogue.Run))
				}
			},
		},
		{
			name:     "Invalid YAML syntax",
			fileName: "invalid.yaml",
			fileContent: `targets:
  build:
    run
      - "invalid yaml"
`,
			expectError: true,
			validateFunc: func(t *testing.T, cfg Config) {
				// Config should remain unchanged on error
			},
		},
		{
			name:        "Empty file",
			fileName:    "empty.yaml",
			fileContent: "",
			expectError: true, // Empty file should cause error
			validateFunc: func(t *testing.T, cfg Config) {
				// Config should remain unchanged on error
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tempDir, tt.fileName)
			err := os.WriteFile(filePath, []byte(tt.fileContent), 0600)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Reset global config
			cfg = Config{
				Targets: make(map[string]Target),
				Vars:    make(map[string]Var),
			}

			// Load config
			err = loadConfig(filePath)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("loadConfig() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("loadConfig() unexpected error: %v", err)
			}

			// Validate config if no error expected
			if !tt.expectError {
				tt.validateFunc(t, cfg)
			}
		})
	}
}

func TestConfigSecurityValidation(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid path in temp directory",
			filePath:    filepath.Join(tempDir, "valid.yaml"),
			expectError: false,
		},
		{
			name:        "Path traversal attempt (../)",
			filePath:    "../../../etc/passwd",
			expectError: true,
			errorMsg:    "configuration file not found", // Orpheus error message
		},
		{
			name:        "Path traversal attempt (..\\)",
			filePath:    "..\\..\\..\\windows\\system32\\config\\sam",
			expectError: true,
			errorMsg:    "configuration file not found", // Orpheus error message
		},
		{
			name:        "Absolute path to system file",
			filePath:    "/etc/passwd",
			expectError: true,
			errorMsg:    "configuration file not found", // Orpheus error message
		},
		{
			name:        "Non-existent file in safe directory",
			filePath:    filepath.Join(tempDir, "nonexistent.yaml"),
			expectError: true,
			errorMsg:    "configuration file not found", // Orpheus error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create valid test file if needed
			if !tt.expectError {
				err := os.WriteFile(tt.filePath, []byte("targets:\n  test:\n    run:\n      - echo test"), 0600)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			// Reset global config
			cfg = Config{
				Targets: make(map[string]Target),
				Vars:    make(map[string]Var),
			}

			// Attempt to load config
			err := loadConfig(tt.filePath)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("loadConfig(%v) expected error but got none", tt.filePath)
			}
			if !tt.expectError && err != nil {
				t.Errorf("loadConfig(%v) unexpected error: %v", tt.filePath, err)
			}

			// Check specific error message if provided
			if tt.expectError && tt.errorMsg != "" && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
					t.Errorf("loadConfig(%v) error = %v, expected to contain %v", tt.filePath, err, tt.errorMsg)
				}
			}
		})
	}
}

func TestTargetDependencyResolution(t *testing.T) {
	// Setup complex dependency scenario
	original := cfg
	defer func() { cfg = original }()

	cfg = Config{
		Targets: map[string]Target{
			"app": {
				Run:  []string{"echo building app"},
				Deps: []string{"compile", "assets"},
			},
			"compile": {
				Run:  []string{"echo compiling"},
				Deps: []string{"deps"},
			},
			"assets": {
				Run: []string{"echo building assets"},
			},
			"deps": {
				Run: []string{"echo installing dependencies"},
			},
			"circular1": {
				Run:  []string{"echo circular1"},
				Deps: []string{"circular2"},
			},
			"circular2": {
				Run:  []string{"echo circular2"},
				Deps: []string{"circular1"},
			},
		},
		Vars: make(map[string]Var),
	}

	tests := []struct {
		name         string
		targetName   string
		shouldExist  bool
		expectedDeps []string
	}{
		{
			name:         "No dependencies",
			targetName:   "assets",
			shouldExist:  true,
			expectedDeps: []string{},
		},
		{
			name:         "Single dependency",
			targetName:   "compile",
			shouldExist:  true,
			expectedDeps: []string{"deps"},
		},
		{
			name:         "Multiple dependencies",
			targetName:   "app",
			shouldExist:  true,
			expectedDeps: []string{"compile", "assets"},
		},
		{
			name:        "Non-existent target",
			targetName:  "nonexistent",
			shouldExist: false,
		},
		{
			name:         "Circular dependency (should still return target)",
			targetName:   "circular1",
			shouldExist:  true,
			expectedDeps: []string{"circular2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := GetTarget(tt.targetName)

			if tt.shouldExist {
				// Target should exist and have expected dependencies
				if len(target.Run) == 0 && tt.targetName != "nonexistent" {
					t.Errorf("GetTarget(%v) should have run commands", tt.targetName)
				}

				if len(target.Deps) != len(tt.expectedDeps) {
					t.Errorf("GetTarget(%v).Deps length = %d, want %d", tt.targetName, len(target.Deps), len(tt.expectedDeps))
					return
				}

				for i, expectedDep := range tt.expectedDeps {
					if target.Deps[i] != expectedDep {
						t.Errorf("GetTarget(%v).Deps[%d] = %v, want %v", tt.targetName, i, target.Deps[i], expectedDep)
					}
				}
			} else {
				// Non-existent target should return empty target
				if len(target.Run) != 0 || len(target.Deps) != 0 {
					t.Errorf("GetTarget(%v) should return empty target for non-existent target", tt.targetName)
				}
			}
		})
	}
}

// ===== BENCHMARK TESTS =====

func BenchmarkGetTargetSimple(b *testing.B) {
	cfg.Targets = map[string]Target{
		"benchmark": {
			Run: []string{"echo benchmark"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTarget("benchmark")
	}
}

func BenchmarkGetTargetWithDeps(b *testing.B) {
	cfg.Targets = map[string]Target{
		"benchmark": {
			Run:  []string{"echo benchmark"},
			Deps: []string{"dep1", "dep2", "dep3"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTarget("benchmark")
	}
}

func BenchmarkGetTargetNonExistent(b *testing.B) {
	cfg.Targets = map[string]Target{
		"existing": {
			Run: []string{"echo existing"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTarget("nonexistent")
	}
}
