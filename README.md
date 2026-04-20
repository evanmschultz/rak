# rak

**rak** is a fast project-sizing CLI for counting code. The name is short for Swedish *räkna* ("to count").

It's `wc` for a whole project — walk directories, split counts by file type and subdirectory, respect `.gitignore`, and print in human-readable, JSON, or tree form. Works in pipes too.

## Status

**Scaffold.** Project layout and rules are in place; real functionality lands incrementally. See [`main/PLAN.md`](./PLAN.md) for the drop tree and [`main/CLAUDE.md`](./CLAUDE.md) for how this project is built.

## Install

```sh
go install github.com/evanmschultz/rak@latest
```

Available once the first release drop lands.

## Usage (Aspirational)

```sh
rak                            # count the current directory
rak ./cmd                      # count a specific path
rak --depth 2 ./               # limit traversal depth
rak --lang go,rs ./            # only Go and Rust files
rak --json ./internal          # machine-readable output (auto-selected when piped)
rak --tokens ./internal        # add tiktoken-based token estimates
curl https://… | rak           # wc-parity counts on a stream
cat main.go | rak --lang go    # code-aware counts on a stream
```

Run `rak --help` for the full flag list once it's built.

## Default Behavior (Target)

- If no path is given, rak uses the current working directory.
- Walking a directory automatically groups counts by file type and by subdirectory.
- `.gitignore` is respected by default; override with `--no-gitignore`, narrow with `--include`/`--exclude`, or use `--tracked-only` for a strict git-tracked-files view.
- Binary files are skipped by default (count separately with `--binary`).
- On a TTY, output is human-readable through `laslig`. When piped, rak emits JSON by default.

## Name

From Swedish *räkna* — "to count". The CLI is named `rak` for brevity.

## License

MIT. See [LICENSE](./LICENSE).
