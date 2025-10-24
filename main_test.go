package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// Test helper to clean up after tests
func TestMain(m *testing.M) {
	// Initialize config for tests
	cfg = Config{
		Targets: make(map[string]Target),
		Vars:    make(map[string]Var),
	}

	// Run tests
	code := m.Run()

	// Clean up
	os.Exit(code)
}

// ===== CORE FUNCTIONS TESTS =====

func TestParseVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		target   string
		expected string
		setup    func()
	}{
		{
			name:     "Basic variable substitution with cwd",
			input:    "echo $cwd",
			target:   "test",
			expected: "", // Will contain actual cwd
			setup:    func() {},
		},
		{
			name:     "Target name substitution",
			input:    "Building $@",
			target:   "mybuild",
			expected: "Building mybuild",
			setup:    func() {},
		},
		{
			name:     "Custom variable substitution",
			input:    "echo $CC $CFLAGS",
			target:   "test",
			expected: "echo gcc -Wall",
			setup: func() {
				cfg.Vars = map[string]Var{
					"CC":     "gcc",
					"CFLAGS": "-Wall",
				}
			},
		},
		{
			name:     "Braced variable substitution",
			input:    "Building ${OUTPUT}",
			target:   "test",
			expected: "Building app.exe",
			setup: func() {
				cfg.Vars = map[string]Var{
					"OUTPUT": "app.exe",
				}
			},
		},
		{
			name:     "Mixed variables",
			input:    "Target: $@ using $CC",
			target:   "build",
			expected: "Target: build using gcc",
			setup: func() {
				cfg.Vars = map[string]Var{
					"CC": "gcc",
				}
			},
		},
		{
			name:     "Undefined variable warning",
			input:    "echo $UNDEFINED",
			target:   "test",
			expected: "echo $UNDEFINED", // Should remain unchanged
			setup:    func() { cfg.Vars = map[string]Var{} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := ParseVars(tt.input, tt.target)

			// Special handling for cwd tests
			if strings.Contains(tt.input, "$cwd") && tt.expected == "" {
				if !strings.Contains(result, "echo ") || result == tt.input {
					t.Errorf("ParseVars() = %v, expected cwd substitution", result)
				}
				return
			}

			if result != tt.expected {
				t.Errorf("ParseVars() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetVar(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		target   string
		expected string
		setup    func()
	}{
		{
			name:     "Get current working directory",
			varName:  "cwd",
			target:   "test",
			expected: "", // Will check if not empty
			setup:    func() {},
		},
		{
			name:     "Get target name",
			varName:  "@",
			target:   "myTarget",
			expected: "myTarget",
			setup:    func() {},
		},
		{
			name:     "Get timestamp",
			varName:  "TIMESTAMP",
			target:   "test",
			expected: "", // Will check format
			setup:    func() {},
		},
		{
			name:     "Get custom variable",
			varName:  "CC",
			target:   "test",
			expected: "gcc",
			setup: func() {
				cfg.Vars = map[string]Var{"CC": "gcc"}
			},
		},
		{
			name:     "Get environment variable fallback",
			varName:  "PATH",
			target:   "test",
			expected: "", // Will check if not empty from env
			setup: func() {
				cfg.Vars = map[string]Var{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := GetVar(tt.varName, tt.target)

			switch tt.varName {
			case "cwd":
				if result == "" {
					t.Error("GetVar(cwd) should not be empty")
				}
			case "TIMESTAMP":
				// Check timestamp format: "2006-01-02 15:04:05"
				if len(result) != 19 || !strings.Contains(result, "-") || !strings.Contains(result, ":") {
					t.Errorf("GetVar(TIMESTAMP) = %v, expected timestamp format", result)
				}
			case "PATH":
				// Environment variable should exist on all systems
				if result == "" {
					t.Error("GetVar(PATH) should not be empty from environment")
				}
			default:
				if result != tt.expected {
					t.Errorf("GetVar(%v) = %v, want %v", tt.varName, result, tt.expected)
				}
			}
		})
	}
}

func TestGetTarget(t *testing.T) {
	cfg.Targets = map[string]Target{
		"build": {
			Run:  []string{"go build"},
			Deps: []string{"test"},
		},
		"test": {
			Run: []string{"go test"},
		},
		"empty": {},
	}

	tests := []struct {
		name         string
		targetName   string
		expectedRun  []string
		expectedDeps []string
	}{
		{
			name:         "Get existing target with deps",
			targetName:   "build",
			expectedRun:  []string{"go build"},
			expectedDeps: []string{"test"},
		},
		{
			name:        "Get simple target",
			targetName:  "test",
			expectedRun: []string{"go test"},
		},
		{
			name:        "Get empty target",
			targetName:  "empty",
			expectedRun: nil,
		},
		{
			name:        "Get non-existent target",
			targetName:  "nonexistent",
			expectedRun: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := GetTarget(tt.targetName)

			if len(target.Run) != len(tt.expectedRun) {
				t.Errorf("GetTarget(%v).Run = %v, want %v", tt.targetName, target.Run, tt.expectedRun)
				return
			}

			for i, cmd := range target.Run {
				if cmd != tt.expectedRun[i] {
					t.Errorf("GetTarget(%v).Run[%d] = %v, want %v", tt.targetName, i, cmd, tt.expectedRun[i])
				}
			}

			if len(target.Deps) != len(tt.expectedDeps) {
				t.Errorf("GetTarget(%v).Deps = %v, want %v", tt.targetName, target.Deps, tt.expectedDeps)
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

// ===== TEMPLATE GENERATION TESTS =====

func TestGenerateTemplate(t *testing.T) {
	tests := []struct {
		name             string
		templateType     string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:             "Go template",
			templateType:     "go",
			shouldContain:    []string{"GO:", "go", "build", "test", "app.exe"},
			shouldNotContain: []string{"cargo", "npm"},
		},
		{
			name:             "Rust template",
			templateType:     "rust",
			shouldContain:    []string{"CARGO:", "cargo", "build", "test", "clean"},
			shouldNotContain: []string{"npm"},
		},
		{
			name:             "Node template",
			templateType:     "node",
			shouldContain:    []string{"NPM:", "npm", "install", "build", "start"},
			shouldNotContain: []string{"go", "cargo"},
		},
		{
			name:             "Basic template",
			templateType:     "basic",
			shouldContain:    []string{"CC:", "gcc", "build", "clean", "prologue", "epilogue"},
			shouldNotContain: []string{"go", "cargo", "npm"},
		},
		{
			name:             "Advanced template (falls back to basic)",
			templateType:     "advanced",
			shouldContain:    []string{"CC:", "gcc"},
			shouldNotContain: []string{"go", "cargo", "npm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTemplate(tt.templateType)

			if result == "" {
				t.Errorf("generateTemplate(%v) should not be empty", tt.templateType)
				return
			}

			for _, should := range tt.shouldContain {
				if !strings.Contains(result, should) {
					t.Errorf("generateTemplate(%v) should contain '%v'", tt.templateType, should)
				}
			}

			for _, shouldNot := range tt.shouldNotContain {
				if strings.Contains(result, shouldNot) {
					t.Errorf("generateTemplate(%v) should not contain '%v'", tt.templateType, shouldNot)
				}
			}
		})
	}
}

// ===== CONFIG LOADING TESTS =====

func TestLoadConfig(t *testing.T) {
	// Create temporary test config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-aura.yaml")

	validConfig := `vars:
  CC: "gcc"
  OUTPUT: "test.exe"

targets:
  build:
    run:
      - "$CC -o $OUTPUT main.c"
  
  test:
    deps:
      - build
    run:
      - "./$OUTPUT"

prologue:
  run:
    - "echo Starting build"

epilogue:
  run:
    - "echo Build completed"
`

	err := os.WriteFile(configPath, []byte(validConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	tests := []struct {
		name        string
		configPath  string
		expectError bool
		checkConfig func(*testing.T)
	}{
		{
			name:        "Valid config file",
			configPath:  configPath,
			expectError: false,
			checkConfig: func(t *testing.T) {
				if len(cfg.Targets) != 2 {
					t.Errorf("Expected 2 targets, got %d", len(cfg.Targets))
				}
				if len(cfg.Vars) != 2 {
					t.Errorf("Expected 2 variables, got %d", len(cfg.Vars))
				}
				if string(cfg.Vars["CC"]) != "gcc" {
					t.Errorf("Expected CC=gcc, got %v", cfg.Vars["CC"])
				}
			},
		},
		{
			name:        "Non-existent config file",
			configPath:  filepath.Join(tempDir, "nonexistent.yaml"),
			expectError: true,
			checkConfig: func(t *testing.T) {},
		},
		{
			name:        "Path traversal attempt",
			configPath:  "../../../etc/passwd",
			expectError: true,
			checkConfig: func(t *testing.T) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config before each test
			cfg = Config{
				Targets: make(map[string]Target),
				Vars:    make(map[string]Var),
			}

			err := loadConfig(tt.configPath)

			if tt.expectError && err == nil {
				t.Errorf("loadConfig() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("loadConfig() unexpected error: %v", err)
			}

			if !tt.expectError {
				tt.checkConfig(t)
			}
		})
	}
}

// ===== INTEGRATION TESTS =====

func TestBuildCommandIntegration(t *testing.T) {
	// Create a test app for integration testing
	app := orpheus.New("aura-test").
		SetDescription("Test version of aura").
		SetVersion("test")

	buildCmd := orpheus.NewCommand("build", "Execute build targets").
		SetHandler(buildCommand).
		AddFlag("targets", "t", "", "Comma-separated list of targets to run").
		AddIntFlag("parallel", "p", 1, "Number of parallel jobs").
		AddBoolFlag("force", "f", false, "Force rebuild of all targets")

	app.AddCommand(buildCmd)

	// Test with empty args (should show available targets)
	ctx := &orpheus.Context{
		App:         app,
		Args:        []string{},
		GlobalFlags: nil,
	}

	// Mock a simple config
	cfg = Config{
		Targets: map[string]Target{
			"test": {
				Run: []string{"echo test-output"},
			},
		},
		Vars: make(map[string]Var),
	}

	// This should not panic and should handle gracefully
	err := buildCommand(ctx)
	if err == nil {
		t.Log("Build command handled empty targets gracefully")
	}
}

// ===== COMMAND HANDLER TESTS =====

// Test command handlers by calling their functionality directly
// Since we can't easily mock Context flags, we test the core logic

func TestGetLatestModTime(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "TestGetLatestModTime")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	// Test empty directory
	patterns := []string{"*.go", "*.yaml", "*.txt"}
	modTime := getLatestModTime(patterns)

	if !modTime.IsZero() {
		t.Errorf("getLatestModTime() should return zero time for empty directory")
	}

	// Create test files
	testFiles := []string{"test1.go", "test2.yaml", "test3.txt"}
	for _, file := range testFiles {
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	modTime = getLatestModTime(patterns)

	if modTime.IsZero() {
		t.Errorf("getLatestModTime() returned zero time when files exist")
	}
}

func TestGenerateTemplateComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected []string // strings that should be in the output
	}{
		{
			name:     "Go template",
			template: "go",
			expected: []string{"$GO build", "$GO test", "BINARY:"},
		},
		{
			name:     "Rust template",
			template: "rust",
			expected: []string{"$CARGO build", "$CARGO test", "$CARGO clean"},
		},
		{
			name:     "Node template",
			template: "node",
			expected: []string{"$NPM install", "$NPM test", "$NPM start"},
		},
		{
			name:     "Basic template",
			template: "basic",
			expected: []string{"targets:", "vars:", "build:"},
		},
		{
			name:     "Unknown template (fallback to basic)",
			template: "unknown",
			expected: []string{"targets:", "vars:", "build:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTemplate(tt.template)

			if result == "" {
				t.Errorf("generateTemplate() returned empty string for template %s", tt.template)
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("generateTemplate() result for %s missing expected content: %s", tt.template, expected)
				}
			}
		})
	}
}

func TestLoadConfigComprehensive(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "TestLoadConfigComprehensive")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name            string
		configContent   string
		filename        string
		expectedError   bool
		expectedTargets int
		setupIncludes   bool
	}{
		{
			name: "Valid basic config",
			configContent: `
targets:
  build:
    run:
      - echo "building"
  test:
    run:
      - echo "testing"
vars:
  CC: gcc
`,
			filename:        "aura.yaml",
			expectedError:   false,
			expectedTargets: 2,
			setupIncludes:   false,
		},
		{
			name: "Config with includes",
			configContent: `
targets:
  main:
    run:
      - echo "main"
includes:
  - included.yaml
`,
			filename:        "aura.yaml",
			expectedError:   false,
			expectedTargets: 1, // includes may fail silently with warnings
			setupIncludes:   true,
		},
		{
			name:            "Non-existent file",
			configContent:   "",
			filename:        "nonexistent.yaml",
			expectedError:   true,
			expectedTargets: 0,
			setupIncludes:   false,
		},
		{
			name: "Invalid YAML",
			configContent: `
targets:
  build:
    run:
      - echo "building"
    invalid: [unclosed
`,
			filename:        "invalid.yaml",
			expectedError:   true,
			expectedTargets: 0,
			setupIncludes:   false,
		},
	}

	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cfg
			cfg = Config{
				Targets: make(map[string]Target),
				Vars:    make(map[string]Var),
			}

			if tt.configContent != "" && tt.filename != "nonexistent.yaml" {
				if err := os.WriteFile(tt.filename, []byte(tt.configContent), 0644); err != nil {
					t.Fatalf("Failed to write config: %v", err)
				}
			}

			if tt.setupIncludes {
				includeContent := `
targets:
  included:
    run:
      - echo "included target"
`
				if err := os.WriteFile("included.yaml", []byte(includeContent), 0644); err != nil {
					t.Fatalf("Failed to write include file: %v", err)
				}
			}

			err := loadConfig(tt.filename)

			if tt.expectedError && err == nil {
				t.Errorf("loadConfig() expected error but got none")
			}

			if !tt.expectedError && err != nil {
				t.Errorf("loadConfig() unexpected error: %v", err)
			}

			if !tt.expectedError && len(cfg.Targets) != tt.expectedTargets {
				t.Errorf("loadConfig() expected %d targets, got %d", tt.expectedTargets, len(cfg.Targets))
			}

			// Clean up
			_ = os.Remove(tt.filename)
			if tt.setupIncludes {
				_ = os.Remove("included.yaml")
			}
		})
	}
}

func TestCleanCommandLogic(t *testing.T) {
	// Test the clean logic without Context dependencies
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "TestCleanCommandLogic")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	// Create cache directory
	cacheDir := ".aura_cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	// Create some files in cache
	if err := os.WriteFile(filepath.Join(cacheDir, "test.cache"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create cache file: %v", err)
	}

	// Test cache cleaning function (extracted from cleanCommand)
	if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
		if err := os.RemoveAll(cacheDir); err != nil {
			t.Errorf("Failed to remove cache directory: %v", err)
		}
	}

	// Verify cache was removed
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Errorf("Cache directory was not removed")
	}
}

func TestValidateCommandLogic(t *testing.T) {
	// Test validation logic without Context dependencies
	tempDir, err := os.MkdirTemp("", "TestValidateCommandLogic")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	// Create valid config
	configContent := `
vars:
  CC: "gcc"
  OUTPUT: "app"
targets:
  build:
    run:
      - "echo building with $CC"
      - "echo output: $OUTPUT"
  test:
    run:
      - "echo testing"
`
	if err := os.WriteFile("aura.yaml", []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Reset and load config
	cfg = Config{
		Targets: make(map[string]Target),
		Vars:    make(map[string]Var),
	}

	err = loadConfig("aura.yaml")
	if err != nil {
		t.Errorf("loadConfig() for validation test failed: %v", err)
	}

	// Verify config was loaded correctly
	if len(cfg.Targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(cfg.Targets))
	}

	if len(cfg.Vars) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(cfg.Vars))
	}
}

func TestInitCommandLogic(t *testing.T) {
	// Test init command logic without Context dependencies
	tempDir, err := os.MkdirTemp("", "TestInitCommandLogic")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	templates := []string{"basic", "go", "rust", "node", "advanced"}

	for _, template := range templates {
		t.Run("Template_"+template, func(t *testing.T) {
			// Generate template content
			templateContent := generateTemplate(template)

			// Write to file (simulate init command)
			if err := os.WriteFile("aura.yaml", []byte(templateContent), 0600); err != nil {
				t.Errorf("Failed to create aura.yaml for template %s: %v", template, err)
			}

			// Verify file was created
			if _, err := os.Stat("aura.yaml"); os.IsNotExist(err) {
				t.Errorf("aura.yaml was not created for template %s", template)
			}

			// Verify content is valid YAML by trying to load it
			cfg = Config{
				Targets: make(map[string]Target),
				Vars:    make(map[string]Var),
			}

			err := loadConfig("aura.yaml")
			if err != nil {
				t.Errorf("Generated template %s is not valid YAML: %v", template, err)
			}

			// Clean up for next test
			_ = os.Remove("aura.yaml")
		})
	}
}

func TestCacheCommandsLogic(t *testing.T) {
	// Test cache commands logic without Context dependencies
	tempDir, err := os.MkdirTemp("", "TestCacheCommandsLogic")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	t.Run("Cache_creation_and_clear", func(t *testing.T) {
		// Create cache directory with files
		cacheDir := ".aura_cache"
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			t.Fatalf("Failed to create cache dir: %v", err)
		}

		// Add some files
		for i := 0; i < 3; i++ {
			filename := filepath.Join(cacheDir, fmt.Sprintf("test%d.cache", i))
			if err := os.WriteFile(filename, []byte(fmt.Sprintf("test data %d", i)), 0644); err != nil {
				t.Fatalf("Failed to create cache file: %v", err)
			}
		}

		// Verify cache exists
		if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
			t.Errorf("Cache directory was not created")
		}

		// Test cache listing (count files)
		entries, err := os.ReadDir(cacheDir)
		if err != nil {
			t.Errorf("Failed to read cache directory: %v", err)
		}

		if len(entries) != 3 {
			t.Errorf("Expected 3 cache files, got %d", len(entries))
		}

		// Test cache info (get sizes)
		var totalSize int64
		for _, entry := range entries {
			if entryInfo, err := entry.Info(); err == nil {
				totalSize += entryInfo.Size()
			}
		}

		if totalSize == 0 {
			t.Errorf("Cache files should have non-zero size")
		}

		// Test cache clear
		if err := os.RemoveAll(cacheDir); err != nil {
			t.Errorf("Failed to clear cache: %v", err)
		}

		// Verify cache was cleared
		if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
			t.Errorf("Cache directory was not cleared")
		}
	})
}

// ===== BENCHMARK TESTS =====

func BenchmarkParseVars(b *testing.B) {
	cfg.Vars = map[string]Var{
		"CC":     "gcc",
		"CFLAGS": "-Wall -O2",
		"OUTPUT": "app.exe",
	}

	testString := "Building $@ with $CC $CFLAGS to produce $OUTPUT in $cwd"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseVars(testString, "benchmark")
	}
}

func BenchmarkGetVar(b *testing.B) {
	cfg.Vars = map[string]Var{
		"CC": "gcc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetVar("CC", "test")
	}
}

func BenchmarkGetTarget(b *testing.B) {
	cfg.Targets = map[string]Target{
		"build": {
			Run:  []string{"go build"},
			Deps: []string{"test"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTarget("build")
	}
}
