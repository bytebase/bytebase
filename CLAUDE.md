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