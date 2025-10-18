package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func ParseVars(text string, targetname string) string {

	// $var or ${var} or $@
	r := regexp.MustCompile(`\$\w+|\$\{[^}]+\}|\$@`)
	matches := r.FindAllString(text, -1)

	for _, m := range matches {
		varname := strings.TrimPrefix(m, "$")
		varname = strings.Trim(varname, "{}")

		val := GetVar("$"+varname, targetname)
		if val == "" {
			fmt.Fprintf(os.Stderr, "[warn] undefined variable %s in target %s\n", m, targetname)
			continue
		}

		text = strings.Replace(text, m, val, 1)
	}

	return text
}
