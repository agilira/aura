package main

type Var string

type Target struct {
	Run             []string `yaml:"run"`
	Deps            []string `yaml:"deps"`
	Onerror         string   `yaml:"onerror"`
	ContinueOnError bool     `yaml:"continue_on_error"`
}

type Config struct {
	ContinueOnError bool              `yaml:"continue_on_error"`
	Includes        []string          `yaml:"include"`
	Prologue        Target            `yaml:"prologue"`
	Vars            map[string]Var    `yaml:"vars"`
	Targets         map[string]Target `yaml:"targets"`
	Epilogue        Target            `yaml:"epilogue"`
}
