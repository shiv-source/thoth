package main

import (
	"fmt"
	"os"

	"github.com/shiv-source/thoth/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "thoth:", err)
		os.Exit(1)
	}
}
