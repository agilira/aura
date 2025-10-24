# Fuzz Testing Guide for Aura Build Tool

## Overview

This document describes the fuzz testing implementation for the Aura build tool, focusing on security-critical functions that process user input.

## Why Fuzz Testing is Critical

Aura processes:
- YAML configuration files (user-controlled)
- Variable substitution patterns (user-controlled)
- Command execution strings (user-controlled)
- File paths (user-controlled)

These inputs can potentially be exploited if not properly validated.

## Fuzz Tests Implemented

### 1. FuzzParseVars
**Target**: `ParseVars()` function
**Risk**: Variable injection, regex denial of service
**Coverage**:
- Variable substitution patterns
- Malformed variable syntax
- Extremely long variable names
- Special characters in variables
- Nested variable references

### 2. FuzzExecuteCommand
**Target**: `ExecuteCommand()` function  
**Risk**: Command injection, arbitrary code execution
**Safety Measures**:
- Blocks dangerous commands (rm, del, format, etc.)
- Only allows safe echo commands in fuzzing
- Validates UTF-8 encoding
- Limits command length

### 3. FuzzLoadConfig
**Target**: `loadConfig()` function
**Risk**: YAML parsing vulnerabilities, path traversal
**Coverage**:
- Malformed YAML structures
- Extremely nested YAML
- Invalid UTF-8 in configuration
- Large configuration files

### 4. FuzzGetVar
**Target**: `GetVar()` function
**Risk**: Information disclosure, environment variable leakage
**Coverage**:
- Built-in variable handling
- Environment variable fallback
- Variable name edge cases
- Target name validation

### 5. FuzzGenerateTemplate
**Target**: `generateTemplate()` function
**Risk**: Template injection, file generation issues
**Coverage**:
- Unknown template types
- Template type validation
- Output validation

### 6. FuzzPathValidation
**Target**: Path handling in `loadConfig()`
**Risk**: Directory traversal, arbitrary file access
**Coverage**:
- Directory traversal attempts (../)
- Absolute path handling
- Windows-specific paths (CON, AUX, etc.)
- Long path names
- Special characters in paths

## Running Fuzz Tests

### Prerequisites
```bash
go version  # Requires Go 1.18+
```

### Basic Fuzzing
```bash
# Run all fuzz tests for 30 seconds each
go test -fuzz=FuzzParseVars -fuzztime=30s
go test -fuzz=FuzzExecuteCommand -fuzztime=30s
go test -fuzz=FuzzLoadConfig -fuzztime=30s
go test -fuzz=FuzzGetVar -fuzztime=30s
go test -fuzz=FuzzGenerateTemplate -fuzztime=30s
go test -fuzz=FuzzPathValidation -fuzztime=30s
```

### Extended Fuzzing (CI/CD)
```bash
# Run for longer periods to find edge cases
go test -fuzz=FuzzParseVars -fuzztime=5m
go test -fuzz=FuzzLoadConfig -fuzztime=10m
```

### Parallel Fuzzing
```bash
# Run multiple fuzz tests in parallel
go test -fuzz=. -parallel=4 -fuzztime=2m
```

## Security Findings and Mitigations

### 1. Variable Substitution
**Potential Issues**:
- Regex complexity attacks
- Infinite recursion in variable expansion
- Memory exhaustion with large variables

**Mitigations Implemented**:
- Length limits on variable names and values
- UTF-8 validation
- Bounded expansion depth

### 2. Command Execution
**Potential Issues**:
- Command injection via variable substitution
- Shell metacharacter exploitation
- Path traversal in cd commands

**Mitigations Implemented**:
- Input validation before execution
- Safe command construction
- Limited shell feature usage

### 3. Configuration Loading
**Potential Issues**:
- YAML bomb attacks (billion laughs)
- Path traversal via include files
- Memory exhaustion with large configs

**Mitigations Implemented**:
- Path validation and sanitization
- File size limits
- Include depth limits

### 4. Path Handling
**Potential Issues**:
- Directory traversal (../)
- Symbolic link exploitation
- Windows reserved names

**Mitigations Implemented**:
- filepath.Clean() usage
- Directory traversal detection
- Path validation before file operations

## Continuous Fuzzing Integration

### GitHub Actions Example
```yaml
name: Fuzz Testing

on:
  schedule:
    - cron: '0 2 * * *'  # Run daily at 2 AM
  push:
    branches: [main]
  pull_request:

jobs:
  fuzz:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        fuzz-target:
          - FuzzParseVars
          - FuzzExecuteCommand  
          - FuzzLoadConfig
          - FuzzGetVar
          - FuzzGenerateTemplate
          - FuzzPathValidation
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    
    - name: Run fuzz test
      run: |
        go test -fuzz=${{ matrix.fuzz-target }} -fuzztime=5m
    
    - name: Upload crash files
      if: failure()
      uses: actions/upload-artifact@v3
      with:
        name: fuzz-failures-${{ matrix.fuzz-target }}
        path: testdata/fuzz/
```

### Local Development Workflow
```bash
# Quick fuzz check before commit
make fuzz-quick

# Deep fuzz testing before release
make fuzz-deep

# Add to Makefile:
fuzz-quick:
	@echo "Running quick fuzz tests..."
	@go test -fuzz=. -fuzztime=30s

fuzz-deep:
	@echo "Running deep fuzz tests..."
	@go test -fuzz=FuzzParseVars -fuzztime=5m
	@go test -fuzz=FuzzLoadConfig -fuzztime=5m
	@go test -fuzz=FuzzExecuteCommand -fuzztime=3m
	@go test -fuzz=FuzzGetVar -fuzztime=2m
	@go test -fuzz=FuzzGenerateTemplate -fuzztime=1m
	@go test -fuzz=FuzzPathValidation -fuzztime=3m
```

## Crash Analysis

### When Fuzz Tests Find Issues
1. **Examine crash files**: Located in `testdata/fuzz/`
2. **Reproduce manually**: Use the failing input to understand the issue
3. **Fix the vulnerability**: Implement proper validation/sanitization
4. **Add regression test**: Ensure the fix works and doesn't break

### Example Crash Investigation
```bash
# If FuzzParseVars finds a crash
ls testdata/fuzz/FuzzParseVars/

# View the crash-causing input
cat testdata/fuzz/FuzzParseVars/[crash-file]

# Reproduce in debugger
go test -fuzz=FuzzParseVars -fuzztime=1s -fuzzminimizetime=1m
```

## Performance Considerations

### Fuzz Test Performance
- **Memory usage**: Monitor with `go test -memprofile`
- **CPU usage**: Some fuzz targets are CPU-intensive
- **Time limits**: Set appropriate `-fuzztime` for CI/CD

### Optimization Tips
```bash
# Profile fuzz tests
go test -fuzz=FuzzParseVars -cpuprofile=fuzz.prof -fuzztime=1m

# Analyze with pprof
go tool pprof fuzz.prof
```

## Best Practices

### 1. Seed Quality
- Include both valid and invalid inputs
- Cover edge cases and boundary conditions
- Use real-world examples

### 2. Safety First
- Never fuzz with destructive operations
- Isolate fuzz tests from production data
- Use temporary directories and files

### 3. Coverage Focus
- Target user input processing functions
- Prioritize parsing and validation code
- Focus on security-sensitive operations

### 4. Regular Execution
- Run fuzz tests in CI/CD pipeline
- Schedule longer fuzz sessions
- Monitor for regressions

## Security Recommendations

### For Users
1. **Validate configuration files** before use
2. **Use absolute paths** when possible  
3. **Avoid complex variable substitution** patterns
4. **Review generated templates** before use

### For Developers
1. **Always validate user input** before processing
2. **Use safe string operations** (avoid regex complexity)
3. **Implement length limits** on all inputs
4. **Sanitize file paths** before filesystem operations
5. **Log security-relevant events** for monitoring

## Conclusion

Fuzz testing is essential for a build tool like Aura because:
- It processes user-controlled configuration
- It executes commands based on user input
- It handles file system operations
- It's used in security-sensitive environments

The implemented fuzz tests provide comprehensive coverage of attack surfaces and help ensure the tool remains secure against malicious inputs.