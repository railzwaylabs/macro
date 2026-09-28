package main

import (
	"fmt"
	"os"

	"github.com/railzwaylabs/macro/internal/command"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd := command.New(os.Stdout, os.Stderr, command.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
