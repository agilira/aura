package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func ExecuteCommand(command string) (string, error) {

	var cmd *exec.Cmd
	var shell string
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
		cmd = exec.Command(shell, "/C", command)
	} else {
		// Linux && MacOsX
		shell = "/bin/bash"
		cmd = exec.Command(shell, "-c", command)
	}

	out, err := cmd.CombinedOutput()

	return string(out), err
}

func ExecuteAll(name string, target *Target) {

	cmds := target.Run
	for _, cmd := range cmds {
		cmd = ParseVars(cmd, name)
		out, err := ExecuteCommand(cmd)
		// if error then (get target on_error || cmd stderr)
		if err != nil {
			exit_status := SkipError(target.ContinueOnError)
			outerr := fmt.Sprintf("in %s -> \n", name)
			if strings.TrimSpace(target.Onerror) == "" {
				outerr += err.Error()
				RaiseException(TARGET_ERROR, outerr, exit_status)
			} else {
				outerr += target.Onerror
				RaiseException(TARGET_ERROR, outerr, exit_status)
			}
		}

		if strings.TrimSpace(out) != "" {
			fmt.Print(out)
		}

	}

}

func (t *Target) RunDeps() {
	deps := t.Deps
	for _, dep := range deps {
		// if dep is file
		if strings.Contains(dep, ".") {

		} else {
			RunTarget(dep)
		}
	}

}

func (c *Config) RunPrologue() {
	c.Prologue.RunDeps()
	ExecuteAll("prologue", &c.Prologue)
}

func (c *Config) RunEpilogue() {
	c.Epilogue.RunDeps()
	ExecuteAll("epilogue", &c.Epilogue)
}

func RunTarget(name string) {

	target := GetTarget(name)

	target.RunDeps()

	exit := SkipError(target.ContinueOnError)

	if target.Run == nil && target.Deps == nil {
		RaiseException(TARGET_NOT_FOUND, name, exit)
	}
	ExecuteAll(name, &target)

}
