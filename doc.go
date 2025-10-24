/*
Package main implements Aura, a fast and powerful build tool with modern CLI capabilities.

Aura is a build automation tool that processes YAML configuration files to execute
build targets with support for variable substitution, dependency management, and
advanced CLI operations.

# Core Features

Variable Substitution:
Aura supports variable substitution in commands using $VAR or ${VAR} syntax.
Built-in variables include $cwd (current directory), $@ (target name), and
$TIMESTAMP (current time).

Target Dependencies:
Build targets can declare dependencies on other targets or files, ensuring
proper build order and incremental builds.

Template System:
Initialize new projects with built-in templates for Go, Rust, Node.js, and
basic C/C++ projects using the init command.

File Watching:
Continuously monitor files for changes and automatically rebuild targets
using the watch command with configurable polling intervals.

Build Caching:
Advanced caching system powered by Orpheus storage backend with metrics
and configurable storage providers.

# CLI Commands

Build Operations:
  - build: Execute build targets with parallel jobs and force rebuild options
  - list: Display available targets in table, JSON, or YAML format
  - clean: Remove build artifacts and cache files
  - validate: Validate configuration file syntax and structure

Project Management:
  - init: Initialize new project with language-specific templates
  - watch: Monitor files and rebuild on changes

Cache Management:
  - cache clear: Clear build cache
  - cache info: Show cache information and statistics
  - cache list: List cached items

# Configuration

Aura uses YAML configuration files (default: aura.yaml) with the following structure:

	vars:
	  CC: "gcc"
	  CFLAGS: "-Wall -O2"

	targets:
	  build:
	    run:
	      - "$CC $CFLAGS -o app main.c"
	    deps:
	      - "main.c"

	prologue:
	  run:
	    - "echo Starting build"

	epilogue:
	  run:
	    - "echo Build completed"

# Security

The tool implements comprehensive security measures:
- Input validation and sanitization
- Path traversal prevention
- Command injection protection
- Fuzz testing for all input processing functions

# Usage Examples

Basic build execution:

	aura build -t compile,test

Initialize new Go project:

	aura init --template go

Watch for file changes:

	aura watch -t build -i 2s

List available targets:

	aura list --format json

# Dependencies

Aura leverages the Orpheus framework for advanced CLI capabilities,
error handling, and storage management. Core dependencies include:
- github.com/agilira/orpheus: Modern CLI framework
- gopkg.in/yaml.v3: YAML parsing and processing

# Compatibility

Requires Go 1.18+ for fuzzing support and modern language features.
Supports Windows, Linux, and macOS with platform-specific optimizations.

# Build System Integration

Cross-platform build system support:
- Windows: PowerShell scripts (Makefile.ps1)
- Unix/Linux: Traditional Makefiles
- CI/CD: GitHub Actions workflows with automated testing

For detailed documentation, see the README.md file and docs/ directory.
*/
package main
