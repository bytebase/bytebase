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

Anything that writes SQL against the metadata database is governed by
[`backend/store/AGENTS.md`](backend/store/AGENTS.md): composite-primary-key
predicates, pagination ordering, and transaction row-lock ordering. That means
all of `backend/store/` — including the CEL-to-SQL filter builders, which live
there and not in the service layer — plus the raw metadata reads in the
`backend/tests/` collision tests. Read it before adding or modifying a query, a
paginated list, or a multi-row transaction.

## Testing

Test API and workflow behavior against Postgres only — engine dialects and DDL
fidelity belong in [omni](https://github.com/bytebase/omni), so never write
`TestFooForMySQL` beside `TestFooForPostgreSQL`. A test earns a Bytebase server
boot only if it needs a background runner, a real rollout, or the audit trail —
those live in `backend/tests`, the only package that may start one; everything
else asserts in `backend/api/v1` or `backend/store`. Packages needing a
real database share one container via `TestMain` and give each test its own
`CREATE DATABASE`, never a container per test — a Postgres container costs 4.3s
against a 0.7s server boot.

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
- Keep comments short while preserving necessary information

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

The canonical design foundations and workflow recipes are in
[`docs/agents/frontend-ux.md`](docs/agents/frontend-ux.md). New and directly
modified UI must follow it; unrelated legacy UI is governed by its incremental
baseline and must not gain new violations.

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

### Imports

- Use organized imports (sorted by the import path)

### Formatting

- Use linting/formatting tools before committing

### Error Handling

- Be explicit but concise about error cases

## Naming

- Use American English
- Avoid plurals like "xxxList" for simplicity and to prevent singular/plural ambiguity stemming from poor design

## Pull Request Guidelines

**Before running `gh pr create`, walk through [`docs/pre-pr-checklist.md`](docs/pre-pr-checklist.md).** It covers the breaking-change review, composite-PK query safety, and lint/test gates — the checks that lint and CI can't catch on their own.

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
