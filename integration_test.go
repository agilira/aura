package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ===== INTEGRATION TESTS =====

func TestE2EBuildWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary project directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "aura.yaml")

	// Create a realistic build configuration
	buildConfig := `vars:
  CC: "go"
  OUTPUT: "testapp"
  SRC: "*.go"

targets:
  clean:
    run:
      - "echo Cleaning build artifacts"

  deps:
    run:
      - "echo Installing dependencies"

  build:
    deps:
      - "deps"
    run:
      - "echo Building $@ with $CC"
      - "echo Source files: $SRC"
      - "echo Output: $OUTPUT"

  test:
    deps:
      - "build"
    run:
      - "echo Running tests for $@"
      - "echo Test completed successfully"

prologue:
  run:
    - "echo === Build started at $TIMESTAMP ==="
    - "echo Working directory: $cwd"

epilogue:
  run:
    - "echo === Build completed ==="
`

	// Write config file
	err := os.WriteFile(configPath, []byte(buildConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	tests := []struct {
		name          string
		targets       []string
		expectSuccess bool
		expectedLogs  []string
	}{
		{
			name:          "Build single target",
			targets:       []string{"deps"},
			expectSuccess: true,
			expectedLogs:  []string{"Installing dependencies"},
		},
		{
			name:          "Build with dependencies",
			targets:       []string{"build"},
			expectSuccess: true,
			expectedLogs:  []string{"Installing dependencies", "Building build"},
		},
		{
			name:          "Build complex target",
			targets:       []string{"test"},
			expectSuccess: true,
			expectedLogs:  []string{"Building build", "Running tests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global config
			cfg = Config{
				Targets: make(map[string]Target),
				Vars:    make(map[string]Var),
			}

			// Load config
			err := loadConfig("aura.yaml")
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Simulate build execution
			success := true
			var capturedOutput []string

			// Execute prologue
			if len(cfg.Prologue.Run) > 0 {
				for _, cmd := range cfg.Prologue.Run {
					processed := ParseVars(cmd, "prologue")
					output, err := ExecuteCommandWithContext(processed, false, false)
					if err != nil {
						success = false
						break
					}
					capturedOutput = append(capturedOutput, output)
				}
			}

			// Execute targets (with dependency resolution)
			for _, targetName := range tt.targets {
				target := GetTarget(targetName)
				if len(target.Run) == 0 {
					success = false
					break
				}

				// Execute dependencies first
				for _, dep := range target.Deps {
					depTarget := GetTarget(dep)
					for _, cmd := range depTarget.Run {
						processed := ParseVars(cmd, dep)
						output, err := ExecuteCommandWithContext(processed, false, false)
						if err != nil {
							success = false
							break
						}
						capturedOutput = append(capturedOutput, output)
					}
					if !success {
						break
					}
				}

				// Execute target commands
				for _, cmd := range target.Run {
					processed := ParseVars(cmd, targetName)
					output, err := ExecuteCommandWithContext(processed, false, false)
					if err != nil {
						success = false
						break
					}
					capturedOutput = append(capturedOutput, output)
				}
			}

			// Execute epilogue
			if len(cfg.Epilogue.Run) > 0 {
				for _, cmd := range cfg.Epilogue.Run {
					processed := ParseVars(cmd, "epilogue")
					output, err := ExecuteCommandWithContext(processed, false, false)
					if err != nil {
						success = false
						break
					}
					capturedOutput = append(capturedOutput, output)
				}
			}

			// Verify results
			if tt.expectSuccess && !success {
				t.Errorf("Expected success but build failed")
			}

			// Verify expected logs appear in output
			allOutput := strings.Join(capturedOutput, " ")
			for _, expectedLog := range tt.expectedLogs {
				if !strings.Contains(allOutput, expectedLog) {
					t.Errorf("Expected log '%s' not found in output: %s", expectedLog, allOutput)
				}
			}
		})
	}
}

func TestE2ETemplateGeneration(t *testing.T) {
	tempDir := t.TempDir()

	templates := []string{"go", "rust", "node", "basic"}

	for _, tmpl := range templates {
		t.Run("Template_"+tmpl, func(t *testing.T) {
			// Generate template
			content := generateTemplate(tmpl)
			if content == "" {
				t.Fatalf("Template %s should not be empty", tmpl)
			}

			// Write to file
			configPath := filepath.Join(tempDir, tmpl+"-aura.yaml")
			err := os.WriteFile(configPath, []byte(content), 0600)
			if err != nil {
				t.Fatalf("Failed to write template: %v", err)
			}

			// Reset and load config
			cfg = Config{
				Targets: make(map[string]Target),
				Vars:    make(map[string]Var),
			}

			err = loadConfig(configPath)
			if err != nil {
				t.Fatalf("Failed to load generated template: %v", err)
			}

			// Verify template has essential targets
			expectedTargets := map[string][]string{
				"go":    {"build", "test", "clean"},
				"rust":  {"build", "test", "clean"},
				"node":  {"install", "build", "start"},
				"basic": {"build", "clean"},
			}

			for _, expectedTarget := range expectedTargets[tmpl] {
				target := GetTarget(expectedTarget)
				if len(target.Run) == 0 {
					t.Errorf("Template %s should have target %s", tmpl, expectedTarget)
				}
			}

			// Verify variables are defined
			if len(cfg.Vars) == 0 {
				t.Errorf("Template %s should define variables", tmpl)
			}
		})
	}
}

func TestE2EDryRunMode(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "aura.yaml")

	// Create config with potentially destructive commands
	dryRunConfig := `targets:
  dangerous:
    run:
      - "echo This would be dangerous: rm -rf /"
      - "echo Another dangerous command"
      
  safe:
    run:
      - "echo This is safe"
`

	err := os.WriteFile(configPath, []byte(dryRunConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Load config
	cfg = Config{
		Targets: make(map[string]Target),
		Vars:    make(map[string]Var),
	}

	err = loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test dry run mode
	target := GetTarget("dangerous")
	for _, cmd := range target.Run {
		processed := ParseVars(cmd, "dangerous")

		// Execute in dry run mode
		output, err := ExecuteCommandWithContext(processed, true, true)

		// In dry run mode, should not execute and should not error
		if err != nil {
			t.Errorf("Dry run should not produce error: %v", err)
		}

		// Should not produce actual output in dry run
		if output != "" {
			t.Errorf("Dry run should not produce output, got: %s", output)
		}
	}
}

func TestE2EErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "aura.yaml")

	// Create config with failing commands
	errorConfig := `targets:
  fail-early:
    run:
      - "echo Starting"
      - "false"  # This will fail
      - "echo This should not run"
      
  mixed:
    run:
      - "echo Good command"
      - "invalidcommand12345"
      - "echo After error"
`

	err := os.WriteFile(configPath, []byte(errorConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Load config
	cfg = Config{
		Targets: make(map[string]Target),
		Vars:    make(map[string]Var),
	}

	err = loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	tests := []struct {
		name        string
		targetName  string
		expectError bool
	}{
		{
			name:        "Target with failing command",
			targetName:  "fail-early",
			expectError: true,
		},
		{
			name:        "Target with invalid command",
			targetName:  "mixed",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := GetTarget(tt.targetName)

			// Execute target using ExecuteAllWithContext
			err := ExecuteAllWithContext(tt.targetName, &target, false, false)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// ===== BENCHMARK INTEGRATION TESTS =====

func BenchmarkE2EFullBuild(b *testing.B) {
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "aura.yaml")

	benchConfig := `vars:
  OUTPUT: "benchapp"

targets:
  benchmark:
    run:
      - "echo Building $OUTPUT"
      - "echo Compilation step"
      - "echo Linking step"
      - "echo Build complete"
`

	err := os.WriteFile(configPath, []byte(benchConfig), 0600)
	if err != nil {
		b.Fatalf("Failed to create benchmark config: %v", err)
	}

	// Load config once
	cfg = Config{
		Targets: make(map[string]Target),
		Vars:    make(map[string]Var),
	}

	err = loadConfig(configPath)
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		target := GetTarget("benchmark")
		_ = ExecuteAllWithContext("benchmark", &target, false, false)
	}
}
