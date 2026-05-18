// Package main implements the rak CLI entry point.
package main

import (
	"context"
	"os"
	"syscall"

	"github.com/charmbracelet/fang"
)

// version is the canonical release identifier. Build-time -ldflags injection
// via GoReleaser sets this to the tagged version at release time; the fallback
// value "v0.2.0-dev" is used for local builds without ldflags.
var version = "v0.2.0-dev"

func main() {
	if err := fang.Execute(
		context.Background(),
		newRootCmd(),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
		fang.WithVersion(version),
	); err != nil {
		os.Exit(1)
	}
}
