// Package main implements the rak CLI entry point.
package main

import (
	"context"
	"os"
	"syscall"

	"github.com/charmbracelet/fang"
)

// version is the canonical release identifier for v0.1.4. Build-time
// -ldflags injection via GoReleaser is deferred to v0.2; for now the
// string is hardcoded here and referenced by fang.WithVersion so that
// `rak --version` emits it directly.
const version = "v0.1.4"

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
