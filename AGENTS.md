# Repository Guidelines

## Project Structure & Module Organization
Core Go sources live in the repository root: `findref.go` exposes the CLI entrypoint and version, while `match.go`, `settings.go`, `stats.go`, and `colors.go` encapsulate search logic, configuration, metrics, and terminal styling. Reusable helpers sit in `file_list.go` and `helpers.rb`, the latter supporting the build pipeline. Unit tests currently target the critical helpers inside `findref_test.go`. Release artifacts are produced under `findref-bin/`, and static screenshots live in `images/`. The ERB templates (`README.md.erb`, `ARCHIVES.md.erb`, `install.sh.erb`) feed the documentation and installer scripts through the Rake tasks.

## Build, Test, and Development Commands
- `rake`: preferred entrypoint; rebuilds docs from ERB templates and kicks off the release pipeline (Docker or Podman required for cross-builds).
- `go build`: quick local compile of the CLI for the current platform.
- `GOOS=<target> GOARCH=<arch> go build`: cross-compile when debugging a single target without the full release pass.
- `go test ./...`: run the Go test suite; keep it green before opening a PR.

## Coding Style & Naming Conventions
- Run `gofmt` on every touched `.go` file; the project relies on standard Go formatting (tabs for indentation, grouped imports). Favor small, composable functions and keep exported identifiers in PascalCase with accompanying doc comments when user-facing. Internal helpers should use camelCase. Shell and Ruby scripts already ship with two-space indentation—match the existing style when editing them.
- Always update tests when making code changes
- Always update the autocompletion scripts when making changes that affect them

## Testing Guidelines
Extend `go test ./...` coverage with table-driven cases that mirror real CLI usage. Name new tests with the `TestXxx` convention and keep fixtures inline unless they are shared across multiple cases. When adding behavior tied to flags or file filters, assert both positive and negative paths to protect against regressions.

## Commit & Pull Request Guidelines
Follow the existing imperative tone in Git history (`Update Go version...`, `Add description...`). Keep commits scoped, reference related issues with `#123` where applicable, and update `findref.go`’s `Version` constant alongside documentation changes for releases. Pull requests should summarize intent, list validation commands (e.g., `go test ./...`), and include screenshots for documentation or UI changes. Tag reviewers on cross-language edits (Go plus shell/Ruby) so both contexts receive attention.

## Release Notes & Packaging
When cutting a release: bump the `Version` constant, run `rake erb` to refresh docs, execute `rake` to rebuild archives, and verify `findref-bin/` before publishing. Remove transient binaries after validation to keep the repository tidy.
