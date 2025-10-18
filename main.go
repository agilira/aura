package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var AURA_PATH, TARGETS, WORKING_DIR string
var cfg Config

func main() {
	// initialize exceptions
	InitExceptions()

	flag.StringVar(&TARGETS, "t", "", "Targets to run (comma separated)")
	flag.StringVar(&WORKING_DIR, "D", ".", "Working Directory")

	flag.Parse()

	AURA_PATH = filepath.Join(WORKING_DIR, "aura.yaml")

	// check if aura exist
	f, err := os.Open(AURA_PATH)
	if err != nil {
		cd, _ := os.Getwd()
		RaiseException(FILE_NOT_FOUND, cd, true)
	}
	// decode main file
	yaml.NewDecoder(f).Decode(&cfg)

	// load includes
	for _, inc := range cfg.Includes {
		inc_f, err := os.Open(inc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[!] Warning: Cannot Load %s\n", inc)
			continue
		}
		yaml.NewDecoder(inc_f).Decode(&cfg)

	}

	cfg.RunPrologue()

	if TARGETS != "" {
		splitted := strings.Split(TARGETS, ",")
		for _, TARGET := range splitted {
			RunTarget(TARGET)
		}
	}

	cfg.RunEpilogue()
}
