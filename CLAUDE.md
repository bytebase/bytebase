# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build/Test Commands
- Backend: `go build -ldflags "-w -s" -p=16 -o ./.air/bytebase ./backend/bin/server/main.go`
- Start backend: `PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug`
- Run single Go test: `go test -v github.com/bytebase/bytebase/backend/tests -run TestFunctionName`
- Frontend install: `pnpm --dir frontend i`
- Frontend dev: `pnpm --dir frontend dev`
- Frontend lint: `pnpm --dir frontend lint`
- Frontend type check: `pnpm --dir frontend type-check`
- Frontend test: `pnpm --dir frontend test`
- Proto lint: `cd proto && buf lint`
- Go lint command: `g ./path/to/package/...` - This custom alias runs linting tools for Go code

## Code Style
- **General**: Follow Google style guides for all languages
- **Conciseness**: Write clean, minimal code; fewer lines is better
- **Comments**: Only include comments that are essential to understanding functionality or convey non-obvious information
- **Go**: Use standard Go error handling with detailed error messages
- **Frontend**: Follow TypeScript style with strict type checking
- **Naming**: Use American English, avoid plurals like "xxxList"
- **Git**: Follow conventional commit format
- **Imports**: Use organized imports (sorted by the import path)
- **Formatting**: Use linting/formatting tools before committing
- **Error Handling**: Be explicit but concise about error cases
- **Go Resources**: Always use `defer` for resource cleanup like `rows.Close()` (sqlclosecheck)
- **Go Defer**: Avoid using `defer` inside loops (revive) - use IIFE or scope properly

## Common Go Lint Rules
Always follow these guidelines to avoid common linting errors:

- **File Formatting**: Run `go fmt` on files before committing to ensure proper formatting
- **Unused Parameters**: Prefix unused parameters with underscore (e.g., `func foo(_ *Bar)`)
- **Modern Go Conventions**: Use `any` instead of `interface{}` (since Go 1.18)
- **Confusing Naming**: Avoid similar names that differ only by capitalization
- **Identical Branches**: Don't use if-else branches that contain identical code
- **Unused Functions**: Mark unused functions with `// nolint:unused` comment if needed for future use
- **Function Receivers**: Don't create unnecessary function receivers; use regular functions if receiver is unused
- **Proper Import Ordering**: Maintain correct grouping and ordering of imports
- **Consistency**: Keep function signatures, naming, and patterns consistent with existing code
- **Export Rules**: Only export (capitalize) functions and types that need to be used outside the package