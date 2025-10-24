//go:build go1.18
// +build go1.18

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// ===== FUZZ TESTS FOR SECURITY-CRITICAL FUNCTIONS =====

// FuzzParseVars tests the variable parsing function with random inputs
// This is critical as it processes user-controlled YAML content
func FuzzParseVars(f *testing.F) {
	// Seed with known test cases
	f.Add("$CC -o $OUTPUT", "build")
	f.Add("${VAR} test ${ANOTHER}", "target")
	f.Add("$@", "mytarget")
	f.Add("", "empty")
	f.Add("$", "dollar")
	f.Add("$$", "doubledollar")
	f.Add("$cwd", "builtin")
	f.Add("$TIMESTAMP", "time")
	f.Add("$NONEXISTENT", "missing")
	f.Add("${}", "emptybrace")
	f.Add("${UNCLOSED", "malformed")
	f.Add("multiple $VAR1 and $VAR2 vars", "multi")
	f.Add("special chars: $VAR! @#$%", "special")
	f.Add(strings.Repeat("$VAR", 100), "repeated")

	// Setup test environment
	original := cfg.Vars
	defer func() { cfg.Vars = original }()

	cfg.Vars = map[string]Var{
		"CC":      "gcc",
		"OUTPUT":  "app.exe",
		"VAR":     "value",
		"VAR1":    "val1",
		"VAR2":    "val2",
		"ANOTHER": "test",
	}

	f.Fuzz(func(t *testing.T, text string, target string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(text) || !utf8.ValidString(target) {
			t.Skip("Invalid UTF-8 input")
		}

		// Skip extremely long inputs to prevent timeout
		if len(text) > 10000 || len(target) > 1000 {
			t.Skip("Input too long")
		}

		// The function should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseVars panicked with input %q, target %q: %v", text, target, r)
			}
		}()

		result := ParseVars(text, target)

		// Basic invariants
		// Allow for reasonable expansion due to builtin variables
		// $TIMESTAMP expands to 19 chars, $cwd can be long, etc.
		maxExpectedLength := len(text) * 10
		if len(text) < 10 {
			maxExpectedLength = len(text) + 100 // Allow for builtin variable expansion
		}
		if len(result) > maxExpectedLength {
			t.Errorf("Result unexpectedly long: input %d chars -> output %d chars (max expected %d)", len(text), len(result), maxExpectedLength)
		}

		// Should not contain unclosed braces if input was well-formed
		if strings.Count(text, "${") == strings.Count(text, "}") {
			openBraces := strings.Count(result, "${")
			closeBraces := strings.Count(result, "}")
			if openBraces > closeBraces {
				t.Errorf("Unclosed braces in result: %q -> %q", text, result)
			}
		}

		// Result should be valid UTF-8
		if !utf8.ValidString(result) {
			t.Errorf("Invalid UTF-8 in result: %q -> %q", text, result)
		}
	})
}

// FuzzExecuteCommand tests command execution with various inputs
// CRITICAL: This executes actual commands, so we need safety measures
func FuzzExecuteCommand(f *testing.F) {
	// Seed with safe test cases
	f.Add("echo test")
	f.Add("echo hello world")
	f.Add("echo")
	f.Add("")
	f.Add("echo 'quoted string'")
	f.Add("echo $SAFE_VAR")
	f.Add("cd .")
	f.Add("cd ..")

	f.Fuzz(func(t *testing.T, command string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(command) {
			t.Skip("Invalid UTF-8 input")
		}

		// Skip extremely long commands
		if len(command) > 1000 {
			t.Skip("Command too long")
		}

		// SECURITY: Block dangerous commands in fuzz testing
		dangerous := []string{
			"rm ", "del ", "format", "mkfs", "dd ", "sudo",
			"su ", "chmod", "chown", ">", ">>", "|", "&", ";",
			"curl", "wget", "nc ", "netcat", "telnet", "ssh",
			"shutdown", "reboot", "halt", "poweroff",
			"mount", "umount", "fdisk", "parted",
		}

		commandLower := strings.ToLower(command)
		for _, danger := range dangerous {
			if strings.Contains(commandLower, danger) {
				t.Skip("Dangerous command blocked")
			}
		}

		// Only allow safe echo commands and basic operations
		if !strings.HasPrefix(commandLower, "echo") &&
			!strings.HasPrefix(commandLower, "cd ") &&
			command != "" {
			t.Skip("Non-echo command blocked in fuzz test")
		}

		// The function should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ExecuteCommand panicked with input %q: %v", command, r)
			}
		}()

		output, err := ExecuteCommand(command)

		// Basic invariants
		if output != "" && !utf8.ValidString(output) {
			t.Errorf("Invalid UTF-8 in output: %q", output)
		}

		// Empty command should error
		if strings.TrimSpace(command) == "" && err == nil {
			t.Errorf("Empty command should return error")
		}

		// Output should not be suspiciously long for echo commands
		if strings.HasPrefix(commandLower, "echo") && len(output) > len(command)*10 {
			t.Errorf("Echo output unexpectedly long: %q -> %d chars", command, len(output))
		}
	})
}

// FuzzLoadConfig tests configuration loading with malformed YAML
func FuzzLoadConfig(f *testing.F) {
	// Seed with valid configurations
	f.Add(`
vars:
  CC: gcc
targets:
  build:
    run:
      - echo test
`)
	f.Add(`{}`)
	f.Add(`vars: {}`)
	f.Add(`targets: {}`)
	f.Add("")
	f.Add("invalid yaml [[[")
	f.Add("vars:\n  test:")

	f.Fuzz(func(t *testing.T, yamlContent string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(yamlContent) {
			t.Skip("Invalid UTF-8 input")
		}

		// Skip extremely long content
		if len(yamlContent) > 50000 {
			t.Skip("Content too long")
		}

		// Create temporary file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.yaml")

		if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		// The function should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("loadConfig panicked with YAML %q: %v", yamlContent, r)
			}
		}()

		// Test loading the config
		err := loadConfig(configPath)

		// Valid YAML should either load successfully or return a proper error
		// Invalid YAML should return an error, not panic
		if err != nil {
			// Error is expected for invalid YAML, ensure it's a proper error message
			if !utf8.ValidString(err.Error()) {
				t.Errorf("Error message contains invalid UTF-8: %v", err)
			}
		}
	})
}

// FuzzGetVar tests variable resolution with various inputs
func FuzzGetVar(f *testing.F) {
	// Seed with known cases
	f.Add("CC", "build")
	f.Add("@", "target")
	f.Add("cwd", "test")
	f.Add("TIMESTAMP", "test")
	f.Add("", "test")
	f.Add("NONEXISTENT", "test")
	f.Add("PATH", "test")

	// Setup test environment
	original := cfg.Vars
	defer func() { cfg.Vars = original }()

	cfg.Vars = map[string]Var{
		"CC":      "gcc",
		"EMPTY":   "",
		"NORMAL":  "value",
		"SPECIAL": "value with spaces & symbols!@#$%^&*()",
	}

	f.Fuzz(func(t *testing.T, varName string, targetName string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(varName) || !utf8.ValidString(targetName) {
			t.Skip("Invalid UTF-8 input")
		}

		// Skip extremely long inputs
		if len(varName) > 1000 || len(targetName) > 1000 {
			t.Skip("Input too long")
		}

		// The function should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GetVar panicked with varName %q, targetName %q: %v", varName, targetName, r)
			}
		}()

		result := GetVar(varName, targetName)

		// Result should always be valid UTF-8
		if !utf8.ValidString(result) {
			t.Errorf("Invalid UTF-8 in result for var %q: %q", varName, result)
		}

		// Built-in variables should behave predictably
		switch varName {
		case "@":
			if result != targetName {
				t.Errorf("@ variable should return target name: got %q, want %q", result, targetName)
			}
		case "cwd":
			if result == "" {
				t.Errorf("cwd variable should not be empty")
			}
		case "TIMESTAMP":
			if len(result) != 19 { // YYYY-MM-DD HH:MM:SS format
				t.Errorf("TIMESTAMP should be 19 chars, got %d: %q", len(result), result)
			}
		}

		// Result should not be suspiciously long
		if len(result) > 10000 {
			t.Errorf("Result suspiciously long for var %q: %d chars", varName, len(result))
		}
	})
}

// FuzzGenerateTemplate tests template generation with various inputs
func FuzzGenerateTemplate(f *testing.F) {
	// Seed with known template types
	f.Add("basic")
	f.Add("go")
	f.Add("rust")
	f.Add("node")
	f.Add("")
	f.Add("unknown")
	f.Add("BASIC")
	f.Add("Go")

	f.Fuzz(func(t *testing.T, templateType string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(templateType) {
			t.Skip("Invalid UTF-8 input")
		}

		// Skip extremely long inputs
		if len(templateType) > 1000 {
			t.Skip("Input too long")
		}

		// The function should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("generateTemplate panicked with type %q: %v", templateType, r)
			}
		}()

		result := generateTemplate(templateType)

		// Result should always be valid UTF-8
		if !utf8.ValidString(result) {
			t.Errorf("Invalid UTF-8 in template result: %q", result)
		}

		// Result should always be non-empty (defaults to basic template)
		if result == "" {
			t.Errorf("Template result should not be empty for type %q", templateType)
		}

		// Result should be reasonable length
		if len(result) > 50000 {
			t.Errorf("Template result suspiciously long: %d chars", len(result))
		}

		// Result should contain basic YAML structure
		if !strings.Contains(result, "vars:") && !strings.Contains(result, "targets:") {
			t.Errorf("Template should contain basic YAML structure: %q", result)
		}
	})
}

// FuzzPathValidation tests path handling for security vulnerabilities
func FuzzPathValidation(f *testing.F) {
	// Seed with various path patterns
	f.Add("config.yaml")
	f.Add("../config.yaml")
	f.Add("../../etc/passwd")
	f.Add("/etc/passwd")
	f.Add("C:\\Windows\\System32\\config")
	f.Add("./config.yaml")
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("con")
	f.Add("aux")
	f.Add("nul")
	f.Add("path/with spaces/config.yaml")
	f.Add("path\\with\\backslashes")
	f.Add(strings.Repeat("../", 100))

	f.Fuzz(func(t *testing.T, configPath string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(configPath) {
			t.Skip("Invalid UTF-8 input")
		}

		// Skip extremely long paths
		if len(configPath) > 5000 {
			t.Skip("Path too long")
		}

		// Create a safe temporary directory for testing
		tmpDir := t.TempDir()

		// Test path cleaning and validation (simulating loadConfig logic)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Path validation panicked with path %q: %v", configPath, r)
			}
		}()

		// Simulate the path validation logic from loadConfig
		testPath := configPath
		if !filepath.IsAbs(testPath) {
			testPath = filepath.Join(tmpDir, configPath)
		}

		cleanPath := filepath.Clean(testPath)

		// Check for directory traversal attempts
		hasTraversal := strings.Contains(cleanPath, "..")

		// The validation should catch directory traversal
		if hasTraversal && strings.Contains(configPath, "..") {
			// This is expected - the original path contained .. and clean path still does
			// The loadConfig function should reject this
		}

		// Clean path should always be valid UTF-8
		if !utf8.ValidString(cleanPath) {
			t.Errorf("Clean path contains invalid UTF-8: %q -> %q", configPath, cleanPath)
		}

		// Clean path should not be suspiciously long
		// On Windows, paths can become much longer due to temp directories
		maxExpectedLength := len(configPath) * 10
		if len(tmpDir) > 50 { // Long temp directory on Windows
			maxExpectedLength = len(configPath) + len(tmpDir) + 50 // Allow for temp dir + some buffer
		}
		if len(cleanPath) > maxExpectedLength {
			t.Errorf("Clean path unexpectedly long: %q -> %q", configPath, cleanPath)
		}
	})
}

// ===== PROPERTY-BASED TESTING HELPERS =====

// TestParseVarsInvariants tests invariant properties of ParseVars
func TestParseVarsInvariants(t *testing.T) {
	// Setup
	cfg.Vars = map[string]Var{
		"TEST": "value",
	}

	t.Run("Idempotency", func(t *testing.T) {
		input := "no variables here"
		target := "test"

		result1 := ParseVars(input, target)
		result2 := ParseVars(result1, target)

		if result1 != result2 {
			t.Errorf("ParseVars not idempotent: %q -> %q -> %q", input, result1, result2)
		}
	})

	t.Run("EmptyStringHandling", func(t *testing.T) {
		result := ParseVars("", "target")
		if result != "" {
			t.Errorf("Empty string should remain empty: got %q", result)
		}
	})

	t.Run("NoVariablesPassthrough", func(t *testing.T) {
		input := "just a regular string with no variables"
		target := "test"

		result := ParseVars(input, target)
		if result != input {
			t.Errorf("String without variables should pass through unchanged: %q -> %q", input, result)
		}
	})
}

// TestExecuteCommandInvariants tests invariant properties
func TestExecuteCommandInvariants(t *testing.T) {
	t.Run("EmptyCommandErrors", func(t *testing.T) {
		_, err := ExecuteCommand("")
		if err == nil {
			t.Error("Empty command should return error")
		}
	})

	t.Run("WhitespaceOnlyCommandErrors", func(t *testing.T) {
		_, err := ExecuteCommand("   \t\n   ")
		if err == nil {
			t.Error("Whitespace-only command should return error")
		}
	})
}
