# Oracle (and Trino) splitter trailing `;` leak (BYT-9367) — design

## 1. Problem

Two simple Oracle `INSERT`s submitted as one Bytebase issue:

```sql
insert into ODISTG.ODS_DATA_RETENTION_DROP_PAR_CONFIG values('RPT', 'DBA_CHANGE_INFO ',183) ;

insert into ODISTG.ODS_DATA_RETENTION_DROP_PAR_CONFIG values('RPT', 'CHANGE_INFO_SERVICE ',183) ;
```

Issue execution emits `Command Execute (2)`: the first `INSERT` succeeds, the
second fails with `ORA-00900: invalid SQL statement error occur at position: 1`.
SQL Editor on superficially the same input succeeds; the difference is paste-
normalization, not a separate code path (see §3).

Both layers go through the same splitter, so the bug must live there. It does.
A code audit during review found the same bug class in the Trino splitter
(`backend/plugin/parser/trino/split.go:88-91`); both are fixed in this PR.

## 2. Root cause

`backend/plugin/parser/plsql/split.go:108-113` — the "advance past trailing `;`
so it does not bleed into the next statement's `leadingContent`" step:

```go
prevStopTokenIndex = stmt.GetStop().GetTokenIndex()
if nextIdx := prevStopTokenIndex + 1; nextIdx < len(tokens.GetAllTokens()) {
    if nextToken := tokens.Get(nextIdx); nextToken.GetTokenType() == parser.PlSqlParserSEMICOLON {
        prevStopTokenIndex = nextIdx
    }
}
```

For an `INSERT`, `stmt.GetStop()` is the closing `)`. ANTLR's
`CommonTokenStream` keeps whitespace and comments on the **hidden channel** but
still in the stream. Around the first `;` the tokens are:

```
… 16: ')' (channel 0)
   17: ' ' (channel 1, hidden)
   18: ';' (channel 0)
   19: '\n' (channel 1, hidden)
   20: '\n' (channel 1, hidden)
   21: 'insert' (channel 0)
…
```

`tokens.Get(stopIdx + 1)` returns the whitespace token, not `;`. The equality
check fails, `prevStopTokenIndex` stays on `)`, and the next statement's
`leadingContent` (computed by `tokens.GetTextFromTokens(prevStop+1,
nextStart-1)`) absorbs the space, the `;`, and both newlines. Statement #2 goes
to Oracle as `" ;\n\ninsert into …"` — Oracle parses position 1 and reports
`ORA-00900`.

Trigger-condition table:

| input ending          | second stmt leading | result                    |
|-----------------------|---------------------|---------------------------|
| `,183) ;` (space)     | `" ;\n…"`           | **ORA-00900 (broken)**    |
| `,183);` (no space)   | `"\n…"`             | OK                        |
| `,183)` (no `;`)      | `"\n…"`             | OK                        |

Gated on whitespace (or any hidden-channel token) between the statement's stop
token and the trailing `;`.

The Trino splitter exhibits the same bug class via the same single-token SEMI
lookahead at `trino/split.go:88-91`. Trino's lexer routes whitespace
(`WS_`) and comments (`SIMPLE_COMMENT_`, `BRACKETED_COMMENT_`) to
`channel(HIDDEN)`, and `tokens := stream.GetAllTokens()` (line 63) returns
hidden tokens, so the same shape (`SELECT 1 ; SELECT 2`) leaks ` ;` into the
next statement's `text` (computed via `statement[byteOffsetStart:rangeEnd]`).

## 3. Why "SQL Editor worked"

There is one Oracle splitter (`backend/plugin/parser/plsql/split.go:13`,
`base.RegisterSplitterFunc(storepb.Engine_ORACLE, SplitSQL)`). All callers route
through it:

| caller                | path                                             |
|-----------------------|--------------------------------------------------|
| Issue / rollout       | `Driver.Execute` → `plsqlparser.SplitSQL`        |
| SQL Editor (Admin)    | `Driver.QueryConn` → `plsqlparser.SplitSQL`      |
| SQL Editor (Query)    | `parserbase.SplitMultiSQL` → `QueryConn` → same  |

So byte-identical input must hit the same leak. The SQL Editor screenshot
"working" is best explained by paste/format normalization stripping the space
before `;` (which §2 shows avoids the leak). No path-specific bypass exists.

## 4. Design gap

The splitter has two linked invariants that the original 1-token lookahead
encoded only partially:

**Invariant A — Text correctness.** After consuming statement N,
`prevStopTokenIndex` must point to the last token that "belongs" to N
(including any trailing `;` separator), so statement N+1's `leadingContent`
starts cleanly at the first non-belonging token. The OLD code's lookahead
encoded a weaker assumption: that `;`, when present, sits **immediately** after
`stmt.GetStop()`. That holds for tightly written SQL (`stmt;`) and breaks for
any statement with hidden-channel tokens before `;`.

**Invariant B — Range alignment.** `byteOffsetStart` of statement N+1 (which
becomes `Range.Start` of N+1 and the input to `CalculateLineAndColumn` for
`Start.Line`/`Start.Column`) must equal the byte position of the first
character of N+1's `leadingContent`. This is so that `source[Range.Start :
Range.End]` reproduces `Text`, and `Start.Line`/`Column` points at where
the statement actually begins in source.

In the OLD code, `byteOffsetStart` of N+1 = `byteOffsetEnd` of N = byte after
N's `lastToken` text. For statements where `prevStopTokenIndex` equals
`lastToken` (e.g., BYT-9367 shape, `stop = )`), Invariant B holds. For
statements where `prevStopTokenIndex` is advanced past `lastToken` by the
1-token lookahead (e.g., `stop = ;`, `lastToken = char-before-;`), Invariant B
is off by 1 byte (a pre-existing quirk visible in fixture
`multiple SELECT statements`: `Range.Start = 16` is the position of `;`, not
the position of the leading space at byte 17 where Text actually begins).

**The fix's Text-only loop change satisfies Invariant A.** For BYT-9367 inputs,
it advances `prevStopTokenIndex` past N bytes of hidden tokens + the `;`. But
without a corresponding advance of `byteOffsetStart`, Invariant B is now off
by N bytes (where N can be tens of bytes for `/* long comment */ ;` patterns),
which would visibly drift `Start.Line`/`Column` and `source[Range:]` for
downstream consumers (e.g., BYT-9089 rollout error mapping). The fix must
satisfy both invariants.

The corrected statement of intent: **(A) scan forward through hidden-channel
tokens to find the trailing `;`, if any; consume it and stop. Otherwise (the
next default-channel token is not `;`), do not advance. (B) If the scan
consumed any tokens, advance `byteOffsetStart` (= next statement's
`Range.Start`) by the byte length of those consumed tokens, so it lands at
the first byte of the next statement's `leadingContent`.**

The Trino splitter only needs Invariant A's loop. Trino's `text` is already
`statement[byteOffsetStart:rangeEnd]` directly (no separate `leadingContent`),
and `rangeEnd = endToken.GetStop() + 1` extends naturally to the byte after
the consumed `;` once `finalEndIdx` advances. Both invariants are preserved
by the loop alone.

## 5. Fix

### 5.1 plsql

Replace the single-step lookahead at `split.go:108-113` with a loop plus a
post-loop bookkeeping update:

```go
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
        // Hit a real next-statement token; this stmt has no trailing ';'.
        break
    }
}
// If the loop consumed any tokens, advance byteOffsetStart so the next
// statement's Range.Start lands at the byte AFTER the consumed `;` (matching
// where its leadingContent actually begins in source).
if prevStopTokenIndex > loopStart {
    consumed := tokens.GetTextFromTokens(allTokens[loopStart+1], allTokens[prevStopTokenIndex])
    byteOffsetStart += len(consumed)
}
```

Properties:

* **Grammar-stable.** The skip predicate is `channel != DEFAULT`. We do not
  enumerate token types. `PlSqlLexer.g4` routes whitespace (`SPACES`,
  line 2638) and all comment forms (`SINGLE_LINE_COMMENT`,
  `MULTI_LINE_COMMENT`, `REMARK_COMMENT`, lines 2621-2624) to `channel(HIDDEN)`.
  If the grammar adds another hidden-channel token type later, no code change
  is needed.
* **Backward-compatible for inputs that don't trigger the bug.** For
  `stmt;stmt2` (DML/DDL with stop = last non-`;` token, `;` is immediate next
  token): loop's first iteration matches `;` and breaks, with bookkeeping
  triggered (consumed = `";"`, byteOffsetStart += 1). For `stmt;` where stop
  is `;` itself (parse tree includes `;`, e.g., `needSemicolon` blocks): loop
  starts at `;+1`, sees hidden tokens or next stmt's start, and bails or
  consumes a stray `;` — the bookkeeping fires only if a stray `;` was
  consumed, which preserves the existing 1-byte mismatch for non-stray cases.
* **No existing fixture Range values change.** For the 14 existing fixtures,
  the loop either does not advance `prevStopTokenIndex` past the original
  stop+1 position (because `;` is immediate or the next default-channel token
  bails the loop) — so `consumed` is at most 1 byte (the immediate `;`), and
  the bookkeeping update happens to land on values consistent with the OLD
  code's `byteOffsetStart = byteOffsetEnd_of_prev_stmt + 1` arithmetic that
  already happened for the immediate-`;` cases. Verified by mental trace on
  `multiple SELECT statements`: OLD `byteOffsetStart=16`; NEW after
  bookkeeping `byteOffsetStart += 1 = 17`. **This IS a 1-byte numerical
  change** — see §6.2 for which existing fixtures will be re-recorded.
* **Bounded.** The loop bails on the first default-channel token that isn't
  `;`, so it cannot consume a `;` that belongs to a later statement.
* **Local.** No signature change, no new helper, no callsite churn.

### 5.2 trino

Replace the single-step lookahead at `trino/split.go:88-91` with the same
channel-aware loop:

```go
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

No bookkeeping update needed. Trino computes `text :=
statement[byteOffsetStart:rangeEnd]` (line 101), and `rangeEnd =
endToken.GetStop() + 1` (line 97) where `endToken = tokens[finalEndIdx]`.
Once `finalEndIdx` advances to the consumed `;`, `rangeEnd` extends to byte
after `;`, and `byteOffsetStart` of the next statement (set on line 123 to
`rangeEnd`) lands cleanly at the next statement's leadingContent. Both
invariants are satisfied by the loop alone.

### 5.3 Spec correction (subsumed by §4)

The original draft's §5 said `byteOffsetEnd` is computed before the lookahead
and "no change needed there." That was correct for `byteOffsetEnd` itself,
but missed that `byteOffsetStart` of the next statement (= this statement's
`byteOffsetEnd`) needs to advance when the loop consumes tokens. The full
§4 design above corrects this.

## 6. Test plan

### 6.1 plsql splitter unit tests

Add cases to `backend/plugin/parser/plsql/test-data/test_split.yaml`. Each
covers a distinct cell of the design gap:

| # | cell                                                  | input fixture                                                         | new? |
|---|-------------------------------------------------------|-----------------------------------------------------------------------|------|
| a | immediate `;`                                         | (existing) "multiple SELECT statements"                               | no   |
| b | whitespace before `;` (BYT-9367 exact)                | `insert into t values('a',1) ;\n\ninsert into t values('b',2) ;`     | yes  |
| c | comment before `;`                                    | `insert into t values('a',1) /* note */ ;\ninsert into t values('b',2) ;` | yes  |
| d | multi-newline / mixed whitespace before `;`           | `insert into t values('a',1)\n\n;\n\ninsert into t values('b',2);`    | yes  |
| e | no `;` at end of input                                | (existing) "multiple statements with newlines"                        | no   |
| f | hidden tokens then default-channel non-`;` (no sep.)  | needSemicolon block followed by `/` with no `;` after `END`: `BEGIN NULL; END\n/\nBEGIN NULL; END\n/` | yes  |
| g | `needSemicolon` branch (anonymous block, `/`)         | (existing) "anonymous block with forward slash" et al.                | no   |
| h | trailing `;` with hidden tokens before it at EOF      | `insert into t values('a',1) ;` (no second statement)                  | yes  |

**Five new cases.** Cell (f) replaces the prior misclassification (forward-
slash fixtures with `;` before `/` exercise the loop's *consume* branch, not
the *bail-on-default-channel* branch). The new cell (f) input has no `;`
between `END` and `/`, so the loop walks past the newline (HIDDEN) and bails
on `/` (DEFAULT, FORWARD_SLASH, not SEMI) without consuming anything —
locking the bookkeeping no-op invariant. Cell (h) covers the EOF bound of
the loop.

For non-`needSemicolon` statements (`INSERT`/`UPDATE`/`DELETE`/`SELECT`/DDL),
the trailing `;` is excluded from `Statement.Text` (existing behavior at
`split.go:71-77`). For `needSemicolon` statements (anonymous blocks, procedure
bodies, etc.), the trailing `;` is preserved.

### 6.2 plsql existing fixtures — Range value changes

Three existing fixtures will see numerical Range/Start.Column shifts (1 byte /
1 column) from the bookkeeping fix in §5.1 because they exercise the
"immediate `;` consumed by lookahead" path that the new bookkeeping now
correctly accounts for:

| fixture (line in `test_split.yaml`)                        | what shifts                                  |
|-------------------------------------------------------------|----------------------------------------------|
| `multiple SELECT statements` (line 1)                       | stmt 2 `Range.Start` 16→17; `Start.Column` 17→18; `Range.End` 33→34 |
| `multiple statements with newlines` (line 28)               | stmt 2's `Range`/`Start.Column` shift by 1 byte/col |
| `position semantic: multi-statement with leading whitespace` (line 342) | stmt 2 Range/Start shift by 1                |

`Text`, `End.Line`/`End.Column`, and `Empty` are unchanged. Re-record via
`go test -args -record` and verify in diff that ONLY Range/Start.Column
shift, NEVER Text. Any Text change would indicate a bug elsewhere.

The four anonymous-block-with-`/` fixtures (lines 55, 79, 120, 150) are
unchanged — `prevStopTokenIndex` points at `;` (the parse-tree stop),
loop walks past `\n` and bails on `/`, no consumption, no bookkeeping
update.

### 6.3 trino splitter unit tests

Add cases to `backend/plugin/parser/trino/test-data/test_split.yaml`:

| # | cell                                       | input fixture                                                |
|---|--------------------------------------------|--------------------------------------------------------------|
| 1 | whitespace before `;` (Trino BYT-9367)     | `SELECT 1 ; SELECT 2 ;`                                      |
| 2 | comment before `;`                         | `SELECT 1 /* note */ ; SELECT 2 ;`                            |
| 3 | multi-newline before `;`                   | `SELECT 1\n\n;\n\nSELECT 2;`                                 |

Trino's existing fixtures (`test_split.yaml`) all use immediate `;` or no `;` at
end — none will see Range value changes under the §5.2 fix. Verify in diff:
zero existing-fixture changes.

### 6.4 Driver-level integration tests — skipped

There is no existing testcontainer infrastructure for the Oracle or Trino
driver packages targeting this code path. Building it for one bug isn't
proportional. The downstream chain `Statement.Text` → `conn.ExecContext` →
go-ora / Trino driver performs no transformation, so once the splitter emits
clean text the rest is correct by construction. §6.1 / §6.3 cell-coverage is
the load-bearing surface.

## 7. Audit (other engines / other splitters)

Cross-engine audit, updated:

| splitter                                            | mechanism                                                     | status                                |
|-----------------------------------------------------|---------------------------------------------------------------|---------------------------------------|
| `plsql.SplitSQL`                                    | parse-tree walk + `prevStopTokenIndex` + 1-token lookahead    | **fixed in this PR (loop + bookkeeping)** |
| `trino.splitByParser`                               | parse-tree walk + `finalEndIdx` + 1-token lookahead           | **fixed in this PR (loop only)**       |
| `plsql.SplitSQLForCompletion`                       | parse-tree walk, no `leadingContent` computation              | not affected                          |
| `base.SplitSQLByLexer` (snowflake, …)               | token-stream walk, `;` always lands in current buffer         | not affected                          |
| `mysql.SplitSQL`, `tsql.SplitSQL`, `pg.SplitSQL`, `tidb.SplitSQL` | no `prevStopTokenIndex` / single-token-`;`-lookahead pattern | not affected            |
| `trino.splitByTokenizer` (fallback path at `trino/split.go:27-29`) | tokenizer-based, different mechanism            | verified during implementation (§Plan Task 1)  |

## 8. Risk and rollback

* **Blast radius.** `plsql.SplitSQL` is shared across Oracle issue execution
  and SQL Editor (Admin and Query). `trino.splitByParser` is shared across
  Trino issue execution and SQL Editor.
* **Existing test fixture re-records.** Three plsql fixtures (§6.2) will see
  1-byte/1-column numerical shifts in Range/Start.Column. No Text changes.
  Trino fixtures unchanged. The PR description and commit message must call
  out these specific fixtures and what shifted, so a reviewer can scan the
  diff and confirm the changes are mechanical re-records, not bugs.
* **Rollback.** Revert the one commit. No data, schema, or proto changes;
  no new dependencies; no callsite changes.

## 9. Out of scope

* **`trino.splitByTokenizer` fallback.** The fallback path uses
  `tokenizer.NewTokenizer` / `SplitStandardMultiSQL`, a different mechanism.
  Implementation Task 1 verifies whether this path has the same bug class;
  if it does, file a follow-up ticket rather than expanding scope.
* **Multi-`;` runs (`stmt;;stmt2`).** The new loop consumes one `;` per
  statement. If multiple `;` appear with hidden tokens between, only the
  first is consumed and subsequent ones leak. Pre-existing limitation, not
  made worse by this fix; document if reported.
* **Empty-statement edge case after `needSemicolon` blocks** (e.g.,
  `BEGIN NULL; END;\n;\nSELECT 1 FROM dual;` where the standalone `;` is a
  PL/SQL empty-statement boundary). The new loop consumes that standalone
  `;` rather than emitting an empty stmt, which actually matches existing
  behavior at `plsql.go:142` (`stmtText == ";"` is skipped). Documented as
  a known shape; not regressed.
* **Defensive `leadingContent` rebuild.** Mentioned as long-term hardening
  in the original Linear analysis. The channel-aware loop + bookkeeping
  closes the documented design gap.
* **Full plan→issue→rollout integration test.** Path is exercised by other
  rollout tests; adds no marginal coverage.
