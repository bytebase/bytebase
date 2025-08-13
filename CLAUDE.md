# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Architecture
- The database schema is defined in `./backend/migrator/migration/LATEST.sql`
- The database migration files are in `./backend/migrator/<<version>>/`. `TestLatestVersion` in `./backend/migrator/migrator_test.go` needs update after new migration files are added. `./backend/migrator/migration/LATEST.sql` should be updated for DDL migrations.
- Files in `./backend/store` are mappings to the database tables.

## Development Workflow
**ALWAYS follow these steps after making code changes:**

### After Go Code Changes
1. **Format**: Run `gofmt -w` on modified files
2. **Lint**: Run `golangci-lint run --allow-parallel-runners` to catch issues
   - **Important**: Run golangci-lint repeatedly until there are no issues. The linter has a max-issues limit and may not show all issues in a single run.
3. **Auto-fix**: Use `golangci-lint run --fix --allow-parallel-runners` to fix issues automatically
4. **Test**: Run relevant tests before committing
5. **Build**: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`

### After Frontend Code Changes
1. **Lint**: Run `pnpm --dir frontend lint --fix`
2. **Type check**: Run `pnpm --dir frontend type-check`
3. **Test**: Run `pnpm --dir frontend test`

### After Proto Changes
1. **Format**: Run `buf format -w proto`
2. **Lint**: Run `buf lint proto`
3. **Generate**: Run `cd proto && buf generate`

## Build/Test Commands
- Backend: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
- Start backend: `PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug`
- Run a single Go test: `go test -v -count=1 github.com/bytebase/bytebase/backend/path/to/tests -run ^TestFunctionName$`
- Run two Go tests: `go test -v -count=1 github.com/bytebase/bytebase/backend/path/to/tests -run ^(TestFunctionName|TestFunctionNameTwo)$`
- Frontend install: `pnpm --dir frontend i`
- Frontend dev: `pnpm --dir frontend dev`
- Frontend lint: `pnpm --dir frontend lint`
- Frontend type check: `pnpm --dir frontend type-check`
- Frontend test: `pnpm --dir frontend test`
- Proto format: `buf format -w proto`
- Proto lint: `buf lint proto`
- Proto generate: `cd proto && buf generate`
- Go lint: `golangci-lint run --allow-parallel-runners`
- Connect to Postgres: `psql -U bbdev bbdev`

## Code Style
- **General**: Follow Google style guides for all languages
  - **Go**: https://google.github.io/styleguide/go/
  - **TypeScript**: https://google.github.io/styleguide/tsguide.html and https://google.github.io/styleguide/jsguide.html
- **Conciseness**: Write clean, minimal code; fewer lines is better. Prioritize simplicity for effective and maintainable software.
- **Comments**: Only include comments that are essential to understanding functionality or convey non-obvious information
- **Go**: Use standard Go error handling with detailed error messages
- **API and Proto**: Follow AIPs at https://google.aip.dev/general. When AIP and the proto guide conflict, AIP takes precedence. For example, use HELLO for enum names, not TYPE_HELLO.
- **Frontend**: Follow TypeScript style with strict type checking
  - **i18n**: All user-facing display text in the UI must be defined and maintained in `./frontend/src/locales/en-US.json` using the i18n internationalization system. Do not hardcode any display strings directly in the source code.
- **Naming**: Use American English, avoid plurals like "xxxList" for simplicity and to prevent singular/plural ambiguity stemming from poor design
- **Git**: Follow conventional commit format
- **Imports**: Use organized imports (sorted by the import path)
- **Formatting**: Use linting/formatting tools before committing
- **Error Handling**: Be explicit but concise about error cases
- **Go Resources**: Always use `defer` for resource cleanup like `rows.Close()` (sqlclosecheck)
- **Go Defer**: Avoid using `defer` inside loops (revive) - use IIFE or scope properly

## Pull Request Guidelines
When creating pull requests:
- **Code Review**: Follow [Google's Code Review Guideline](https://google.github.io/eng-practices/)
- **Author Responsibility**: Authors are responsible for driving discussions, resolving comments, and promptly merging pull requests
- **Breaking Changes**: Add the `breaking` label to PRs that contain breaking changes, including:
  - API breaking changes (removed/renamed endpoints, changed request/response formats)
  - Database schema breaking changes that are not backward-compatible
  - Proto message breaking changes (removed/renamed fields, changed field types)
  - Configuration breaking changes (removed flags, changed behavior)
- **Description**: Clearly describe what the PR changes and why
- **Testing**: Include information about how the changes were tested

## Common Go Lint Rules
Always follow these guidelines to avoid common linting errors:

- **Unused Parameters**: Prefix unused parameters with underscore (e.g., `func foo(_ *Bar)`)
- **Modern Go Conventions**: Use `any` instead of `interface{}` (since Go 1.18)
- **Confusing Naming**: Avoid similar names that differ only by capitalization
- **Identical Branches**: Don't use if-else branches that contain identical code
- **Unused Functions**: Mark unused functions with `// nolint:unused` comment if needed for future use
- **Function Receivers**: Don't create unnecessary function receivers; use regular functions if receiver is unused
- **Proper Import Ordering**: Maintain correct grouping and ordering of imports
- **Consistency**: Keep function signatures, naming, and patterns consistent with existing code
- **Export Rules**: Only export (capitalize) functions and types that need to be used outside the package
- **Linting Command**: Always run `golangci-lint run --allow-parallel-runners` without appending filenames to avoid "function not defined" errors (functions are defined in other files within the package)

## Misc

- The database JSONB columns store JSON marshalled by protojson.Marshal in Go code. protojson.Marshal produces camelCased keys rather than the snake_case keys defined in the proto files. e.g. task_run becomes taskRun.
- When modifying multiple files, run file modification tasks in parallel whenever possible, instead of processing them sequentially.

## Individual Preferences

- @~/.claude/bytebase-instructions.md
