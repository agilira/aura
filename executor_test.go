package main

import (
	"runtime"
	"strings"
	"testing"
)

// ===== EXECUTOR UNIT TESTS =====

func TestExecuteCommand(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		expectOutput bool
		expectError  bool
	}{
		{
			name:         "Simple echo command",
			command:      "echo hello",
			expectOutput: true,
			expectError:  false,
		},
		{
			name:         "Command with output",
			command:      getTestCommand("version"),
			expectOutput: true,
			expectError:  false,
		},
		{
			name:         "Invalid command",
			command:      "invalidcommand12345",
			expectOutput: false,
			expectError:  true,
		},
		{
			name:         "Empty command",
			command:      "",
			expectOutput: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ExecuteCommand(tt.command)

			if tt.expectError && err == nil {
				t.Errorf("ExecuteCommand() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("ExecuteCommand() unexpected error: %v", err)
			}

			if tt.expectOutput && output == "" && !tt.expectError {
				t.Errorf("ExecuteCommand() expected output but got none")
			}
		})
	}
}

func TestExecuteCommandWithContext(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		verbose      bool
		dryRun       bool
		expectOutput bool
		expectError  bool
	}{
		{
			name:         "Normal execution with output",
			command:      "echo test-output",
			verbose:      false,
			dryRun:       false,
			expectOutput: true,
			expectError:  false,
		},
		{
			name:         "Verbose mode execution",
			command:      "echo verbose-test",
			verbose:      true,
			dryRun:       false,
			expectOutput: true,
			expectError:  false,
		},
		{
			name:         "Dry run mode (no execution)",
			command:      "echo dry-run-test",
			verbose:      true,
			dryRun:       true,
			expectOutput: false,
			expectError:  false,
		},
		{
			name:         "Error command in dry run",
			command:      "invalidcommand",
			verbose:      true,
			dryRun:       true,
			expectOutput: false,
			expectError:  false, // No error in dry run
		},
		{
			name:         "Error command in normal run",
			command:      "invalidcommand",
			verbose:      false,
			dryRun:       false,
			expectOutput: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ExecuteCommandWithContext(tt.command, tt.verbose, tt.dryRun)

			if tt.expectError && err == nil {
				t.Errorf("ExecuteCommandWithContext() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("ExecuteCommandWithContext() unexpected error: %v", err)
			}

			if tt.dryRun && output != "" {
				t.Errorf("ExecuteCommandWithContext() in dry run should not produce output, got: %v", output)
			}

			if tt.expectOutput && !tt.dryRun && output == "" && !tt.expectError {
				t.Errorf("ExecuteCommandWithContext() expected output but got none")
			}
		})
	}
}

func TestExecuteAllWithContext(t *testing.T) {
	tests := []struct {
		name        string
		target      Target
		targetName  string
		dryRun      bool
		expectError bool
		description string
	}{
		{
			name: "Sequential execution",
			target: Target{
				Run: []string{"echo step1", "echo step2", "echo step3"},
			},
			targetName:  "test-target",
			dryRun:      false,
			expectError: false,
			description: "Execute commands sequentially",
		},
		{
			name: "Dry run mode",
			target: Target{
				Run: []string{"echo dry1", "echo dry2"},
			},
			targetName:  "dry-target",
			dryRun:      true,
			expectError: false,
			description: "Dry run execution",
		},
		{
			name: "Error in sequence",
			target: Target{
				Run: []string{"echo good", "invalidcommand12345", "echo after-error"},
			},
			targetName:  "error-target",
			dryRun:      false,
			expectError: true,
			description: "Handle error in command sequence",
		},
		{
			name: "Empty command list",
			target: Target{
				Run: []string{},
			},
			targetName:  "empty-target",
			dryRun:      false,
			expectError: false,
			description: "Handle empty command list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteAllWithContext(tt.targetName, &tt.target, false, tt.dryRun)

			if tt.expectError && err == nil {
				t.Errorf("ExecuteAllWithContext() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("ExecuteAllWithContext() unexpected error: %v", err)
			}
		})
	}
}

func TestExecuteAllWithContextCancellation(t *testing.T) {
	// Test with a target that should complete quickly
	target := Target{
		Run: []string{"echo quick-test"},
	}

	err := ExecuteAllWithContext("test-target", &target, false, false)

	// Should complete without error
	if err != nil {
		t.Errorf("ExecuteAllWithContext() unexpected error: %v", err)
	}
}

func TestShellCommandGeneration(t *testing.T) {
	tests := []struct {
		name            string
		command         string
		expectedWindows string
		expectedUnix    string
	}{
		{
			name:            "Simple command",
			command:         "echo test",
			expectedWindows: "pwsh.exe",
			expectedUnix:    "/bin/sh",
		},
		{
			name:            "Complex command with pipes (Windows compatible)",
			command:         "echo test | findstr test",
			expectedWindows: "pwsh.exe",
			expectedUnix:    "/bin/sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the internal getShellCommand function
			// since it's not exported, but we can test that ExecuteCommand
			// works with different shell commands
			_, err := ExecuteCommand(tt.command)

			// The command should work regardless of the shell
			if err != nil && !strings.Contains(err.Error(), "executable file not found") {
				t.Errorf("Command execution failed unexpectedly: %v", err)
			}
		})
	}
}

// ===== HELPER FUNCTIONS =====

// getTestCommand returns a platform-appropriate test command
func getTestCommand(cmdType string) string {
	switch cmdType {
	case "version":
		if runtime.GOOS == "windows" {
			return "pwsh.exe -Command \"$PSVersionTable.PSVersion\""
		}
		return "sh -c 'echo $0' sh"
	case "echo":
		return "echo test-output"
	default:
		return "echo default"
	}
}

// ===== BENCHMARK TESTS =====

func BenchmarkExecuteCommand(b *testing.B) {
	command := "echo benchmark-test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExecuteCommand(command)
	}
}

func BenchmarkExecuteCommandWithContext(b *testing.B) {
	command := "echo benchmark-test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExecuteCommandWithContext(command, false, false)
	}
}

// ===== MISSING FUNCTION TESTS =====

func TestExecuteAll(t *testing.T) {
	target := Target{
		Run: []string{"echo test1", "echo test2"},
	}

	// ExecuteAll doesn't return error, just calls ExecuteAllWithContext
	ExecuteAll("test-target", &target)
}

func TestTargetRunDeps(t *testing.T) {
	// Mock cfg for this test
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = Config{
		Targets: map[string]Target{
			"dep1": {Run: []string{"echo dependency1"}},
			"dep2": {Run: []string{"echo dependency2"}},
		},
	}

	target := Target{
		Deps: []string{"dep1", "dep2"},
	}

	// RunDeps doesn't return error, just calls RunDepsWithContext
	target.RunDeps()
}

func TestTargetRunDepsWithContext(t *testing.T) {
	// Mock cfg for this test
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = Config{
		Targets: map[string]Target{
			"dep1": {Run: []string{"echo dependency1"}},
		},
	}

	target := Target{
		Deps: []string{"dep1"},
	}

	err := target.RunDepsWithContext(false, false)
	if err != nil {
		t.Errorf("RunDepsWithContext() unexpected error: %v", err)
	}

	// Test with file dependency
	target.Deps = []string{"file.txt"}
	err = target.RunDepsWithContext(true, false)
	if err != nil {
		t.Errorf("RunDepsWithContext() unexpected error with file dependency: %v", err)
	}

	// Test with invalid dependency
	target.Deps = []string{"nonexistent"}
	err = target.RunDepsWithContext(false, false)
	if err == nil {
		t.Errorf("RunDepsWithContext() expected error for invalid dependency")
	}
}

func TestConfigRunPrologue(t *testing.T) {
	config := &Config{
		Prologue: Target{
			Run: []string{"echo prologue1", "echo prologue2"},
		},
	}

	// RunPrologue doesn't return error, just calls RunPrologueWithContext
	config.RunPrologue()
}

func TestConfigRunPrologueWithContext(t *testing.T) {
	tests := []struct {
		name        string
		prologue    Target
		dryRun      bool
		expectError bool
	}{
		{
			name: "Normal prologue",
			prologue: Target{
				Run: []string{"echo prologue-test"},
			},
			dryRun:      false,
			expectError: false,
		},
		{
			name: "Dry run prologue",
			prologue: Target{
				Run: []string{"echo dry-prologue"},
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "Empty prologue",
			prologue: Target{
				Run: []string{},
			},
			dryRun:      false,
			expectError: false,
		},
		{
			name: "Error in prologue",
			prologue: Target{
				Run: []string{"invalidcommand12345"},
			},
			dryRun:      false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Prologue: tt.prologue,
			}

			err := config.RunPrologueWithContext(false, tt.dryRun)

			if tt.expectError && err == nil {
				t.Errorf("RunPrologueWithContext() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("RunPrologueWithContext() unexpected error: %v", err)
			}
		})
	}
}

func TestConfigRunEpilogue(t *testing.T) {
	config := &Config{
		Epilogue: Target{
			Run: []string{"echo epilogue1", "echo epilogue2"},
		},
	}

	// RunEpilogue doesn't return error, just calls RunEpilogueWithContext
	config.RunEpilogue()
}

func TestConfigRunEpilogueWithContext(t *testing.T) {
	tests := []struct {
		name        string
		epilogue    Target
		dryRun      bool
		expectError bool
	}{
		{
			name: "Normal epilogue",
			epilogue: Target{
				Run: []string{"echo epilogue-test"},
			},
			dryRun:      false,
			expectError: false,
		},
		{
			name: "Dry run epilogue",
			epilogue: Target{
				Run: []string{"echo dry-epilogue"},
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "Empty epilogue",
			epilogue: Target{
				Run: []string{},
			},
			dryRun:      false,
			expectError: false,
		},
		{
			name: "Error in epilogue",
			epilogue: Target{
				Run: []string{"invalidcommand12345"},
			},
			dryRun:      false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Epilogue: tt.epilogue,
			}

			err := config.RunEpilogueWithContext(false, tt.dryRun)

			if tt.expectError && err == nil {
				t.Errorf("RunEpilogueWithContext() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("RunEpilogueWithContext() unexpected error: %v", err)
			}
		})
	}
}

func TestRunTarget(t *testing.T) {
	// Mock cfg for this test
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = Config{
		Targets: map[string]Target{
			"test": {Run: []string{"echo target-test"}},
		},
	}

	// RunTarget doesn't return error, just calls runTargetWithContext
	RunTarget("test")
}

func TestRunTargetWithContext(t *testing.T) {
	// Mock cfg for this test
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = Config{
		Targets: map[string]Target{
			"test": {Run: []string{"echo target-test"}},
		},
	}

	err := runTargetWithContext("test", false, false)
	if err != nil {
		t.Errorf("runTargetWithContext() unexpected error: %v", err)
	}

	// Test nonexistent target
	err = runTargetWithContext("nonexistent", false, false)
	if err == nil {
		t.Errorf("runTargetWithContext() expected error for nonexistent target")
	}
}

func TestListTargets(t *testing.T) {
	// Mock cfg for this test
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = Config{
		Targets: map[string]Target{
			"build": {
				Run: []string{"go build"},
			},
			"test": {
				Run: []string{"go test"},
			},
		},
	}

	tests := []struct {
		name   string
		format string
	}{
		{"Table format", "table"},
		{"JSON format", "json"},
		{"YAML format", "yaml"},
		{"Default format", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := listTargets(tt.format)
			if err != nil {
				t.Errorf("listTargets() unexpected error: %v", err)
			}
		})
	}
}

func TestListTargetsTable(t *testing.T) {
	// Mock cfg for this test
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = Config{
		Targets: map[string]Target{
			"build": {
				Run: []string{"go build"},
			},
			"test": {
				Run: []string{"go test"},
			},
		},
	}

	err := listTargetsTable()
	if err != nil {
		t.Errorf("listTargetsTable() unexpected error: %v", err)
	}

	// Test empty targets
	cfg.Targets = map[string]Target{}
	err = listTargetsTable()
	if err != nil {
		t.Errorf("listTargetsTable() unexpected error with empty targets: %v", err)
	}
}

func TestListTargetsJSON(t *testing.T) {
	// Mock cfg for this test
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = Config{
		Targets: map[string]Target{
			"build": {
				Run: []string{"go build"},
			},
		},
	}

	err := listTargetsJSON()
	if err != nil {
		t.Errorf("listTargetsJSON() unexpected error: %v", err)
	}

	// Test empty targets
	cfg.Targets = map[string]Target{}
	err = listTargetsJSON()
	if err != nil {
		t.Errorf("listTargetsJSON() unexpected error with empty targets: %v", err)
	}
}

func TestListTargetsYAML(t *testing.T) {
	// Mock cfg for this test
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = Config{
		Targets: map[string]Target{
			"build": {
				Run: []string{"go build"},
			},
		},
	}

	err := listTargetsYAML()
	if err != nil {
		t.Errorf("listTargetsYAML() unexpected error: %v", err)
	}

	// Test empty targets
	cfg.Targets = map[string]Target{}
	err = listTargetsYAML()
	if err != nil {
		t.Errorf("listTargetsYAML() unexpected error with empty targets: %v", err)
	}
}
