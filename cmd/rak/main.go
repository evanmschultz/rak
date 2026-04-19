// Package main implements the rak CLI entry point.
package main

import (
	"context"
	"os"
	"syscall"

	"github.com/charmbracelet/fang"
)

func main() {
	if err := fang.Execute(
		context.Background(),
		newRootCmd(),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	); err != nil {
		os.Exit(1)
	}
}
