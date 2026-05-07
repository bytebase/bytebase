# Splitter trailing `;` leak (BYT-9367) — implementation plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `plsql.SplitSQL` and `trino.splitByParser` so a trailing `;` separated from the statement's parse-tree stop token by hidden-channel tokens (whitespace, comments) is correctly consumed by the current statement instead of leaking into the next statement's text/range.

**Architecture:** Replace single-token SEMI lookaheads with channel-aware loops. plsql also needs a post-loop `byteOffsetStart` bookkeeping update so `Range.Start` of the next statement aligns with where its leadingContent actually begins in source. Trino's text/range model (`statement[byteOffsetStart:rangeEnd]`) handles bookkeeping naturally, so only the loop is needed there.

**Tech Stack:** Go, ANTLR4 (`github.com/antlr4-go/antlr/v4`), `github.com/bytebase/parser/plsql`, `github.com/bytebase/parser/trino`, YAML test runner (`backend/plugin/parser/base/split_test_runner.go`).

**Spec:** `docs/superpowers/specs/2026-05-07-oracle-splitter-trailing-semicolon-leak-design.md`

---

## File Structure

| File | Role | Change |
|---|---|---|
| `backend/plugin/parser/plsql/split.go` | Oracle splitter | Replace `split.go:108-113` lookahead block with channel-aware loop + bookkeeping |
| `backend/plugin/parser/plsql/test-data/test_split.yaml` | plsql YAML fixtures | Append 5 new fixtures (cells b, c, d, f, h); 3 existing fixtures will re-record with 1-byte/1-column Range/Start.Column shifts |
| `backend/plugin/parser/trino/split.go` | Trino splitter | Replace `split.go:88-91` lookahead block with channel-aware loop |
| `backend/plugin/parser/trino/test-data/test_split.yaml` | Trino YAML fixtures | Append 3 new fixtures (whitespace-before-`;`, comment, multi-newline); existing fixtures unchanged |

## Test plan recap (from spec §6)

**plsql** — five new cells (b, c, d, f, h); see spec §6.1. Three existing fixtures re-record (§6.2): `multiple SELECT statements`, `multiple statements with newlines`, `position semantic: multi-statement with leading whitespace`. Range/Start.Column shift by 1; Text unchanged.

**Trino** — three new fixtures; existing fixtures unchanged.

---

## Chunk 1: implementation

### Task 1: Verify starting state and audit `splitByTokenizer` fallback

**Files:**
- Read: `backend/plugin/parser/plsql/split.go`
- Read: `backend/plugin/parser/trino/split.go`
- Read: `backend/plugin/parser/tokenizer/standard.go` (or wherever `SplitStandardMultiSQL` is defined)

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

If the code differs (e.g., line numbers shifted), update the file references in subsequent steps.

- [ ] **Step 2: Verify the buggy trino code is at the expected location**

```bash
sed -n '85,92p' backend/plugin/parser/trino/split.go
```

Expected output:
```go
// Find the actual start position
endIdx := stopToken.GetTokenIndex()

// Check if there's a semicolon after the statement and include it
finalEndIdx := endIdx
if endIdx+1 < len(tokens) && tokens[endIdx+1].GetTokenType() == trinoparser.TrinoLexerSEMICOLON_ {
    finalEndIdx = endIdx + 1
}
```

- [ ] **Step 3: Audit the trino tokenizer fallback for the same bug class**

The trino splitter has a fallback at `splitByTokenizer` (`trino/split.go:27-29`) that uses `tokenizer.NewTokenizer` / `SplitStandardMultiSQL`. Verify whether that path has the same single-token SEMI lookahead bug.

```bash
grep -nE "SEMICOLON|';'" backend/plugin/parser/tokenizer/standard.go | head -20
grep -nE "SplitStandardMultiSQL" backend/plugin/parser/tokenizer/*.go
```

Read the relevant `SplitStandardMultiSQL` function. If it walks the input character-by-character (typical tokenizer pattern) rather than using a parse-tree-stop + lookahead pattern, it is structurally immune. Document the finding in a one-line comment in the spec's §7 audit row for `splitByTokenizer`. If it has the same bug, **stop and file a follow-up ticket** rather than expanding scope.

- [ ] **Step 4: Run the existing plsql + trino test suites to establish a green baseline**

```bash
go test -count=1 ./backend/plugin/parser/plsql/ ./backend/plugin/parser/trino/
```

Expected: all existing subtests pass.

### Task 2: Add new test fixtures with placeholder results

**Files:**
- Modify: `backend/plugin/parser/plsql/test-data/test_split.yaml` (append five new cases)
- Modify: `backend/plugin/parser/trino/test-data/test_split.yaml` (append three new cases)

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
- description: 'BYT-9367: anonymous block followed by / with no terminating semicolon (cell f)'
  input: "BEGIN NULL; END\n/\nBEGIN NULL; END\n/"
  result: []
- description: 'BYT-9367: trailing semicolon with leading space at EOF (cell h)'
  input: "insert into t values('a',1) ;"
  result: []
```

`result: []` is a placeholder — Step 4 below records the expected values. Hand-computing positions is error-prone (`byteOffsetStart` arithmetic depends on token byte offsets across hidden channels), and the `-record` mode is the load-bearing tool here.

- [ ] **Step 2: Append three trino test cases**

Append to the END of `backend/plugin/parser/trino/test-data/test_split.yaml`:

```yaml
- description: 'BYT-9367 trino: trailing semicolon with leading space does not leak'
  input: "SELECT 1 ; SELECT 2 ;"
  result: []
- description: 'BYT-9367 trino: trailing semicolon with leading inline comment does not leak'
  input: "SELECT 1 /* note */ ; SELECT 2 ;"
  result: []
- description: 'BYT-9367 trino: trailing semicolon with multiple newlines does not leak'
  input: "SELECT 1\n\n;\n\nSELECT 2;"
  result: []
```

### Task 3: Apply the fixes

**Files:**
- Modify: `backend/plugin/parser/plsql/split.go:108-113`
- Modify: `backend/plugin/parser/trino/split.go:88-91`

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
					consumed := tokens.GetTextFromTokens(allTokens[loopStart+1], allTokens[prevStopTokenIndex])
					byteOffsetStart += len(consumed)
				}
```

The `antlr` import is already present at `split.go:4`. The `parser.PlSqlParserSEMICOLON` symbol is unchanged.

- [ ] **Step 2: Apply the trino fix**

In `backend/plugin/parser/trino/split.go`, replace lines 87-91 (the `// Check if there's a semicolon...` block):

```go
				// Check if there's a semicolon after the statement and include it
				finalEndIdx := endIdx
				if endIdx+1 < len(tokens) && tokens[endIdx+1].GetTokenType() == trinoparser.TrinoLexerSEMICOLON_ {
					finalEndIdx = endIdx + 1
				}
```

with:

```go
				// Walk forward through hidden-channel tokens (whitespace, comments) to
				// find a trailing ';' belonging to this statement. Bail on the first
				// default-channel non-';' token — that's the start of the next statement.
				finalEndIdx := endIdx
				for nextIdx := endIdx + 1; nextIdx < len(tokens); nextIdx++ {
					next := tokens[nextIdx]
					if next.GetTokenType() == trinoparser.TrinoLexerSEMICOLON_ {
						finalEndIdx = nextIdx
						break
					}
					if next.GetChannel() == antlr.TokenDefaultChannel {
						break
					}
				}
```

The `antlr` import is already present at `trino/split.go:4`. The `trinoparser.TrinoLexerSEMICOLON_` symbol is unchanged.

- [ ] **Step 3: Format both files**

```bash
gofmt -w backend/plugin/parser/plsql/split.go backend/plugin/parser/trino/split.go
```

### Task 4: Record YAML and verify text/range invariants

**Files:**
- Read/diff: `backend/plugin/parser/plsql/test-data/test_split.yaml`
- Read/diff: `backend/plugin/parser/trino/test-data/test_split.yaml`

- [ ] **Step 1: Record both packages**

```bash
go test -count=1 ./backend/plugin/parser/plsql/ -run TestPLSQLSplitSQL -args -record
go test -count=1 ./backend/plugin/parser/trino/ -run TestTrinoSplitSQL -args -record
```

If the trino test function has a different name, find it:
```bash
grep -n "RunSplitTests" backend/plugin/parser/trino/*_test.go
```

This populates `result:` for new cases and re-records all others.

- [ ] **Step 2: Diff the plsql YAML and verify text invariants for new cells**

```bash
git diff backend/plugin/parser/plsql/test-data/test_split.yaml
```

For each of the five NEW cases, verify stmt 2's `text` is CLEAN:

| cell | input shape | expected stmt 2 text invariant |
|---|---|---|
| b | `,1) ;\n\nstmt2 ;` | starts with `"\n\n"`, NO leading `;`, NO leading space |
| c | `,1) /* note */ ;\nstmt2 ;` | starts with `"\n"`, NO leading `;`, NO leading comment |
| d | `,1)\n\n;\n\nstmt2;` | starts with `"\n\n"` (or similar pure-whitespace prefix); NO leading `;`. **If the parse tree produces 3 unit_statements** (with empty middle from the standalone `;`), confirm the middle stmt has `Empty: true` and stmt 3's text starts with `"\n\n"` instead. |
| f | `BEGIN NULL; END\n/\nBEGIN NULL; END\n/` | 2 anonymous blocks, each with text `"BEGIN NULL; END;"` (the trailing `;` synthesized by `needSemicolon`); NO leading `/` or `\n` on stmt 2 |
| h | `,1) ;` (EOF, no stmt 2) | 1 stmt only, text `"insert into t values('a',1)"`, NO `;` in text (non-needSemicolon strips trailing `;`) |

If any invariant is violated, the fix is wrong — investigate before continuing.

- [ ] **Step 3: Diff the plsql YAML and verify expected existing-fixture changes**

The same diff should show 1-byte/1-column shifts (NOT text changes) in three existing fixtures:

| fixture (line)                                          | expected change                                  |
|----------------------------------------------------------|--------------------------------------------------|
| `multiple SELECT statements` (line 1)                   | stmt 2: `range.start` 16→17, `range.end` 33→34, `start.column` 17→18 |
| `multiple statements with newlines` (line 28)           | stmt 2: range/start.column shift by 1 byte/column |
| `position semantic: multi-statement with leading whitespace` (line 342) | stmt 2: range/start.column shift by 1 |

For each: verify `text` is UNCHANGED. Any text change is a flag.

For the four anonymous-block-with-`/` fixtures (lines 55, 79, 120, 150), verify ZERO changes — they don't trigger the bookkeeping update.

- [ ] **Step 4: Diff the trino YAML and verify text invariants for new cases**

```bash
git diff backend/plugin/parser/trino/test-data/test_split.yaml
```

For each of the three NEW trino cases, verify stmt 2's `text`:

| input | expected stmt 2 text invariant |
|---|---|
| `SELECT 1 ; SELECT 2 ;` | starts with `" "` (leading space), no leading `;`, ends with `";"` (trino includes consumed `;` in text) |
| `SELECT 1 /* note */ ; SELECT 2 ;` | starts with `" "`, no leading comment, no leading `;` |
| `SELECT 1\n\n;\n\nSELECT 2;` | starts with `"\n\n"` (or pure whitespace), no leading `;` |

Verify ZERO changes to existing trino fixtures (none should re-record because none use the bug shape).

- [ ] **Step 5: Run both test suites in normal (non-record) mode**

```bash
go test -count=1 -v ./backend/plugin/parser/plsql/ ./backend/plugin/parser/trino/
```

Expected: all subtests pass.

### Task 5: Lint and build

- [ ] **Step 1: Lint**

```bash
golangci-lint run --allow-parallel-runners ./backend/plugin/parser/plsql/... ./backend/plugin/parser/trino/...
```

Run repeatedly until no issues remain (per `AGENTS.md`). Use `--fix` to auto-fix:

```bash
golangci-lint run --fix --allow-parallel-runners ./backend/plugin/parser/plsql/... ./backend/plugin/parser/trino/...
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
        backend/plugin/parser/plsql/test-data/test_split.yaml \
        backend/plugin/parser/trino/split.go \
        backend/plugin/parser/trino/test-data/test_split.yaml
```

- [ ] **Step 2: Commit**

```bash
git commit -m "$(cat <<'EOF'
fix(plsql,trino): trailing ';' leak when whitespace/comment separates stop and ';'

BYT-9367. The plsql and trino splitters' "advance past trailing ';'"
lookahead inspected only the token immediately after the parse-tree stop,
so any hidden-channel token (whitespace, comment) between the stop and
the ';' prevented consumption. The ';' then leaked into the next
statement's text. For Oracle this surfaced as ORA-00900 at position 1.

Replace the single-token lookaheads with channel-aware loops that skip
hidden-channel tokens until they find ';' (consume) or a default-channel
non-';' token (bail). For plsql, also advance byteOffsetStart past
loop-consumed tokens so the next statement's Range.Start aligns with
where its leadingContent actually begins in source. Trino's text/range
model already handles this naturally via statement[byteOffsetStart:rangeEnd].

Tests:
- 5 new plsql YAML fixtures (cells b/c/d/f/h of the design gap).
- 3 new trino YAML fixtures (Trino-equivalent inputs).
- 3 existing plsql fixtures re-recorded with 1-byte/1-column Range/Start.Column
  shifts (text unchanged): multiple SELECT statements, multiple statements
  with newlines, position semantic: multi-statement with leading whitespace.
- Trino existing fixtures unchanged.

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

- **`trino.splitByTokenizer` fallback.** Audited in Task 1 Step 3. If it has the same bug, separate ticket.
- **Multi-`;` runs (`stmt;;stmt2`).** Pre-existing limitation; not made worse.
- **Driver-level integration tests.** No existing testcontainer infra for either driver targeting this code path; building it for one bug isn't proportional.
- **Defensive `leadingContent` rebuild.** Long-term hardening idea from the original Linear analysis; the channel-aware loop + bookkeeping closes the documented gap.
- **Variable rename of `prevStopTokenIndex` / `finalEndIdx`.** Names are mildly confusing in the inner loop's local context but renaming would balloon the diff.
