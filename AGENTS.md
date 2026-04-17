This file provides guidance to AI coding assistants when working with code in this repository.

## Project Architecture

- Database schema is defined in `./backend/migrator/migration/LATEST.sql`
- Database migration files are in `./backend/migrator/<<version>>/`
  - `TestLatestVersion` in `./backend/migrator/migrator_test.go` needs update after new migration files are added
  - `./backend/migrator/migration/LATEST.sql` should be updated for DDL migrations
- Files in `./backend/store` are mappings to the database tables

## Development Workflow

**ALWAYS follow these steps after making code changes:**

### Go Code Changes

1. **Format** — Run `gofmt -w` on modified files
2. **Lint** — Run `golangci-lint run --allow-parallel-runners` to catch issues
   - **Important**: Run golangci-lint repeatedly until there are no issues (the linter has a max-issues limit and may not show all issues in a single run)
3. **Auto-fix** — Use `golangci-lint run --fix --allow-parallel-runners` to fix issues automatically
4. **Test** — Run relevant tests before committing
5. **Build** — `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
6. **Tidy** — After upgrading Go dependencies, run `go mod tidy` to clean up `go.mod` and `go.sum`

### Frontend Code Changes

1. **Fix** — Run `pnpm --dir frontend fix` to auto-fix ESLint + Biome issues (format, lint, organize imports)
2. **Check** — Run `pnpm --dir frontend check` to validate without modifying files (for CI)
3. **Type check** — Run `pnpm --dir frontend type-check`
4. **Test** — Run `pnpm --dir frontend test`

### Proto Changes

1. **Format** — Run `buf format -w proto`
2. **Lint** — Run `buf lint proto`
3. **Generate** — Run `cd proto && buf generate`

## Build/Test Commands

### Backend

```bash
# Build
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go

# Start backend
PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug

# Run single test
go test -v -count=1 github.com/bytebase/bytebase/backend/path/to/tests -run ^TestFunctionName$

# Run multiple tests
go test -v -count=1 github.com/bytebase/bytebase/backend/path/to/tests -run ^(TestFunctionName|TestFunctionNameTwo)$

# Lint
golangci-lint run --allow-parallel-runners
```

### Frontend

```bash
# Install dependencies
pnpm --dir frontend i

# Dev server
pnpm --dir frontend dev

# Fix (ESLint + Biome: format, lint, organize imports)
pnpm --dir frontend fix

# Check (validate without modifying, for CI)
pnpm --dir frontend check

# Type check
pnpm --dir frontend type-check

# Test
pnpm --dir frontend test
```

### Proto

```bash
# Format
buf format -w proto

# Lint
buf lint proto

# Generate
cd proto && buf generate
```

### Database

```bash
# Connect to Postgres
psql -U bbdev bbdev
```

## Code Style

### General

- Follow Google style guides for all languages
  - [Go](https://google.github.io/styleguide/go/)
  - [TypeScript](https://google.github.io/styleguide/tsguide.html) and [JavaScript](https://google.github.io/styleguide/jsguide.html)
- Write clean, minimal code; fewer lines is better
- Prioritize simplicity for effective and maintainable software
- Only include comments that are essential to understanding functionality or convey non-obvious information

### Go

- Use standard Go error handling with detailed error messages
- Always use `defer` for resource cleanup like `rows.Close()` (sqlclosecheck)
- Avoid using `defer` inside loops (revive) — use IIFE or scope properly

### API and Proto

- Follow [AIPs](https://google.aip.dev/general)
- When AIP and the proto guide conflict, AIP takes precedence
- Use `HELLO` for enum names, not `TYPE_HELLO`

### Frontend

- Follow TypeScript style with strict type checking
- **i18n**: All user-facing display text in the UI must be defined and maintained in locale files under `./frontend/src/locales/` using the i18n internationalization system. Do not hardcode any display strings directly in the source code
  - **No Empty Objects**: Do not add empty JSON objects (e.g., `"key": {}`) to locale files. Remove any empty objects you encounter
- **Button Spacing**: Use `gap-x-2` for ALL button groups (modals, drawers, toolbars, inline actions). Never use `space-x` for buttons. See `./frontend/.claude/BUTTON_SPACING_STANDARDIZATION.md` for full guidelines

### React (New Code)

The frontend is migrating from Vue to React. **All new UI code should be written in React.**

**Stack**: React + [Base UI](https://base-ui.com/) (`@base-ui/react`) + Tailwind CSS v4 + shadcn-style component patterns

**Component patterns**:
- Build UI components in the shadcn style — `class-variance-authority` (cva) for variant props, `clsx`/`tailwind-merge` for class merging
- Wrap Base UI primitives (Button, Tabs, Input, etc.) with styled variants in `./frontend/src/react/components/ui/`
- Use `useTranslation()` from `react-i18next` for i18n
- Use CSS custom properties (`--color-accent`, `--color-error`, `--color-control-border`, etc.) for theme tokens shared with the Vue layer

**Tailwind CSS v4**:
- CSS-first config in `./frontend/src/assets/css/tailwind.css` — no JS config file
- Custom utilities use `@utility`, design tokens use `@theme`
- Default border color is `currentcolor` (compat shim in `tailwind.css` preserves v3 behavior)

**Accessing Vue state from React**:
- `useVueState(getter)` — React hook that subscribes to Vue reactive state (Pinia stores, refs, computed) via `useSyncExternalStore`. React components access stores directly — no bridge layer needed
- React pages are self-contained: import Pinia stores, `router`, utility functions directly
- React `.tsx` is compiled by esbuild (`react-tsx-transform` Vite plugin) and type-checked separately via `tsconfig.react.json` (excluded from vue-tsc)

## Naming

- Use American English
- Avoid plurals like "xxxList" for simplicity and to prevent singular/plural ambiguity stemming from poor design

## Composite Primary Keys

Several tables use composite primary keys (e.g., `(project, id)`). Check
`backend/migrator/migration/LATEST.sql` for the full list — any table with a
multi-column PRIMARY KEY.

When writing or modifying queries on these tables:
- Every WHERE, JOIN, USING, DELETE, and UPDATE predicate must include ALL primary key
  columns — never filter by `id` alone
- When adding a new store method touching a composite-PK table, add a corresponding
  `TestCollision_*` test in `backend/tests/`. The existing `setupCollidingProjects`
  fixture and `assertProjectUnchanged` helper cover `plan`, `issue`, `task`, `task_run`,
  and `plan_check_run`. For tables not in that set (e.g., `plan_webhook_delivery`,
  `task_run_log`, `db_group`, `release`), write table-specific seed and assertion
  helpers — or extend the shared helper first
- Collision tests use `ctl.server.StoreForTest()` — a test-only accessor. Run with:
  `go test -v -count=1 ./backend/tests/ -run "^(TestClaim|TestCollision)" -timeout 5m`
- For the full pre-PR checklist, see `docs/pre-pr-checklist.md`

### Imports

- Use organized imports (sorted by the import path)

### Formatting

- Use linting/formatting tools before committing

### Error Handling

- Be explicit but concise about error cases

## Pull Request Guidelines

- **Code Review** — Follow [Google's Code Review Guideline](https://google.github.io/eng-practices/)
- **Author Responsibility** — Authors are responsible for driving discussions, resolving comments, and promptly merging pull requests
- **Description** — Clearly describe what the PR changes and why
- **Testing** — Include information about how the changes were tested
- **SonarCloud** — When creating or updating a PR, update `.sonarcloud.properties` to reflect the latest file structure. Use `sonar.exclusions` for generated code, build artifacts, and dependencies (directory paths only). Use `sonar.test.inclusions` for test file patterns (wildcards like `**/*_test.go`). Use `sonar.cpd.exclusions` to skip copy-paste detection on test files

### Breaking Change Check (MANDATORY before `gh pr create`)

**This is a required step in the PR creation workflow. Run `git diff main...HEAD` and check every item below BEFORE writing the `gh pr create` command. This step comes after pushing the branch and before creating the PR.**

1. **API breaking changes** — removed/renamed endpoints, changed request/response formats, removed/renamed query parameters
2. **Database schema breaking changes** — dropped columns/tables, non-backward-compatible migrations
3. **Proto breaking changes** — removed/renamed fields, changed field numbers/types, removed RPCs
4. **Configuration breaking changes** — removed flags, changed default behavior, renamed environment variables
5. **Behavior changes** — changed default values, altered existing workflows, modified permission/access control logic
6. **Webhook/event changes** — renamed/removed events, changed payload formats
7. **UI workflow changes** — redesigned user-facing flows that change how users perform existing tasks (e.g., merging steps, splitting pages, changing navigation)
8. **Composite-PK migration** — adding a composite PK to an existing table, or changing the PK columns of an existing table (not: new queries — those are bugs; not: new tables with composite PKs — those are additive)

**If ANY of the above apply:**
1. Add `--label breaking` to the `gh pr create` command
2. Include a `## Breaking Changes` section in the PR description summarizing what changed and what users need to be aware of

Do NOT skip this check. Do NOT assume changes are non-breaking without reviewing the diff.

## Common Go Lint Rules

Always follow these guidelines to avoid common linting errors:

- **Unused Parameters** — Prefix unused parameters with underscore (e.g., `func foo(_ *Bar)`)
- **Modern Go Conventions** — Use `any` instead of `interface{}` (since Go 1.18)
- **Confusing Naming** — Avoid similar names that differ only by capitalization
- **Identical Branches** — Don't use if-else branches that contain identical code
- **Unused Functions** — Mark unused functions with `// nolint:unused` comment if needed for future use
- **Function Receivers** — Don't create unnecessary function receivers; use regular functions if receiver is unused
- **Proper Import Ordering** — Maintain correct grouping and ordering of imports
- **Consistency** — Keep function signatures, naming, and patterns consistent with existing code
- **Export Rules** — Only export (capitalize) functions and types that need to be used outside the package
- **Linting Command** — Always run `golangci-lint run --allow-parallel-runners` without appending filenames to avoid "function not defined" errors (functions are defined in other files within the package)

## Miscellaneous

- The database JSONB columns store JSON marshalled by `protojson.Marshal` in Go code. `protojson.Marshal` produces camelCased keys rather than the snake_case keys defined in the proto files. e.g. `task_run` becomes `taskRun`
- When modifying multiple files, run file modification tasks in parallel whenever possible, instead of processing them sequentially
