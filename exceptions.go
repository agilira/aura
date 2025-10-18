package main

import (
	"fmt"
	"os"
)

// Exception Numbers
const (
	TARGET_NOT_FOUND int8 = iota + 1
	FILE_NOT_FOUND
	TARGET_ERROR
)

var Exps map[int8]string

// Initialize Exceptions Map
func InitExceptions() {
	Exps = make(map[int8]string, 0)
	Exps[TARGET_NOT_FOUND] = "Target %s Not Found"
	Exps[FILE_NOT_FOUND] = "aura.yaml Not Found In '%s'"
	Exps[TARGET_ERROR] = "TargetError: %s"

}

func RaiseException(exception_number int8, value string, exit bool) {
	fmt.Fprintf(os.Stderr, Exps[exception_number]+"\n", value)
	if !exit {
		os.Exit(int(exception_number))
	}
}

func SkipError(local bool) bool {
	return local || cfg.ContinueOnError
}
