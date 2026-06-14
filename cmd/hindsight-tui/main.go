package main

import (
	"os"

	"github.com/tlarevo/hindsight-tui/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
