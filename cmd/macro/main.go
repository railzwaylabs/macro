package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/railzwaylabs/macro/internal/command"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd := command.New(os.Stdout, os.Stderr, resolveBuildInfo(command.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}, debug.ReadBuildInfo))

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveBuildInfo(
	linked command.BuildInfo,
	read func() (*debug.BuildInfo, bool),
) command.BuildInfo {
	info, ok := read()
	if !ok {
		return linked
	}

	if (linked.Version == "" || linked.Version == "dev") &&
		info.Main.Version != "" && info.Main.Version != "(devel)" {
		linked.Version = info.Main.Version
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if linked.Commit == "" || linked.Commit == "unknown" {
				linked.Commit = setting.Value
			}
		case "vcs.time":
			if linked.Date == "" || linked.Date == "unknown" {
				linked.Date = setting.Value
			}
		}
	}

	return linked
}
