package main

import (
	"runtime/debug"
	"testing"

	"github.com/railzwaylabs/macro/internal/command"
)

func TestResolveBuildInfoUsesModuleMetadata(t *testing.T) {
	t.Parallel()

	got := resolveBuildInfo(command.BuildInfo{
		Version: "dev",
		Commit:  "unknown",
		Date:    "unknown",
	}, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v0.1.1"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc123"},
				{Key: "vcs.time", Value: "2026-09-28T10:00:00Z"},
			},
		}, true
	})

	if got.Version != "v0.1.1" {
		t.Fatalf("Version = %q, want v0.1.1", got.Version)
	}
	if got.Commit != "abc123" {
		t.Fatalf("Commit = %q, want abc123", got.Commit)
	}
	if got.Date != "2026-09-28T10:00:00Z" {
		t.Fatalf("Date = %q, want build time", got.Date)
	}
}

func TestResolveBuildInfoKeepsLinkedMetadata(t *testing.T) {
	t.Parallel()

	want := command.BuildInfo{
		Version: "v0.2.0",
		Commit:  "release-commit",
		Date:    "release-date",
	}
	got := resolveBuildInfo(want, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v0.1.1"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "module-commit"},
				{Key: "vcs.time", Value: "module-date"},
			},
		}, true
	})

	if got != want {
		t.Fatalf("BuildInfo = %#v, want %#v", got, want)
	}
}
