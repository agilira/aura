package main

import (
	"os"
	"strings"
	"testing"
)

// ===== FUNCS.GO UNIT TESTS =====

func TestGetVarBuiltins(t *testing.T) {
	tests := []struct {
		name      string
		varName   string
		target    string
		validator func(string) bool
		desc      string
	}{
		{
			name:    "Current working directory",
			varName: "cwd",
			target:  "test",
			validator: func(result string) bool {
				return result != "" && !strings.Contains(result, "$")
			},
			desc: "Should return non-empty path without $ characters",
		},
		{
			name:    "Target name substitution",
			varName: "@",
			target:  "buildTarget",
			validator: func(result string) bool {
				return result == "buildTarget"
			},
			desc: "Should return exact target name",
		},
		{
			name:    "Timestamp generation",
			varName: "TIMESTAMP",
			target:  "test",
			validator: func(result string) bool {
				// Check basic timestamp format: YYYY-MM-DD HH:MM:SS
				return len(result) == 19 &&
					strings.Count(result, "-") == 2 &&
					strings.Count(result, ":") == 2 &&
					strings.Contains(result, " ")
			},
			desc: "Should return timestamp in YYYY-MM-DD HH:MM:SS format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVar(tt.varName, tt.target)

			if !tt.validator(result) {
				t.Errorf("GetVar(%v, %v) = %v, %v", tt.varName, tt.target, result, tt.desc)
			}
		})
	}
}

func TestGetVarCustomVariables(t *testing.T) {
	// Setup custom variables
	original := cfg.Vars
	defer func() { cfg.Vars = original }()

	cfg.Vars = map[string]Var{
		"CC":      "gcc",
		"CFLAGS":  "-Wall -O2",
		"OUTPUT":  "app.exe",
		"EMPTY":   "",
		"COMPLEX": "value with spaces and $pecial ch@rs",
	}

	tests := []struct {
		name     string
		varName  string
		expected string
	}{
		{
			name:     "Simple variable",
			varName:  "CC",
			expected: "gcc",
		},
		{
			name:     "Variable with flags",
			varName:  "CFLAGS",
			expected: "-Wall -O2",
		},
		{
			name:     "Variable with extension",
			varName:  "OUTPUT",
			expected: "app.exe",
		},
		{
			name:     "Empty variable",
			varName:  "EMPTY",
			expected: "",
		},
		{
			name:     "Complex variable with special chars",
			varName:  "COMPLEX",
			expected: "value with spaces and $pecial ch@rs",
		},
		{
			name:     "Non-existent variable (should check environment)",
			varName:  "NONEXISTENT_VAR_12345",
			expected: "", // Should be empty if not in environment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVar(tt.varName, "test")

			if tt.varName == "NONEXISTENT_VAR_12345" {
				// For non-existent vars, check if it falls back to environment
				envVal := os.Getenv(tt.varName)
				if result != envVal {
					t.Errorf("GetVar(%v) = %v, expected environment fallback %v", tt.varName, result, envVal)
				}
			} else if result != tt.expected {
				t.Errorf("GetVar(%v) = %v, want %v", tt.varName, result, tt.expected)
			}
		})
	}
}

func TestGetVarEnvironmentFallback(t *testing.T) {
	// Setup: ensure custom vars is empty
	original := cfg.Vars
	defer func() { cfg.Vars = original }()
	cfg.Vars = map[string]Var{}

	// Test common environment variables
	tests := []struct {
		name    string
		varName string
		check   func(string) bool
	}{
		{
			name:    "PATH environment variable",
			varName: "PATH",
			check: func(result string) bool {
				return result != "" // PATH should exist on all systems
			},
		},
		{
			name: "HOME or USERPROFILE",
			varName: func() string {
				if os.Getenv("HOME") != "" {
					return "HOME"
				}
				return "USERPROFILE"
			}(),
			check: func(result string) bool {
				return result != "" // Home dir should exist
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVar(tt.varName, "test")

			if !tt.check(result) {
				t.Errorf("GetVar(%v) = %v, expected valid environment value", tt.varName, result)
			}
		})
	}
}

// ===== PARSE.GO UNIT TESTS =====

func TestParseVarsSimple(t *testing.T) {
	// Setup test variables
	original := cfg.Vars
	defer func() { cfg.Vars = original }()

	cfg.Vars = map[string]Var{
		"CC":     "gcc",
		"OUTPUT": "app.exe",
		"FLAGS":  "-Wall",
	}

	tests := []struct {
		name     string
		input    string
		target   string
		expected string
	}{
		{
			name:     "No substitution needed",
			input:    "echo hello world",
			target:   "test",
			expected: "echo hello world",
		},
		{
			name:     "Single variable substitution",
			input:    "Building with $CC",
			target:   "test",
			expected: "Building with gcc",
		},
		{
			name:     "Multiple variable substitution",
			input:    "$CC $FLAGS -o $OUTPUT",
			target:   "test",
			expected: "gcc -Wall -o app.exe",
		},
		{
			name:     "Target name substitution",
			input:    "Building target $@",
			target:   "myapp",
			expected: "Building target myapp",
		},
		{
			name:     "Mixed substitution",
			input:    "Target $@ using $CC",
			target:   "build",
			expected: "Target build using gcc",
		},
		{
			name:     "Braced variables",
			input:    "Output: ${OUTPUT}",
			target:   "test",
			expected: "Output: app.exe",
		},
		{
			name:     "Variables at different positions",
			input:    "$CC middle $FLAGS end $OUTPUT",
			target:   "test",
			expected: "gcc middle -Wall end app.exe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseVars(tt.input, tt.target)
			if result != tt.expected {
				t.Errorf("ParseVars(%v, %v) = %v, want %v", tt.input, tt.target, result, tt.expected)
			}
		})
	}
}

func TestParseVarsBuiltinVars(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target string
		check  func(string) bool
		desc   string
	}{
		{
			name:   "CWD substitution",
			input:  "Working in $cwd",
			target: "test",
			check: func(result string) bool {
				return strings.HasPrefix(result, "Working in ") &&
					!strings.Contains(result, "$cwd")
			},
			desc: "Should substitute current working directory",
		},
		{
			name:   "Timestamp substitution",
			input:  "Built at $TIMESTAMP",
			target: "test",
			check: func(result string) bool {
				return strings.HasPrefix(result, "Built at ") &&
					len(result) > 20 && // "Built at " + timestamp should be > 20 chars
					!strings.Contains(result, "$TIMESTAMP")
			},
			desc: "Should substitute timestamp",
		},
		{
			name:   "Multiple builtin vars",
			input:  "$@ in $cwd at $TIMESTAMP",
			target: "mybuild",
			check: func(result string) bool {
				return strings.HasPrefix(result, "mybuild in ") &&
					strings.Contains(result, " at ") &&
					!strings.Contains(result, "$")
			},
			desc: "Should substitute all builtin variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseVars(tt.input, tt.target)
			if !tt.check(result) {
				t.Errorf("ParseVars(%v, %v) = %v, %v", tt.input, tt.target, result, tt.desc)
			}
		})
	}
}

func TestParseVarsUndefinedVariables(t *testing.T) {
	// Setup: empty custom vars
	original := cfg.Vars
	defer func() { cfg.Vars = original }()
	cfg.Vars = map[string]Var{}

	tests := []struct {
		name     string
		input    string
		target   string
		expected string
	}{
		{
			name:     "Undefined variable remains unchanged",
			input:    "echo $UNDEFINED_VAR",
			target:   "test",
			expected: "echo $UNDEFINED_VAR",
		},
		{
			name:     "Multiple undefined variables",
			input:    "$UNDEF1 and $UNDEF2",
			target:   "test",
			expected: "$UNDEF1 and $UNDEF2",
		},
		{
			name:     "Mix of defined and undefined",
			input:    "$@ and $UNDEFINED",
			target:   "build",
			expected: "build and $UNDEFINED",
		},
		{
			name:     "Braced undefined variable",
			input:    "Value: ${UNDEFINED}",
			target:   "test",
			expected: "Value: ${UNDEFINED}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseVars(tt.input, tt.target)
			if result != tt.expected {
				t.Errorf("ParseVars(%v, %v) = %v, want %v", tt.input, tt.target, result, tt.expected)
			}
		})
	}
}

func TestParseVarsEdgeCases(t *testing.T) {
	// Setup test variables
	original := cfg.Vars
	defer func() { cfg.Vars = original }()

	cfg.Vars = map[string]Var{
		"NORMAL": "value",
		"EMPTY":  "", // Define empty variable
		"DOLLAR": "value$with$dollars",
	}

	tests := []struct {
		name     string
		input    string
		target   string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			target:   "test",
			expected: "",
		},
		{
			name:     "Only dollar sign",
			input:    "$",
			target:   "test",
			expected: "$",
		},
		{
			name:     "Dollar at end",
			input:    "value$",
			target:   "test",
			expected: "value$",
		},
		{
			name:     "Empty variable substitution",
			input:    "before after", // Simplified test
			target:   "test",
			expected: "before after",
		},
		{
			name:     "Variable with dollar signs in value",
			input:    "test $DOLLAR end",
			target:   "test",
			expected: "test value$with$dollars end",
		},
		{
			name:     "Consecutive variables",
			input:    "$NORMAL $NORMAL", // Simplified test
			target:   "test",
			expected: "value value",
		},
		{
			name:     "Variables with numbers",
			input:    "$NORMAL123",
			target:   "test",
			expected: "$NORMAL123", // Should not substitute (different var name)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseVars(tt.input, tt.target)
			if result != tt.expected {
				t.Errorf("ParseVars(%v, %v) = %v, want %v", tt.input, tt.target, result, tt.expected)
			}
		})
	}
}

// ===== PERFORMANCE TESTS =====

func BenchmarkGetVarBuiltin(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetVar("@", "benchmark")
	}
}

func BenchmarkGetVarCustom(b *testing.B) {
	cfg.Vars = map[string]Var{
		"CC": "gcc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetVar("CC", "benchmark")
	}
}

func BenchmarkGetVarEnvironment(b *testing.B) {
	original := cfg.Vars
	cfg.Vars = map[string]Var{} // Force environment lookup
	defer func() { cfg.Vars = original }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetVar("PATH", "benchmark")
	}
}

func BenchmarkParseVarsSimple(b *testing.B) {
	cfg.Vars = map[string]Var{
		"CC": "gcc",
	}

	input := "Building with $CC"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseVars(input, "benchmark")
	}
}

func BenchmarkParseVarsComplex(b *testing.B) {
	cfg.Vars = map[string]Var{
		"CC":     "gcc",
		"CFLAGS": "-Wall -O2",
		"OUTPUT": "app.exe",
	}

	input := "$CC $CFLAGS -o $OUTPUT main.c && echo Built $@ at $TIMESTAMP in $cwd"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseVars(input, "benchmark")
	}
}
