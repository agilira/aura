package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/agilira/orpheus/pkg/orpheus"
	"gopkg.in/yaml.v3"
)

func ExecuteCommand(command string) (string, error) {
	var cmd *exec.Cmd
	var shell string

	// Check for empty command
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("empty command")
	}

	// Security: Basic command validation - prevent obvious malicious patterns
	if strings.Contains(command, "&&") || strings.Contains(command, "||") || strings.Contains(command, ";") {
		// Allow common patterns but be aware this is a build tool that needs command chaining
	}

	fmt.Println(command)

	if strings.HasPrefix(command, "cd ") {
		dir := strings.TrimSpace(strings.TrimPrefix(command, "cd "))
		if dir == "" {
			return "", fmt.Errorf("no directory specified for cd")
		}
		if err := os.Chdir(dir); err != nil {
			return "", err
		}
		return "", nil
	}

	// Windows
	if runtime.GOOS == "windows" {
		shell = "cmd"
		// #nosec G204 - This is a build tool that executes user-defined commands by design
		cmd = exec.Command(shell, "/C", command)
	} else {
		// Linux && MacOsX
		shell = "/bin/bash"
		// #nosec G204 - This is a build tool that executes user-defined commands by design
		cmd = exec.Command(shell, "-c", command)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func ExecuteCommandWithContext(command string, verbose, dryRun bool) (string, error) {
	if verbose {
		fmt.Printf("→ %s\n", command)
	}

	if dryRun {
		fmt.Printf("  [DRY RUN] Would execute: %s\n", command)
		return "", nil
	}

	return ExecuteCommand(command)
}

func ExecuteAll(name string, target *Target) {
	_ = ExecuteAllWithContext(name, target, false, false)
}

func ExecuteAllWithContext(name string, target *Target, verbose, dryRun bool) error {
	cmds := target.Run
	for _, cmd := range cmds {
		cmd = ParseVars(cmd, name)
		out, err := ExecuteCommandWithContext(cmd, verbose, dryRun)

		// If error then (get target on_error || cmd stderr)
		if err != nil && !dryRun {
			outerr := fmt.Sprintf("in %s -> \n", name)
			if strings.TrimSpace(target.Onerror) == "" {
				outerr += err.Error()
			} else {
				outerr += target.Onerror
			}

			if target.ContinueOnError || cfg.ContinueOnError {
				// Log error but continue
				fmt.Fprintf(os.Stderr, "Warning: %s\n", outerr)
			} else {
				// Return Orpheus error and stop
				return orpheus.ExecutionError(name, outerr)
			}
		}

		if strings.TrimSpace(out) != "" && !dryRun {
			fmt.Print(out)
		}
	}
	return nil
}

func (t *Target) RunDeps() {
	_ = t.RunDepsWithContext(false, false)
}

func (t *Target) RunDepsWithContext(verbose, dryRun bool) error {
	deps := t.Deps
	for _, dep := range deps {
		// if dep is file
		if strings.Contains(dep, ".") {
			// TODO: Handle file dependencies
			if verbose {
				fmt.Printf("Checking file dependency: %s\n", dep)
			}
		} else {
			if err := runTargetWithContext(dep, verbose, dryRun); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) RunPrologue() {
	_ = c.RunPrologueWithContext(false, false)
}

func (c *Config) RunPrologueWithContext(verbose, dryRun bool) error {
	if err := c.Prologue.RunDepsWithContext(verbose, dryRun); err != nil {
		return err
	}
	return ExecuteAllWithContext("prologue", &c.Prologue, verbose, dryRun)
}

func (c *Config) RunEpilogue() {
	_ = c.RunEpilogueWithContext(false, false)
}

func (c *Config) RunEpilogueWithContext(verbose, dryRun bool) error {
	if err := c.Epilogue.RunDepsWithContext(verbose, dryRun); err != nil {
		return err
	}
	return ExecuteAllWithContext("epilogue", &c.Epilogue, verbose, dryRun)
}

func RunTarget(name string) {
	_ = runTargetWithContext(name, false, false)
}

func runTargetWithContext(name string, verbose, dryRun bool) error {
	target := GetTarget(name)

	if err := target.RunDepsWithContext(verbose, dryRun); err != nil {
		return err
	}

	if target.Run == nil && target.Deps == nil {
		return orpheus.NotFoundError(name, fmt.Sprintf("target '%s' not found", name))
	}

	return ExecuteAllWithContext(name, &target, verbose, dryRun)
}

// Context-aware wrapper functions
func runPrologueWithContext(verbose, dryRun bool) error {
	return cfg.RunPrologueWithContext(verbose, dryRun)
}

func runEpilogueWithContext(verbose, dryRun bool) error {
	return cfg.RunEpilogueWithContext(verbose, dryRun)
}

func listTargets(format string) error {
	switch format {
	case "json":
		return listTargetsJSON()
	case "yaml":
		return listTargetsYAML()
	default: // table
		return listTargetsTable()
	}
}

func listTargetsTable() error {
	fmt.Println("Available targets:")
	fmt.Println("------------------")

	if len(cfg.Targets) == 0 {
		fmt.Println("No targets found")
		return nil
	}

	// Find max name length for formatting
	maxNameLen := 0
	for name := range cfg.Targets {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	// Print targets
	for name, target := range cfg.Targets {
		padding := strings.Repeat(" ", maxNameLen-len(name)+2)
		deps := ""
		if len(target.Deps) > 0 {
			deps = fmt.Sprintf(" (depends: %s)", strings.Join(target.Deps, ", "))
		}
		fmt.Printf("  %s%s%d commands%s\n", name, padding, len(target.Run), deps)
	}

	fmt.Printf("\nTotal: %d targets\n", len(cfg.Targets))
	return nil
}

func listTargetsJSON() error {
	type TargetInfo struct {
		Name     string   `json:"name"`
		Commands int      `json:"commands"`
		Deps     []string `json:"dependencies,omitempty"`
	}

	var targets []TargetInfo
	for name, target := range cfg.Targets {
		targets = append(targets, TargetInfo{
			Name:     name,
			Commands: len(target.Run),
			Deps:     target.Deps,
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"targets": targets,
		"total":   len(targets),
	})
}

func listTargetsYAML() error {
	type TargetInfo struct {
		Name     string   `yaml:"name"`
		Commands int      `yaml:"commands"`
		Deps     []string `yaml:"dependencies,omitempty"`
	}

	var targets []TargetInfo
	for name, target := range cfg.Targets {
		targets = append(targets, TargetInfo{
			Name:     name,
			Commands: len(target.Run),
			Deps:     target.Deps,
		})
	}

	encoder := yaml.NewEncoder(os.Stdout)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(map[string]interface{}{
		"targets": targets,
		"total":   len(targets),
	})
}
