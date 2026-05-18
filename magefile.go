//go:build mage

// Package main hosts the rak build automation targets driven by mage.
//
// Discover targets with mage -l. The canonical targets mirror the table in
// main/CLAUDE.md § "Build Verification"; any drift between that table and
// this file is a bug. Never invoke raw go build, go test, go vet, gofumpt, or
// golangci-lint — always route through the mage target. If a target is
// broken, fix it here; do not bypass.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// coverageFloor is the minimum acceptable line coverage percentage for the
// ./internal/... scope. mage coverage enforces this gate; mage ci invokes it.
const coverageFloor = 70.0

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
// Lint, then Test, then Coverage (70% floor on ./internal/...). Any failure
// fails CI.
func CI() error {
	mg.SerialDeps(gofumptClean, Lint, Test, Coverage)
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

// AddDep runs `go get <module>` to add or update a Go module dependency.
// It does NOT run `go mod tidy`; callers handle tidy separately if needed.
// Use this from Drop 2 onward whenever a unit introduces a new Go dep.
func AddDep(module string) error {
	if err := sh.RunV("go", "get", module); err != nil {
		return fmt.Errorf("mage addDep %s: %w", module, err)
	}
	return nil
}

// UpdateDeps bumps all module deps to latest minor/patch within the existing
// major version. Equivalent to `go get -u all && go mod tidy`. Use sparingly
// — direct deps should use AddDep for surgical updates. This target is safe to
// run when you want a broad indirect-dep refresh (e.g. before a release cut).
func UpdateDeps() error {
	if err := sh.RunV("go", "get", "-u", "all"); err != nil {
		return fmt.Errorf("mage updateDeps: go get -u all: %w", err)
	}
	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("mage updateDeps: go mod tidy: %w", err)
	}
	return nil
}

// Run executes `go run ./cmd/rak` for local smoke testing from source.
//
// Two invocation patterns are supported:
//
//   - `mage run -- <args>` — preferred for positional args (paths). Mage's
//     CLI may emit "Unknown target specified: --" stderr noise and exit 2
//     when <args> contains flag-prefixed tokens (e.g. --version, --help),
//     even though rak itself ran successfully. Use RAK_ARGS for those.
//
//   - `RAK_ARGS="<args>" mage run` — preferred for flag-prefixed args. Mage
//     exits 0 cleanly because no extra tokens follow the target name.
//
// If both are set, the `--` separator path wins.
func Run() error {
	args := []string{"run", "./cmd/rak"}
	// Prefer `--` separator in os.Args (e.g. `mage run -- <args>`).
	sepFound := false
	for i, a := range os.Args {
		if a == "--" {
			args = append(args, os.Args[i+1:]...)
			sepFound = true
			break
		}
	}
	// Fall back to RAK_ARGS env var (e.g. `RAK_ARGS="--version" mage run`).
	// This path avoids the mage CLI exit-2 that occurs when flag-prefixed
	// args are passed after `--` in some mage versions.
	if !sepFound {
		if extra := os.Getenv("RAK_ARGS"); extra != "" {
			args = append(args, strings.Fields(extra)...)
		}
	}
	if err := sh.RunV("go", args...); err != nil {
		return fmt.Errorf("mage run: %w", err)
	}
	return nil
}

// Coverage runs `go test -race -coverpkg=./internal/... -coverprofile=coverage.out ./...`
// then `go tool cover -func=coverage.out`. Enforces a coverageFloor (70.0%)
// on the ./internal/... scope (cmd/rak CLI wiring excluded per decision 22).
// Exits non-zero if the total percentage falls below the floor.
func Coverage() error {
	if err := sh.RunV(
		"go", "test", "-race",
		"-coverpkg=./internal/...",
		"-coverprofile=coverage.out",
		"./...",
	); err != nil {
		return fmt.Errorf("mage coverage: test: %w", err)
	}

	out, err := sh.Output("go", "tool", "cover", "-func=coverage.out")
	if err != nil {
		return fmt.Errorf("mage coverage: cover func: %w", err)
	}

	// Print the full report so the caller can read the per-function breakdown.
	fmt.Println(out)

	// Parse the total: line. Format: "total:\t(statements)\t87.3%"
	pct, err := parseCoverageTotal(out)
	if err != nil {
		return fmt.Errorf("mage coverage: parse total: %w", err)
	}

	fmt.Printf("coverage: %.1f%% (floor: %.1f%%, scope: ./internal/...)\n", pct, coverageFloor)

	if pct < coverageFloor {
		return fmt.Errorf(
			"coverage %.1f%% is below the %.0f%% floor (scope: ./internal/...)",
			pct, coverageFloor,
		)
	}

	return nil
}

// parseCoverageTotal extracts the total coverage percentage from `go tool cover
// -func` output. It expects a line of the form:
//
//	total:	(statements)	87.3%
//
// Returns an error if no such line is found or the percentage cannot be parsed.
func parseCoverageTotal(output string) (float64, error) {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "total:") {
			continue
		}
		fields := strings.Fields(line)
		// fields[0]="total:", fields[1]="(statements)", fields[2]="87.3%"
		if len(fields) < 3 {
			return 0, fmt.Errorf("unexpected total line format: %q", line)
		}
		pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
		pct, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse coverage percentage %q: %w", pctStr, err)
		}
		return pct, nil
	}
	return 0, fmt.Errorf("no total: line found in go tool cover output")
}
