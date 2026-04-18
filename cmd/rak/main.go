// Package main implements a Fang/Cobra version of a small wc-style CLI.
package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
)

func main() {
	if err := fang.Execute(context.Background(), newRootCmd()); err != nil {
		os.Exit(1)
	}
}
