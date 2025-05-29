# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Architecture
- The database schema is defined in `./backend/migrator/migration/LATEST.sql`
- The database migration files are in `./backend/migrator/<<version>>/`
- Files in `./backend/store` are mappings to the database tables.

## Build/Test Commands
- Backend: `go build -ldflags "-w -s" -p=16 -o ./.air/bytebase ./backend/bin/server/main.go`
- Start backend: `PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug`
- Run single Go test: `go test -v github.com/bytebase/bytebase/backend/tests -run TestFunctionName`
- Frontend install: `pnpm --dir frontend i`
- Frontend dev: `pnpm --dir frontend dev`
- Frontend lint: `pnpm --dir frontend lint`
- Frontend lint fix: `pnpm --dir frontend lint --fix` - Please run this before committing to ensure code quality if changes are made in the frontend code.
- Frontend type check: `pnpm --dir frontend type-check`
- Frontend test: `pnpm --dir frontend test`
- Proto lint: `cd proto && buf lint`
- Proto generate: `cd proto && buf generate`
- Go lint command: `golangci-lint run --allow-parallel-runners` - This custom alias lints Go code, ensuring no errors or warnings are present after updates.
- Connect to Postgres: `psql -U bbdev bbdev`

## Code Style
- **General**: Follow Google style guides for all languages
- **Conciseness**: Write clean, minimal code; fewer lines is better
- **Comments**: Only include comments that are essential to understanding functionality or convey non-obvious information
- **Go**: Use standard Go error handling with detailed error messages
- **API and Proto**: Follow AIPs in https://google.aip.dev/general
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

- **File Formatting**: Run `gofmt -w` on files before committing to ensure proper formatting
- **Unused Parameters**: Prefix unused parameters with underscore (e.g., `func foo(_ *Bar)`)
- **Modern Go Conventions**: Use `any` instead of `interface{}` (since Go 1.18)
- **Confusing Naming**: Avoid similar names that differ only by capitalization
- **Identical Branches**: Don't use if-else branches that contain identical code
- **Unused Functions**: Mark unused functions with `// nolint:unused` comment if needed for future use
- **Function Receivers**: Don't create unnecessary function receivers; use regular functions if receiver is unused
- **Proper Import Ordering**: Maintain correct grouping and ordering of imports
- **Consistency**: Keep function signatures, naming, and patterns consistent with existing code
- **Export Rules**: Only export (capitalize) functions and types that need to be used outside the package

## Individual Preferences
- @~/.claude/bytebase-instructions.md