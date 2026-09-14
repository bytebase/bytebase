## Design Principles

Bytebase is the standard for database development. Every product and engineering decision serves that goal, built on three principles:

1. **Bring every database under unified control.** Every person and AI agent accesses and changes data through one governed path.
2. **Govern change and access as code.** Reviewed, enforced, and recorded by policy, not by discipline.
3. **Make the safe path the easy path.** Safe by default, simple by design — so no one routes around it.

## Read before working

- Before exploring domain behavior, read [domain guidance](docs/agents/domain.md).
- For frontend work, read [frontend/AGENTS.md](frontend/AGENTS.md); before changing UI, also read the [UX contract](docs/agents/frontend-ux.md).
- Before adding or modifying metadata SQL, pagination, or multi-row transactions anywhere in the repo, read [backend/store/AGENTS.md](backend/store/AGENTS.md). This includes CEL-to-SQL filters and raw reads in tests.
- Before changing metadata/DDL conversions or their golden fixtures, read [backend/plugin/schema/AGENTS.md](backend/plugin/schema/AGENTS.md). It covers what these packages test and the `record` process for regenerating fixtures.
- For issue operations, read [issue-tracker.md](docs/agents/issue-tracker.md). Linear is the tracker; agent-created issues go to team `BOT`. For triage, also read [triage-labels.md](docs/agents/triage-labels.md).
- Before creating a PR, complete [docs/pre-pr-checklist.md](docs/pre-pr-checklist.md).
- `AGENTS.md` files are the instruction source of truth; `CLAUDE.md` files import them.

## Metadata and API conventions

- The metadata schema is `backend/migrator/migration/LATEST.sql`; migrations live in `backend/migrator/migration/<major.minor>/`.
- New migrations require updating `TestLatestVersion` in `backend/migrator/migrator_test.go`; DDL changes also update `LATEST.sql`.
- Metadata JSONB uses `protojson.Marshal`: keys are camelCase (`taskRun`), not proto snake_case (`task_run`).
- Follow Google language style guides and AIPs for API/proto design. AIPs take precedence over the proto guide. Enum values use `HELLO`, not `TYPE_HELLO`.
- Use American English. Avoid collection names ending in `List`.

## Code comments

- Use names and structure to explain what code does. Reserve comments for non-obvious intent, invariants, ordering constraints, external behavior, and public API contracts.
- Default to one or two sentences. Longer comments must explain a constraint a maintainer needs to change the code safely.
- Keep implementation history, rejected approaches, review discussions, and follow-up inventories in PRs or issues. For a workaround, keep only the current limitation and a relevant issue reference beside the code.
- Explain each constraint once, where it is enforced. In tests, let case names and assertions describe behavior; comment only on surprising fixtures or semantics.
- Before handoff, reread added or modified comments. Remove code narration and repeated rationale, and verify the remaining claims against the implementation.

## Test placement

Test API and workflow behavior against PostgreSQL. Engine dialect and DDL fidelity tests belong in omni.

- Test logic in its owning package. Use `backend/api/v1` for service behavior and `backend/store` for metadata queries, collision isolation, and transaction contention. CI runs every test with no `-short`, so never check `testing.Short()` or skip a test to park a known gap; gate a test on an environment variable only when it needs credentials or a server CI cannot provide.
- A backend test boots a Bytebase server only when it needs a background runner, real rollout, or audit trail; these tests live in `backend/tests`. Browser E2E tests use the separate frontend harness.
- Packages needing metadata PostgreSQL use `testcontainer.Main` and `testcontainer.NewMetadataDB`. Target-engine tests use `testcontainer.SharedPgContainer` or its siblings, with `NewPgDatabase` for a database per test. Use these shared fixtures instead of package-owned or per-test containers; a package with its own `TestMain` defers `testcontainer.CloseShared`.
- Prefer pure functions for handler decisions and conversions. When state reads are necessary, define a narrow interface beside the handler and fake it, as `backend/api/mcp` does with `serverStore`; avoid an interface over the entire store. Every fake requires a contract test against the real store too.
- New or modified composite-key methods require collision coverage. Prove colliding keys exist, assert the intended effect in the target scope, and assert the other scope is unchanged. Use the lowest test layer that exercises the behavior. Existing API collision tests remain regression coverage; fixture details are in the pre-PR checklist.

## Verification

During iteration, run focused checks for the changed behavior. Before handoff, run the applicable final gates below. Correct failures before rerunning; repeat when fixes expose further issues, rather than rerunning unchanged failures. Report any unavailable prerequisite or remaining failure explicitly. Inspect formatter/autofix output for unrelated edits.

### Go changes

1. Run `gofmt -w` on modified Go files.
2. Run `GOGC=off GOMEMLIMIT=10GiB golangci-lint run --fix -j 8 --allow-parallel-runners`, then the same command without `--fix` until clean after corrections. Run at repo root without filenames so package context is available. Keep the flags: on a cold cache they take the run from ~730 to ~420 CPU-seconds and hold peak RSS near 10 GB, on a box shared with two CI runners.
3. Run tests for changed packages and affected behavior, including required collision or contention coverage.
4. Build: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`.
5. After dependency changes, run `go mod tidy` and recheck affected code.

For focused tests, quote the regex:

```bash
go test -v -count=1 ./backend/store/ -run '^(TestFunctionName|TestFunctionNameTwo)$'
```

### Frontend changes

Run `pnpm --dir frontend fix`, then `pnpm --dir frontend test` — the single gate CI runs. To iterate, `pnpm --dir frontend vitest run <path>` runs one file; `pnpm --dir frontend run prepare` refreshes generated sources.

For browser verification, read [frontend/tests/e2e/README.md](frontend/tests/e2e/README.md); before writing tests, read its [AGENTS.md](frontend/tests/e2e/AGENTS.md).

### Proto changes

Run `buf format -w proto`, `buf lint proto`, and `(cd proto && buf generate)`. Verify generated Go/frontend changes with their corresponding gates.

### Documentation changes

Check local links, referenced paths/symbols, and shell examples. Code build/test gates do not apply to documentation-only changes.
