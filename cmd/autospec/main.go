// autospec - Spec-Driven Development Automation
// Author: Ariel Frischer
// Source: https://github.com/ariel-frischer/autospec

package main

import (
	"os"

	"github.com/ariel-frischer/autospec/internal/cli"
	"github.com/ariel-frischer/autospec/internal/shutdown"
)

func main() {
	if err := cli.Execute(); err != nil {
		if code, ok := shutdown.ExitCode(); ok {
			os.Exit(code)
		}
		os.Exit(1)
	}
}
