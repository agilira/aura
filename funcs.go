package main

import (
	"os"
	"strings"
	"time"
)

// Get a variable else -> environment variable -> ""
func GetVar(name string, target_name string) string {

	name = strings.Trim(name, "$")
	switch name {
	case "TIMESTAMP":
		return time.Now().Format("2006-01-02 15:04:05")
	case "@":
		return target_name
	case "cwd":
		path, _ := os.Getwd()
		return path
	default:
		ret, exists := cfg.Vars[name]
		if exists {
			return string(ret)
		}
		return os.Getenv(name)
	}

}

// Get target by name
func GetTarget(name string) Target {

	target := cfg.Targets[name]
	return target

}
