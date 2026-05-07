# Oracle splitter trailing `;` leak (BYT-9367) — implementation plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `plsql.SplitSQL` so a trailing `;` separated from the statement's parse-tree stop token by hidden-channel tokens (whitespace, comments) is correctly consumed by the current statement instead of leaking into the next statement's text/range. Also fix the latent `byteOffsetStart` misalignment surfaced by the Range invariant in the design.

**Architecture:** Replace the single-token SEMI lookahead with a channel-aware loop. After the loop, advance `byteOffsetStart` by the byte length of any tokens the loop consumed (using token stop-index arithmetic — no string materialization), so `Range.Start` of the next statement aligns with where its leadingContent actually begins in source.

**Tech Stack:** Go, ANTLR4 (`github.com/antlr4-go/antlr/v4`), `github.com/bytebase/parser/plsql`, YAML test runner (`backend/plugin/parser/base/split_test_runner.go`).

**Spec:** `docs/superpowers/specs/2026-05-07-oracle-splitter-trailing-semicolon-leak-design.md`

**Scope note:** Trino was investigated and ruled out — `trino/parser/TrinoParser.g4:38-40` requires `;` in the `singleStatement` parse tree, so the BYT-9367 leak shape can't occur there. See spec §7 audit.

---

## File Structure

| File | Role | Change |
|---|---|---|
| `backend/plugin/parser/plsql/split.go` | Oracle splitter | Replace `split.go:108-113` lookahead block with channel-aware loop + 2-line bookkeeping update |
| `backend/plugin/parser/plsql/test-data/test_split.yaml` | YAML fixtures | Append 5 new fixtures (cells b, c, d, f, h); 4 existing fixtures will re-record with 1-byte/1-column Range/Start.Column shifts |

## Test plan recap (from spec §6)

Five new fixtures (cells b, c, d, f, h). Four existing fixtures re-record (§6.2): `multiple SELECT statements`, `multiple statements with newlines`, `SELECT statements separated by forward slash`, `position semantic: multi-statement with leading whitespace`. Range/Start.Column shift by 1; Text unchanged.

---

## Chunk 1: implementation

### Task 1: Verify starting state

**Files:**
- Read: `backend/plugin/parser/plsql/split.go`

- [ ] **Step 1: Verify the buggy plsql code is at the expected location**

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

If the code differs, update the file references in subsequent steps.

- [ ] **Step 2: Run the existing plsql test suite to establish a green baseline**

```bash
go test -count=1 ./backend/plugin/parser/plsql/
```

Expected: all existing subtests pass.

### Task 2: Add new test fixtures with placeholder results

**Files:**
- Modify: `backend/plugin/parser/plsql/test-data/test_split.yaml` (append five new cases)

- [ ] **Step 1: Append five plsql test cases**

Append to the END of `backend/plugin/parser/plsql/test-data/test_split.yaml`:

```yaml
- description: 'BYT-9367: trailing semicolon with leading space does not leak (cell b)'
  input: "insert into t values('a',1) ;\n\ninsert into t values('b',2) ;"
  result: []
- description: 'BYT-9367: trailing semicolon with leading inline comment does not leak (cell c)'
  input: "insert into t values('a',1) /* note */ ;\ninsert into t values('b',2) ;"
  result: []
- description: 'BYT-9367: trailing semicolon with multiple newlines does not leak (cell d)'
  input: "insert into t values('a',1)\n\n;\n\ninsert into t values('b',2);"
  result: []
- description: 'BYT-9367: anonymous block followed by SELECT with no separator (cell f, bail-on-default-channel)'
  input: "BEGIN NULL; END;\nSELECT 1 FROM dual"
  result: []
- description: 'BYT-9367: trailing semicolon with leading space at EOF (cell h)'
  input: "insert into t values('a',1) ;"
  result: []
```

`result: []` is a placeholder — Step 4 below records the expected values via `-record`. Hand-computing positions is error-prone; the `-record` mode is the load-bearing tool.

### Task 3: Apply the fix

**Files:**
- Modify: `backend/plugin/parser/plsql/split.go:108-113`

- [ ] **Step 1: Apply the plsql fix**

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
				// Walk forward through hidden-channel tokens (whitespace, comments) to
				// find a trailing ';' belonging to this statement. Bail on the first
				// default-channel non-';' token — that's the start of the next statement.
				loopStart := stmt.GetStop().GetTokenIndex()
				prevStopTokenIndex = loopStart
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
				// If the loop consumed any tokens, advance byteOffsetStart so the
				// next statement's Range.Start lands at the byte AFTER the consumed
				// ';' (matching where its leadingContent actually begins in source).
				if prevStopTokenIndex > loopStart {
					byteOffsetStart += allTokens[prevStopTokenIndex].GetStop() - allTokens[loopStart].GetStop()
				}
```

The `antlr` import is already present at `split.go:4`. The `parser.PlSqlParserSEMICOLON` symbol is unchanged.

- [ ] **Step 2: Format**

```bash
gofmt -w backend/plugin/parser/plsql/split.go
```

### Task 4: Record YAML and verify text/range invariants

**Files:**
- Read/diff: `backend/plugin/parser/plsql/test-data/test_split.yaml`

- [ ] **Step 1: Record**

```bash
go test -count=1 ./backend/plugin/parser/plsql/ -run TestPLSQLSplitSQL -args -record
```

This populates `result:` for the new cases and re-records all others.

- [ ] **Step 2: Diff and verify text invariants for new cells**

```bash
git diff backend/plugin/parser/plsql/test-data/test_split.yaml
```

For each of the five NEW cases, verify stmt 2's `text` is CLEAN:

| cell | input shape | expected stmt 2 text invariant |
|---|---|---|
| b | `,1) ;\n\nstmt2 ;` | starts with `"\n\n"`, NO leading `;`, NO leading space |
| c | `,1) /* note */ ;\nstmt2 ;` | starts with `"\n"`, NO leading `;`, NO leading comment |
| d | `,1)\n\n;\n\nstmt2;` | starts with `"\n\n"` (or pure-whitespace prefix); NO leading `;`. **If parse tree produces 3 unit_statements** (with empty middle from the standalone `;`), confirm middle stmt has `Empty: true` and stmt 3's text starts with `"\n\n"`. |
| f | `BEGIN NULL; END;\nSELECT 1 FROM dual` | 2 stmts: stmt 1 text = `"BEGIN NULL; END;"`; stmt 2 text = `"\nSELECT 1 FROM dual"` (single `\n` leading, NO `;` leak) |
| h | `,1) ;` (EOF, no stmt 2) | 1 stmt only, text = `"insert into t values('a',1)"`, NO `;` in text |

If any invariant is violated, the fix is wrong — investigate before continuing.

- [ ] **Step 3: Diff and verify expected existing-fixture changes**

The same diff should show 1-byte/1-column shifts (NOT text changes) in four existing fixtures:

| fixture (line)                                                          | expected change                                      |
|-------------------------------------------------------------------------|------------------------------------------------------|
| `multiple SELECT statements` (line 1)                                   | stmt 2: `range.start` 16→17, `range.end` 33→34, `start.column` 17→18 |
| `multiple statements with newlines` (line 28)                           | stmt 2: range/start.column shift by 1 byte/column    |
| `SELECT statements separated by forward slash` (line 120)               | stmt 2 (after `/`): range/start.column shift by 1 (the SELECT-then-`;` of stmt 1 triggers bookkeeping) |
| `position semantic: multi-statement with leading whitespace` (line 342) | stmt 2: range/start.column shift by 1                 |

For each: verify `text` is UNCHANGED. Any text change is a flag.

For all OTHER existing fixtures (lines 55, 79, 150, 172 region anonymous-block-with-`/`; single-stmt fixtures; position-leading-whitespace fixtures), verify ZERO changes — they don't trigger the bookkeeping update.

- [ ] **Step 4: Run the test suite in normal (non-record) mode**

```bash
go test -count=1 -v ./backend/plugin/parser/plsql/ -run TestPLSQLSplitSQL
```

Expected: all subtests pass, including the five new BYT-9367 subtests.

### Task 5: Lint and build

- [ ] **Step 1: Lint**

```bash
golangci-lint run --allow-parallel-runners ./backend/plugin/parser/plsql/...
```

Run repeatedly until no issues remain. Use `--fix` to auto-fix:

```bash
golangci-lint run --fix --allow-parallel-runners ./backend/plugin/parser/plsql/...
```

Expected: clean.

- [ ] **Step 2: Build the full backend**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: builds successfully.

### Task 6: Commit

- [ ] **Step 1: Stage source and test changes**

```bash
git add backend/plugin/parser/plsql/split.go \
        backend/plugin/parser/plsql/test-data/test_split.yaml
```

- [ ] **Step 2: Commit**

```bash
git commit -m "$(cat <<'EOF'
fix(plsql): trailing ';' leak when whitespace/comment separates stop and ';'

BYT-9367. The plsql splitter's "advance past trailing ';'" lookahead
inspected only the token immediately after the parse-tree stop, so any
hidden-channel token (whitespace, comment) between the stop and the ';'
prevented consumption. The ';' then leaked into the next statement's
text. For Oracle this surfaced as ORA-00900 at position 1.

Replace the single-token lookahead with a channel-aware loop that skips
hidden-channel tokens until it finds ';' (consume) or a default-channel
non-';' token (bail). After the loop, advance byteOffsetStart by the byte
length of any consumed tokens so the next statement's Range.Start aligns
with where its leadingContent actually begins in source. This also
quietly fixes a pre-existing 1-byte off-by-one in the immediate-';' case
(visible in 4 existing fixtures).

Tests:
- 5 new YAML fixtures (cells b/c/d/f/h of the design gap).
- 4 existing fixtures re-recorded with 1-byte/1-column Range/Start.Column
  shifts (text unchanged): multiple SELECT statements, multiple statements
  with newlines, SELECT statements separated by forward slash, position
  semantic: multi-statement with leading whitespace.

Trino was investigated and ruled out — TrinoParser.g4 requires ';' in
the singleStatement parse tree, so the leak shape can't occur there.

Spec: docs/superpowers/specs/2026-05-07-oracle-splitter-trailing-semicolon-leak-design.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 3: Verify clean status**

```bash
git status
```

Expected: working tree clean (besides the spec/plan docs from earlier commits).

---

## Out of scope (for reference)

- **Trino splitter.** Investigated; not affected. `TrinoParser.g4` requires `;` in `singleStatement`.
- **Multi-`;` runs (`stmt;;stmt2`).** Pre-existing limitation; not made worse.
- **Driver-level integration tests.** No existing testcontainer infra targeting this code path.
- **Defensive `leadingContent` rebuild.** Long-term hardening idea; channel-aware loop + bookkeeping closes the documented gap.
- **Variable rename of `prevStopTokenIndex`.** Mildly confusing in the inner loop's local context but renaming would balloon the diff.
