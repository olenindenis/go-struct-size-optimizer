# GO struct_size_optimizer

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/olenindenis/go_struct_size_optimizer.svg)](https://pkg.go.dev/github.com/olenindenis/go_struct_size_optimizer)
[![Go Report Card](https://goreportcard.com/badge/github.com/olenindenis/go-version)](https://goreportcard.com/report/github.com/olenindenis/go_struct_size_optimizer)

A static analysis tool for Go that detects structs with suboptimal field ordering and rewrites them to minimize memory usage through better alignment.

## How it works

The analyzer (`structalign`) inspects every struct in a Go package and simulates the memory layout using the target platform's alignment rules. If reordering the fields would reduce the struct size, it reports a diagnostic with the current and optimized sizes plus a colored inline diff.

**Optimization strategy:**

1. **Hot fields first** — fields whose names contain `count`, `flag`, `state`, or `status` are placed at the top of the struct (likely to be accessed frequently and benefit from cache locality).
2. **Descending alignment within each group** — fields are sorted by their alignment requirement (largest first) to minimize padding bytes inserted by the compiler.

**Skipped structs:**

- Structs where any field has a struct tag (`json:"..."`, `db:"..."`, etc.) — tag order often has semantic meaning (e.g. serialization order).
- Structs preceded by a `//structalign:ignore` comment.
- Empty structs.

## Installation

```bash
go install github.com/olenindenis/go_struct_size_optimizer/cmd/struct_size_optimizer@latest
```

## Usage

```bash
# Analyze packages in the current directory
struct_size_optimizer

# Analyze a specific directory
struct_size_optimizer -path /path/to/project

# Analyze a specific package pattern within a directory
struct_size_optimizer -path /path/to/project ./pkg/models/...

# Rewrite source files in place (also prints diagnostics)
struct_size_optimizer -w

# Rewrite files in a specific directory
struct_size_optimizer -path /path/to/project -w
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-path` | current directory | Directory to analyze |
| `-w` | false | Rewrite source files in place |

When no package pattern is provided, `./...` is used — all packages under the target directory are analyzed.

## Example

Given this struct:

```go
type Request struct {
    A bool
    B int64
    C bool
}
```

Running the analyzer prints:

```
test.go:2:1: struct can be optimized: 24 -> 16 bytes
Diff:
────────────────────────────────────────────────────────
 1   struct {
 2 - 	A        bool
 3 - 	B        int64
 4 - 	C        bool
     + 	B        int64
     + 	A        bool
     + 	C        bool
 5   }
────────────────────────────────────────────────────────
analysis complete
```

Running with `-w` rewrites the file **and** prints the same diagnostics so you can see what was changed.

**Hot field example:**

```go
// Before — flag (hot) buried after large fields
type Response struct {
    Body []byte
    Code int32
    flag bool
}

// After — hot fields first, then sorted by alignment
type Response struct {
    flag bool
    Code int32
    Body []byte
}
```

## Ignore directive

To opt a struct out of analysis, add a comment on the preceding line:

```go
//structalign:ignore
type LegacyStruct struct {
    A bool
    B int64
    C bool
}
```

## Running tests

```bash
go test ./...
```