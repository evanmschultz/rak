//go:build mage

// Package main hosts the rak build automation targets driven by mage.
//
// Discover targets with mage -l. The nine canonical targets mirror the table
// in main/CLAUDE.md § "Build Verification"; any drift between that table and
// this file is a bug. Never invoke raw go build, go test, go vet, gofumpt, or
// golangci-lint — always route through the mage target. If a target is
// broken, fix it here; do not bypass.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Build compiles every package in the module.
func Build() error {
	if err := sh.RunV("go", "build", "./..."); err != nil {
		return fmt.Errorf("mage build: %w", err)
	}
	return nil
}

// Test runs `go test -race ./...`. The race detector is always on.
func Test() error {
	if err := sh.RunV("go", "test", "-race", "./..."); err != nil {
		return fmt.Errorf("mage test: %w", err)
	}
	return nil
}

// Format runs `gofumpt -l -w .` to auto-format the tree. Idempotent.
func Format() error {
	if err := sh.RunV("gofumpt", "-l", "-w", "."); err != nil {
		return fmt.Errorf("mage format: %w", err)
	}
	return nil
}

// Lint runs `go vet ./...` then `golangci-lint run`. Both must succeed.
func Lint() error {
	if err := sh.RunV("go", "vet", "./..."); err != nil {
		return fmt.Errorf("mage lint: go vet: %w", err)
	}
	if err := sh.RunV("golangci-lint", "run"); err != nil {
		return fmt.Errorf("mage lint: golangci-lint: %w", err)
	}
	return nil
}

// CI is the pre-push gate: asserts `gofumpt -l .` output is empty, then runs
// Lint, then Test. Any failure fails CI.
func CI() error {
	mg.SerialDeps(gofumptClean, Lint, Test)
	return nil
}

// gofumptClean asserts `gofumpt -l .` prints nothing (no files need
// reformatting). Used as the first step of CI.
func gofumptClean() error {
	out, err := sh.Output("gofumpt", "-l", ".")
	if err != nil {
		return fmt.Errorf("mage ci: gofumpt -l: %w", err)
	}
	if strings.TrimSpace(out) != "" {
		return fmt.Errorf("mage ci: gofumpt -l reported unformatted files:\n%s", out)
	}
	return nil
}

// Install is dev-only; agents MUST NOT invoke. Promotes the rak binary to
// $GOBIN for local dogfooding.
func Install() error {
	if err := sh.RunV("go", "install", "./cmd/rak"); err != nil {
		return fmt.Errorf("mage install: %w", err)
	}
	return nil
}

// Run executes `go run ./cmd/rak`. Positional args pass through after `--`,
// e.g. `mage run -- --help`.
func Run() error {
	args := []string{"run", "./cmd/rak"}
	args = append(args, os.Args[1:]...)
	if err := sh.RunV("go", args...); err != nil {
		return fmt.Errorf("mage run: %w", err)
	}
	return nil
}

// Coverage runs `go test -race -coverpkg=./internal/... -coverprofile=coverage.out ./...`
// then `go tool cover -func=coverage.out`. report-only until Drop 9.3 flips
// the 70% floor on as a gate.
func Coverage() error {
	if err := sh.RunV(
		"go", "test", "-race",
		"-coverpkg=./internal/...",
		"-coverprofile=coverage.out",
		"./...",
	); err != nil {
		return fmt.Errorf("mage coverage: test: %w", err)
	}
	if err := sh.RunV("go", "tool", "cover", "-func=coverage.out"); err != nil {
		return fmt.Errorf("mage coverage: cover func: %w", err)
	}
	return nil
}

// PlanCheck will diff main/PLAN.md container titles + states against
// main/drops/*/ directory names and each drop dir's PLAN.md header state,
// failing if drift is detected. Stubbed in Drop 1; real parity logic lands
// later.
func PlanCheck() error {
	// TODO(planCheck): real parity check — stub passes in Drop 1
	return nil
}
