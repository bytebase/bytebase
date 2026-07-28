This file provides guidance to AI coding assistants when working with code in this repository.

## Design Principles

Bytebase is the standard for database development. Every product and engineering decision serves that goal, built on three principles:

1. **Bring every database under unified control.** Every person and AI agent accesses and changes data through one governed path.
2. **Govern change and access as code.** Reviewed, enforced, and recorded by policy, not by discipline.
3. **Make the safe path the easy path.** Safe by default, simple by design — so no one routes around it.

## Agent skills

### Issue tracker

Issues are tracked in Linear via Linear MCP when available, falling back to `linctl`; agent-created issues go to Linear team `BOT`; GitHub PRs may link back to Linear issues, but GitHub Issues and PRs are not the triage surface. See `docs/agents/issue-tracker.md`.

### Triage labels

Triage roles use the default canonical labels: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, and `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

This repo uses a single-context domain-doc layout. See `docs/agents/domain.md`.

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

1. **Fix** — Run `pnpm --dir frontend fix` to auto-fix Biome issues (format, lint, organize imports)
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

# Fix (Biome: format, lint, organize imports)
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

### React

The product frontend is built in React. **All product UI code is React** — use the stack and component patterns below. The only Vue runtime is the isolated `pev2` adapter under `frontend/src/apps/explain-visualizer/`.

The canonical frontend ownership map is in `./frontend/AGENTS.md`. In summary:

- `frontend/src/app/` owns bootstrap, layouts, and router infrastructure.
- `frontend/src/routes/` owns route modules and route-local code.
- `frontend/src/modules/` owns reusable application subsystems.
- `frontend/src/components/ui/` owns shared UI primitives; `frontend/src/components/` contains other genuinely shared product components.
- `frontend/src/stores/`, `frontend/src/api/`, `frontend/src/hooks/`, and `frontend/src/lib/` contain cross-route infrastructure. Existing `types/` and `utils/` are compatibility surfaces, not default homes for owner-specific code.
- Do not introduce a generic feature bucket or recreate the migration-era framework namespace.
- Historical frontend migration plans under `docs/superpowers/` preserve the paths that existed when they were written; use `frontend/AGENTS.md`, not those plans, for current placement decisions.

**Stack**: React + [Base UI](https://base-ui.com/) (`@base-ui/react`) + Tailwind CSS v4 + shadcn-style component patterns

**Component patterns**:
- Build UI components in the shadcn style — `class-variance-authority` (cva) for variant props, `clsx`/`tailwind-merge` for class merging
- Wrap Base UI primitives (Button, Tabs, Input, etc.) with styled variants in `./frontend/src/components/ui/`
- Use `useTranslation()` from `react-i18next` for i18n
- Use CSS custom properties (`--color-accent`, `--color-error`, `--color-control-border`, etc.) for theme tokens, defined in `./frontend/src/assets/css/tailwind.css`

**Shared UI primitives**:
- For React UI code, prefer shared components from `./frontend/src/components/ui/` over native HTML controls or ad hoc styled elements
- Before adding or modifying an interactive UI element, first check whether a matching component already exists in `./frontend/src/components/ui/`
- Use shared UI components for common controls such as buttons, inputs, selects, dialogs, dropdowns, tooltips, tabs, checkboxes, radios, switches, tables, and form controls when available
- Do not hand-roll native controls with Tailwind classes when a shared component exists
- Native HTML controls are allowed only when the shared component does not support the required browser behavior, accessibility behavior, or integration pattern
- When touching existing React UI, opportunistically replace nearby native or ad hoc controls with shared UI components if behavior remains equivalent and the scope stays reasonable

**Tailwind CSS v4**:
- CSS-first config in `./frontend/src/assets/css/tailwind.css` — no JS config file
- Custom utilities use `@utility`, design tokens use `@theme`
- Default border color is `currentcolor` (compat shim in `tailwind.css` preserves v3 behavior)

**State & build**:
- React app state lives under `./frontend/src/stores/` — the core slices are in `stores/app/`, consumed via the `useAppStore` hook. Routing helpers live in `./frontend/src/app/router/`
- React `.tsx` is compiled by esbuild (`react-tsx-transform` Vite plugin) and type-checked with `tsc --build` via `pnpm --dir frontend type-check`

## Naming

- Use American English
- Avoid plurals like "xxxList" for simplicity and to prevent singular/plural ambiguity stemming from poor design

## Composite Primary Keys

Several tables use composite primary keys (e.g., `(project, id)`). Check
`backend/migrator/migration/LATEST.sql` for the full list — any table with a
multi-column PRIMARY KEY.

When writing or modifying queries on these tables:
- Every WHERE, JOIN, USING, DELETE, and UPDATE predicate must include every
  project/tenant scope column. Identify rows with either the full primary key or
  a full declared non-partial UNIQUE key that contains the same scope columns;
  verify alternate keys in `LATEST.sql`. Never filter by `id` or another locally
  unique identifier alone
- When adding a new store method touching a composite-PK table, add a corresponding
  `TestCollision_*` test in `backend/tests/`. The existing `setupCollidingProjects`
  fixture and `assertProjectUnchanged` helper cover `plan`, `issue`, `task`, `task_run`,
  and `plan_check_run`. For tables not in that set (e.g., `plan_webhook_delivery`,
  `task_run_log`, `db_group`, `release`), write table-specific seed and assertion
  helpers — or extend the shared helper first
- Collision tests use `setupCollidingProjects` + `fixture.completeRolloutB` for setup
  and `snapshotProject` / `assertProjectUnchanged` for assertions — all going through
  the public gRPC API, no store access. Run with:
  `go test -v -count=1 ./backend/tests/ -run "^(TestClaim|TestCollision)" -timeout 5m`

## Transaction Lock Ordering

Before adding or modifying a transaction that locks multiple rows or tables, follow the canonical [store row-lock ordering](backend/store/README.md#transaction-row-lock-ordering). Lock existing child rows before parents, lock batches in full primary-key order, and treat upserts as existing-row locks. Add the deterministic real-PostgreSQL regression tests required below for new multi-row or multi-table coordination paths.

Row ordering prevents wait-for cycles on existing rows, but it cannot protect a
child row that does not exist yet. `nextProjectID` closes that gap for its callers:
it locks the project and requires the project to be active before allocating an ID,
so creation is rejected when the project is missing or deleted. This is not a
repository-wide purge fence because some writers bypass `nextProjectID`.

Every new or modified writer of purge-managed data must define whether its
lifecycle policy requires an active project or merely an existing project, then
serialize and validate that policy against project deletion. Add deterministic
real-PostgreSQL tests for both lock-acquisition directions. Assert the terminal
outcomes, including that neither direction ends in a foreign-key failure; merely
checking for the absence of SQLSTATE `40P01` is insufficient.

### Imports

- Use organized imports (sorted by the import path)

### Formatting

- Use linting/formatting tools before committing

### Error Handling

- Be explicit but concise about error cases

## Pull Request Guidelines

**Before running `gh pr create`, walk through [`docs/pre-pr-checklist.md`](docs/pre-pr-checklist.md).** It covers the breaking-change review, composite-PK query safety, lint/test gates, and SonarCloud properties — the checks that lint and CI can't catch on their own.

- **Code Review** — Follow [Google's Code Review Guideline](https://google.github.io/eng-practices/)
- **Author Responsibility** — Authors are responsible for driving discussions, resolving comments, and promptly merging pull requests
- **Description** — Clearly describe what the PR changes and why
- **Testing** — Include information about how the changes were tested

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
