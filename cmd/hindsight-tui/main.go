package main

import (
	"os"

	"hindsight-tui/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
