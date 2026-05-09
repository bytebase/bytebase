# Oracle splitter trailing `;` leak (BYT-9367) — implementation plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `plsql.SplitSQL` so a trailing `;` separated from the statement's parse-tree stop token by hidden-channel tokens (whitespace, comments) is correctly consumed by the current statement instead of leaking into the next statement's text/range. Also fix the latent `byteOffsetStart` misalignment surfaced by the Range invariant in the design.

**Architecture:** Replace the single-token SEMI lookahead with a channel-aware loop. After the loop, advance `byteOffsetStart` by `len(GetTextFromTokens(...))` of any tokens the loop consumed (matching the byte-length convention at `split.go:82`), so `Range.Start` of the next statement aligns with where its leadingContent actually begins in source. Note: ANTLR token Start/Stop indices are *rune* indices into a `[]rune` input stream, NOT byte offsets — so the bookkeeping must use `len()` of the materialized text, not stop-index arithmetic.

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

Five new fixtures (cells b, c, d, f, h). The current YAML has **15 existing fixtures**; **4 re-record** with 1-byte/1-column Range/Start.Column shifts (`multiple SELECT statements`, `multiple statements with newlines`, `SELECT statements separated by forward slash`, `position semantic: multi-statement with leading whitespace`); **11 stay unchanged** (see spec §6.2 for the full list and rationale per fixture). Text never changes.

---

## Chunk 1: implementation

### Task 1: Establish green baseline

- [ ] **Step 1: Run the existing plsql test suite**

```bash
go test -count=1 ./backend/plugin/parser/plsql/
```

Expected: all existing subtests pass. The Edit tool in Task 3 will fail loudly if the source has drifted from the spec's `split.go:108-113` reference, so no separate sed-check is needed.

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
				// If the loop consumed any tokens, advance byteOffsetStart by the
				// byte length of those consumed tokens so the next statement's
				// Range.Start lands at the byte AFTER the consumed ';' (matching
				// where its leadingContent actually begins in source).
				//
				// IMPORTANT: use len(GetTextFromTokens(...)) — Go's len() on a
				// string is byte length, while ANTLR token Start/Stop indices are
				// rune indices into the input stream's []rune. For ASCII the
				// difference is zero, but multi-byte UTF-8 inside hidden tokens
				// (e.g., a comment containing non-ASCII characters) would diverge.
				// This matches the byte-offset arithmetic at line 82.
				if prevStopTokenIndex > loopStart {
					byteOffsetStart += len(tokens.GetTextFromTokens(allTokens[loopStart+1], allTokens[prevStopTokenIndex]))
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
| d | `,1)\n\n;\n\nstmt2;` | exactly 2 unit_statements (the standalone `;` matches sql_script's bare-`SEMICOLON` alternative, not a unit_statement, so the splitter — which only iterates `IUnit_statementContext` — sees 2 children). Stmt 2 text starts with pure whitespace, NO leading `;` |
| f | `BEGIN NULL; END;\nSELECT 1 FROM dual` | 2 stmts: stmt 1 text = `"BEGIN NULL; END;"`; stmt 2 text = `"\nSELECT 1 FROM dual"` (single `\n` leading, NO `;` leak) |
| h | `,1) ;` (EOF, no stmt 2) | 1 stmt only, text = `"insert into t values('a',1)"`, NO `;` in text |

If any invariant is violated, the fix is wrong — investigate before continuing.

- [ ] **Step 3: Diff and verify expected existing-fixture changes**

The same diff should show 1-byte/1-column shifts (NOT text changes) in exactly 4 of the 15 existing fixtures:

| fixture (line)                                                          | expected change                                      |
|-------------------------------------------------------------------------|------------------------------------------------------|
| `multiple SELECT statements` (line 1)                                   | stmt 2: `range.start` 16→17, `range.end` 33→34, `start.column` 17→18 |
| `multiple statements with newlines` (line 28)                           | stmt 2: `range.start` 20→21, `range.end` 54→55, `start.column` 20→21 |
| `SELECT statements separated by forward slash` (line 120)               | stmt 2 (after `/`): `range.start` 18→19, `range.end` 35→36, `start.column` 1→2 |
| `position semantic: multi-statement with leading whitespace` (line 342) | stmt 2: `range.start` 18→19, `range.end` 40→41, `start.column` 19→20 |

For each: verify `text` is UNCHANGED. Any text change is a flag.

For the OTHER 11 existing fixtures (lines 55, 79, 150 — needSemicolon-with-`/`; lines 172, 205, 220, 235, 238, 241, 312, 327 — single-statement / parse-error / single-stmt-with-leading-whitespace), verify ZERO changes — they don't trigger the bookkeeping update.

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
