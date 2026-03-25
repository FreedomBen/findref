# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is findref?

`findref` (alias `fr`) is a recursive code search CLI tool written in Go. It finds strings or matches regular expressions across directories of text files, similar to `git grep` but repository-agnostic with better formatting and filtering. Uses RE2 regex syntax.

## Agent Instructions

- `README.md`, `ARCHIVES.md`, and `install.sh` are generated from their corresponding `.erb` templates (`README.md.erb`, `ARCHIVES.md.erb`, `install.sh.erb`) via Rake. Always edit the `.erb` source files, never the rendered output directly.
- Always update README.md.erb and other documentation when making changes that impact the existing documentation
- Always write tests for new or changed functionality
- When fixing broken tests after a change, be thoughtful about whether the test needs to change or if the application is actually broken

## Build & Test Commands

```bash
# Build
go build

# Run tests
go test ./...

# Run a single test
go test -run TestFunctionName

# Install locally
go install
```

## Architecture

Single-package Go application (`package main`). No Makefile — uses standard `go build`.

| File | Purpose |
|------|---------|
| `findref.go` | Entry point (`func main`), flag parsing, usage text, core search orchestration |
| `settings.go` | `Settings` struct, default exclude lists, filtering rules |
| `config.go` | YAML config loading (`~/.findref.yaml`, `.findref.yaml`, `~/.config/findref/config.yaml`) |
| `match.go` | `Match` struct, colored grep-style output formatting |
| `file_list.go` | `FileToScan` struct for directory traversal |
| `stats.go` | Thread-safe statistics tracking (files/lines scanned, matches found) |
| `colors.go` | ANSI color constants |
| `mcp.go` | Model Context Protocol server mode for AI agent integration |

## Key Design Details

- **Smart-case matching**: Searches are case-insensitive by default unless the pattern contains uppercase characters. Override with `-c` (ignore-case) or `-m` (match-case).
- **Default excludes**: 24+ patterns (`.git`, `node_modules`, `vendor`, lock files, etc.) defined in `settings.go`. Users can extend via config files or `-e`/`-E` flags.
- **Binary file detection**: Files containing null bytes are skipped.
- **MCP mode**: `--mcp` flag starts a Model Context Protocol server (implemented in `mcp.go`).
- **Config precedence**: CLI flags override YAML config. Config searched in order: `.findref.yaml`, `~/.findref.yaml`, `~/.config/findref/config.yaml`.

## Dependencies

Single external dependency: `gopkg.in/yaml.v3` for YAML config parsing. Go 1.25+.

## Version

Version constant is in `findref.go` (`const Version`). Update it there for releases.
