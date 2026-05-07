# Oracle splitter trailing `;` leak (BYT-9367) — implementation plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `plsql.SplitSQL` so a trailing `;` separated from the statement's parse-tree stop token by whitespace or comments is consumed by the current statement instead of leaking into the next statement's `leadingContent`.

**Architecture:** Replace a single-token SEMI lookahead with a channel-aware loop that walks forward through hidden-channel tokens (whitespace, comments) until it finds either a SEMI (consume) or a default-channel non-SEMI token (bail). One file changed in source (`backend/plugin/parser/plsql/split.go`), one in tests (`backend/plugin/parser/plsql/test-data/test_split.yaml`). No new files, no signature changes.

**Tech Stack:** Go, ANTLR4 (`github.com/antlr4-go/antlr/v4`), `github.com/bytebase/parser/plsql`, YAML test runner (`backend/plugin/parser/base/split_test_runner.go`).

**Spec:** `docs/superpowers/specs/2026-05-07-oracle-splitter-trailing-semicolon-leak-design.md`

---

## File Structure

| File | Role | Change |
|---|---|---|
| `backend/plugin/parser/plsql/split.go` | Oracle splitter implementation | Modify lines 105-113 (the `prevStopTokenIndex` lookahead block) |
| `backend/plugin/parser/plsql/test-data/test_split.yaml` | YAML-driven test fixtures | Append 3 new cases at the end |
| `backend/plugin/parser/plsql/split_test.go` | Test runner harness | No change; existing `TestPLSQLSplitSQL` picks up new YAML cases automatically |

## Test plan recap (from spec §6.1)

Cells of the design gap and where each is covered:

| cell | description | covered by |
|---|---|---|
| (a) | immediate `;` | existing `multiple SELECT statements` (line 1) |
| (b) | whitespace before `;` (BYT-9367 exact) | **new — case `bytx` below** |
| (c) | comment before `;` | **new — case `bytx-comment` below** |
| (d) | multi-newline / mixed whitespace before `;` | **new — case `bytx-newlines` below** |
| (e) | no `;` at end of input | existing `multiple statements with newlines` (line 28) |
| (f) | hidden tokens then default-channel non-SEMI (no `;`) | existing forward-slash cases (lines 55, 79, 120, 150) — `/` is FORWARD_SLASH, default channel, non-SEMI; the loop's bail branch fires on it |
| (g) | `needSemicolon` branch (anonymous block, `;` IS the stop) | existing forward-slash cases |

Three new cases. Cell (f) was originally listed as new in spec §6.1 but is already exercised by the four existing forward-slash test fixtures via the same code path; documenting that here so the implementer doesn't add a redundant case.

---

## Chunk 1: implementation

### Task 1: Confirm starting state and run existing tests

**Files:**
- Read: `backend/plugin/parser/plsql/split.go`
- Read: `backend/plugin/parser/plsql/test-data/test_split.yaml`

- [ ] **Step 1: Verify the buggy code is at the expected location**

```bash
sed -n '105,113p' backend/plugin/parser/plsql/split.go
```

Expected output:
```go
// Set prevStopTokenIndex to the last token we want to "consume" for this statement.
// For statements where the semicolon is a separator (not part of the statement parse tree),
// we need to skip past the semicolon so it's not included in the next statement's leadingContent.
prevStopTokenIndex = stmt.GetStop().GetTokenIndex()
if nextIdx := prevStopTokenIndex + 1; nextIdx < len(tokens.GetAllTokens()) {
    if nextToken := tokens.Get(nextIdx); nextToken.GetTokenType() == parser.PlSqlParserSEMICOLON {
        prevStopTokenIndex = nextIdx
    }
}
```

If the code differs (e.g., line numbers shifted), update the file references in subsequent steps.

- [ ] **Step 2: Run the existing test suite to establish a green baseline**

```bash
go test -count=1 -v ./backend/plugin/parser/plsql/ -run TestPLSQLSplitSQL
```

Expected: all existing subtests pass.

### Task 2: Capture the buggy output to demonstrate the bug

**Files:**
- Modify: `backend/plugin/parser/plsql/test-data/test_split.yaml` (append three new cases)

- [ ] **Step 1: Append three new test cases with placeholder results**

Append to the END of `backend/plugin/parser/plsql/test-data/test_split.yaml`:

```yaml
- description: 'BYT-9367: trailing semicolon with leading space does not leak'
  input: "insert into t values('a',1) ;\n\ninsert into t values('b',2) ;"
  result: []
- description: 'BYT-9367: trailing semicolon with leading inline comment does not leak'
  input: "insert into t values('a',1) /* note */ ;\ninsert into t values('b',2) ;"
  result: []
- description: 'BYT-9367: trailing semicolon with multiple newlines does not leak'
  input: "insert into t values('a',1)\n\n;\n\ninsert into t values('b',2);"
  result: []
```

`result: []` is a placeholder. Hand-computing byte offsets and line/column positions is error-prone (the splitter computes `Range.Start` as the previous stmt's `byteOffsetEnd`, which equals one past the previous stop token — easy to miscompute by ±1). The `-record` mode below auto-fills correct values; we then visually verify the `text` field, which is the load-bearing invariant for this fix.

**Sanity check for the multi-newline case** (input `"insert into t values('a',1)\n\n;\n\ninsert into t values('b',2);"`): ANTLR's PL/SQL grammar might produce 3 unit_statements (stmt 1, an empty stmt for the standalone `;`, then stmt 2) instead of 2. The `-record` step below will surface this — if you see 3 result entries instead of 2 for this case, do one of, in this preference order:
- **Preferred:** Keep the 3-stmt parse and explicitly assert the empty middle stmt has `Empty: true`. This actually exercises cell (d) "multi-newline before `;`" and documents the grammar's behavior. If a future grammar bump changes the parse to 2-stmt, that's a YAML re-record, not a regression — note this in a comment on the YAML case.
- **Fallback only:** Replace the input with a less aggressive whitespace shape that's known to parse as 2 stmts: `"insert into t values('a',1)\t  ;\n\ninsert into t values('b',2);"` (tab + spaces before `;`, no separate-line `;`). Note: this fallback no longer covers the multi-newline cell — it collapses to the same shape as case (b). Acceptable only if option 1 turns out to be infeasible.

- [ ] **Step 2: Run `-record` BEFORE applying the fix to capture buggy output**

```bash
go test -count=1 ./backend/plugin/parser/plsql/ -run TestPLSQLSplitSQL -args -record
```

This populates `result:` for the new cases using the **current (buggy) splitter**.

- [ ] **Step 3: Inspect the captured YAML and confirm the bug is present**

```bash
git diff backend/plugin/parser/plsql/test-data/test_split.yaml
```

Verify, for each new case, that stmt 2's `text` contains the `;` leak:

| case | expected stmt 2 text (BUGGY pre-fix) | what proves the bug |
|---|---|---|
| BYT-9367 leading space | `" ;\n\ninsert into t values('b',2)"` | starts with ` ;` (space + semicolon) |
| leading inline comment | `" /* note */ ;\ninsert into t values('b',2)"` | starts with ` /* note */ ;` |
| multiple newlines | `"\n\n;\n\ninsert into t values('b',2)"` (two newlines, semicolon, two newlines) — assuming 2-stmt parse; if 3-stmt parse, the middle empty stmt's text contains the `;` and stmt 3's text starts with `"\n\n"` | contains a leading `;` somewhere it shouldn't be |

If you do NOT see a `;` leak in stmt 2's text for any of the three cases, the test inputs are wrong (e.g., wrong escaping in YAML) — fix the inputs before continuing. The whole point of Task 2 is to capture a snapshot that demonstrates the bug.

- [ ] **Step 4: Do not commit the buggy snapshot**

The buggy `result:` values will be overwritten by `-record` after the fix in Task 4. Do not stage or commit the YAML in its current state.

### Task 3: Apply the fix

**Files:**
- Modify: `backend/plugin/parser/plsql/split.go:108-113`

- [ ] **Step 1: Replace the single-token SEMI lookahead with a channel-aware loop**

In `backend/plugin/parser/plsql/split.go`, replace lines 108-113:

```go
				prevStopTokenIndex = stmt.GetStop().GetTokenIndex()
				if nextIdx := prevStopTokenIndex + 1; nextIdx < len(tokens.GetAllTokens()) {
					if nextToken := tokens.Get(nextIdx); nextToken.GetTokenType() == parser.PlSqlParserSEMICOLON {
						prevStopTokenIndex = nextIdx
					}
				}
```

with:

```go
				// prevStopTokenIndex anchors where the next statement's leadingContent starts.
				// Walk forward through hidden-channel tokens (whitespace, comments) to find a
				// trailing ';' belonging to this statement. Bail on the first default-channel
				// non-';' token — that's the start of the next statement.
				prevStopTokenIndex = stmt.GetStop().GetTokenIndex()
				allTokens := tokens.GetAllTokens()
				for nextIdx := prevStopTokenIndex + 1; nextIdx < len(allTokens); nextIdx++ {
					next := allTokens[nextIdx]
					if next.GetTokenType() == parser.PlSqlParserSEMICOLON {
						prevStopTokenIndex = nextIdx
						break
					}
					if next.GetChannel() == antlr.TokenDefaultChannel {
						break
					}
				}
```

Notes:
- `allTokens` is hoisted out of the loop because `tokens.GetAllTokens()` allocates on each call in some ANTLR Go runtimes. Cheap micro-cleanup (advisory item from spec review).
- The `antlr` import is already present at the top of the file (`split.go:4`) — no new import needed.
- The `parser.PlSqlParserSEMICOLON` symbol is unchanged from the previous code.

- [ ] **Step 2: Format**

```bash
gofmt -w backend/plugin/parser/plsql/split.go
```

- [ ] **Step 3: Re-record YAML to capture corrected output**

```bash
go test -count=1 ./backend/plugin/parser/plsql/ -run TestPLSQLSplitSQL -args -record
```

This overwrites the buggy snapshot from Task 2 with the post-fix output for all test cases (new and existing).

### Task 4: Verify the fix and confirm no regression

**Files:**
- Read/diff: `backend/plugin/parser/plsql/test-data/test_split.yaml`

- [ ] **Step 1: Diff the YAML and verify the `text` invariants for new cases**

```bash
git diff backend/plugin/parser/plsql/test-data/test_split.yaml
```

Manually verify, for each of the three new cases, that stmt 2's `text` is now CLEAN (no `;` leak):

| case | expected stmt 2 text (CORRECT post-fix) |
|---|---|
| BYT-9367 leading space | `"\n\ninsert into t values('b',2)"` (starts with two newlines, no `;`, no leading space) |
| leading inline comment | `"\ninsert into t values('b',2)"` (starts with one newline, no `;`, no leading comment, no leading space) |
| multiple newlines | starts with `"\n\n"` (or similar pure-whitespace prefix), absolutely no leading `;` |

For all three: stmt 1 `text` should be `"insert into t values('a',1)"` (27 characters, no leading or trailing whitespace, no `;`).

If any case still shows a leading `;` in stmt 2, the fix is incomplete — return to Task 3 and inspect.

- [ ] **Step 2: Diff the YAML for existing cases to confirm no regression**

The same `git diff` should show only ADDITIONS (the three new cases). **Expected: zero existing-case `text` field changes** — the spec audit (§7) and cell-coverage table (plan top) both show no existing fixture exercises a `<stmt> <hidden> ; <stmt>` shape that would change under the fix. If you see ANY existing-case `text` change, that's a flag — investigate:
- If the change is a `;` leak being removed (e.g., a leading ` ;` disappearing from a stmt): the original fixture was wrong; the new output is correct.
- If the change is anything else: stop and investigate before continuing.

- [ ] **Step 3: Run the test suite in normal (non-record) mode**

```bash
go test -count=1 -v ./backend/plugin/parser/plsql/ -run TestPLSQLSplitSQL
```

Expected: all subtests pass, including the three new BYT-9367 subtests.

### Task 5: Lint and build

- [ ] **Step 1: Lint**

```bash
golangci-lint run --allow-parallel-runners ./backend/plugin/parser/plsql/...
```

Run repeatedly until no issues remain (per `AGENTS.md` — golangci-lint has a max-issues limit). Use `--fix` to auto-fix where possible:

```bash
golangci-lint run --fix --allow-parallel-runners ./backend/plugin/parser/plsql/...
```

Expected: clean.

- [ ] **Step 2: Build the full backend**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: builds successfully.

- [ ] **Step 3: Run the broader splitter tests across engines as a regression sweep**

```bash
go test -count=1 ./backend/plugin/parser/...
```

Expected: pass. The audit (spec §7) confirms no other engine shares the bug pattern, so this should be a no-op safety check, but worth running.

### Task 6: Commit

- [ ] **Step 1: Stage the source and test changes**

```bash
git add backend/plugin/parser/plsql/split.go backend/plugin/parser/plsql/test-data/test_split.yaml
```

- [ ] **Step 2: Commit**

```bash
git commit -m "$(cat <<'EOF'
fix(oracle): trailing ';' leak when whitespace/comment separates stop and ';'

BYT-9367. The plsql splitter's "advance past trailing ';'" lookahead only
inspected the token immediately after the parse-tree stop, so any
hidden-channel token (whitespace, comment) between the stop and the ';'
prevented consumption. The ';' then leaked into the next statement's
leadingContent and Oracle rejected it with ORA-00900 at position 1.

Replace the single-token lookahead with a channel-aware loop that skips
hidden-channel tokens until it finds either ';' (consume) or a
default-channel non-';' token (bail). Backward-compatible for stmt;
inputs (loop's first iteration matches). Grammar-stable: predicate is
"channel != DEFAULT", not an enumerated list of skippable token types.

Tests: 3 new YAML fixtures cover the BYT-9367 exact input, comment
before ';', and multi-newline before ';'. Existing forward-slash tests
already exercise the bail-on-default-channel branch.

Spec: docs/superpowers/specs/2026-05-07-oracle-splitter-trailing-semicolon-leak-design.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 3: Verify clean status**

```bash
git status
```

Expected: working tree clean (or only the spec/plan docs from earlier commits remaining).

---

## Out of scope (for reference)

These were discussed during brainstorming and intentionally excluded — do not add them in this plan:

- **Defensive `leadingContent` rebuild.** Constructing `leadingContent` from default-channel tokens only, or stripping a leading `;\s*` defensively. Mentioned as long-term hardening in the original Linear analysis. The channel-aware loop closes the documented design gap; further hardening would introduce logic not derivable from the gap as currently stated.
- **Driver-level integration test.** No existing testcontainer infrastructure for the Oracle driver package. Building it for one bug isn't proportional. The downstream `Statement.Text` → `conn.ExecContext` → go-ora chain performs no transformation, so splitter unit-test cell coverage is the load-bearing surface.
- **Full plan→issue→rollout integration test.** Path is exercised by other rollout tests (`backend/tests/transaction_mode_test.go`, etc.); adds no marginal coverage of the splitter delta.
- **Variable rename of `prevStopTokenIndex`.** The name is mildly confusing in the inner loop's local context but renaming would balloon the diff. Add the explanatory comment (already in Task 3 Step 1) and leave the name.
