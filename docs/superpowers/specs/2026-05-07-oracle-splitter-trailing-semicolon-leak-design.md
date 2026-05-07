# Oracle splitter trailing `;` leak (BYT-9367) — design

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

The bug is gated on the grammar shape "`;` is at sql_script level (outside
`unit_statement`), not inside the rule itself." Per `PlSqlParser.g4:32-37`,
ordinary DML/DDL is wrapped by `sql_script: ((sql_plus_command |
unit_statement) SEMICOLON? | SEMICOLON)* EOF`, so `stmt.GetStop()` of a SELECT
or INSERT is the last meaningful token (e.g., `t1`, `)`), not the trailing
`;`. `needSemicolon` rules (anonymous blocks, procedure/function/package/
trigger bodies) include their `;` in the rule itself
(`anonymous_block: ... END SEMICOLON`, `create_function_body: ... SEMICOLON`),
so `stmt.GetStop()` IS the `;` for those — the loop starts past it and does
not (in the absence of a stray double-`;`) advance further.

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

**Invariant A — Text correctness.** After processing statement N,
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
N's `lastToken` text. `prevStopTokenIndex` is then advanced (by the 1-token
lookahead) past N's `lastToken` to the consumed `;` IF one is immediately
present, but `byteOffsetStart` is **not** advanced in lockstep. The result is
that for the existing immediate-`;` case (`SELECT t1; SELECT t2`),
`byteOffsetStart` of stmt 2 lands at the byte position of the consumed `;`
itself, off by 1 from where the next `leadingContent` byte actually begins
(visible in the existing fixture: stmt 2 `Range.Start = 16` is the byte of
`;`, while text begins with the leading space at byte 17). For BYT-9367
inputs, this off-by-N drift is even larger (whitespace + `;` + newlines).

**The fix's loop change alone satisfies Invariant A.** For BYT-9367 inputs,
it advances `prevStopTokenIndex` past N bytes of hidden tokens + the `;`. But
without a corresponding advance of `byteOffsetStart`, Invariant B is now off
by N bytes (where N can be tens of bytes for `/* long comment */ ;`
patterns), which would visibly drift `Start.Line`/`Column` and
`source[Range:]` for downstream consumers (e.g., BYT-9089 rollout error
mapping). The fix must satisfy both invariants.

The corrected statement of intent: **(A) scan forward through hidden-channel
tokens to find the trailing `;`, if any; consume it and stop. Otherwise (the
next default-channel token is not `;`), do not advance. (B) If the scan
consumed any tokens, advance `byteOffsetStart` (= next statement's
`Range.Start`) by the byte length of those consumed tokens, so it lands at
the first byte of the next statement's `leadingContent`.**

## 5. Fix

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
// where its leadingContent actually begins in source). Use stop-index
// arithmetic rather than materializing the consumed text — equivalent and
// cheaper.
if prevStopTokenIndex > loopStart {
    byteOffsetStart += allTokens[prevStopTokenIndex].GetStop() - allTokens[loopStart].GetStop()
}
```

Properties:

* **Grammar-stable.** The skip predicate is `channel != DEFAULT`. We do not
  enumerate token types. `PlSqlLexer.g4` routes whitespace (`SPACES`,
  line 2638) and all comment forms (`SINGLE_LINE_COMMENT`,
  `MULTI_LINE_COMMENT`, `REMARK_COMMENT`, lines 2621-2624) to `channel(HIDDEN)`.
  If the grammar adds another hidden-channel token type later, no code change
  is needed.
* **Behavior for inputs that don't trigger the bug:**
  - **Existing `stmt;stmt2` (non-needSemicolon, immediate `;`)**: `stop` is
    the last token of the unit_statement (e.g., `t1`); loop's first iteration
    matches `;` and breaks; bookkeeping fires (`consumed = ";"`,
    `byteOffsetStart += 1`). This **shifts the existing 1-byte mismatch to
    correctness**: stmt 2's `Range.Start` now lands at the byte AFTER `;`
    rather than at `;` itself. Existing fixtures with this shape will see
    a 1-byte/1-column re-record (see §6.2).
  - **Existing `needSemicolon` blocks followed by `/`**: `stop` is the
    block's own `;`; loop walks past `\n` (HIDDEN) and bails on `/` (DEFAULT,
    FORWARD_SLASH, not SEMI); `prevStopTokenIndex` unchanged; bookkeeping
    does not fire. **Unchanged.**
  - **Statements with no trailing `;`**: loop bails on first default-channel
    token (or EOF); `prevStopTokenIndex` unchanged; bookkeeping does not fire.
    **Unchanged.**
* **Bounded.** The loop bails on the first default-channel token that isn't
  `;`, so it cannot consume a `;` that belongs to a later statement.
* **Local.** No signature change, no new helper, no callsite churn.
* **Stop-index arithmetic.** `allTokens[i].GetStop()` returns the inclusive
  byte-end of token `i` in the input. The delta between consumed-`;`'s end
  and `loopStart`'s end equals the byte length of the run of tokens consumed
  by the loop (whitespace + `;`), without needing to materialize the text.

## 6. Test plan

### 6.1 Splitter unit tests — new fixtures

Add cases to `backend/plugin/parser/plsql/test-data/test_split.yaml`. Each
covers a distinct cell of the design gap:

| # | cell                                                  | input fixture                                                         | new? |
|---|-------------------------------------------------------|-----------------------------------------------------------------------|------|
| a | immediate `;` (non-needSemicolon)                     | (existing) `multiple SELECT statements`                               | no, will re-record (§6.2) |
| b | whitespace before `;` (BYT-9367 exact)                | `insert into t values('a',1) ;\n\ninsert into t values('b',2) ;`     | yes  |
| c | comment before `;`                                    | `insert into t values('a',1) /* note */ ;\ninsert into t values('b',2) ;` | yes  |
| d | multi-newline / mixed whitespace before `;`           | `insert into t values('a',1)\n\n;\n\ninsert into t values('b',2);`    | yes  |
| e | no `;` at end of input                                | (existing) `multiple statements with newlines`                        | no, will re-record (§6.2) |
| f | hidden tokens then default-channel non-`;` (no sep.)  | `BEGIN NULL; END;\nSELECT 1 FROM dual` (anonymous block then SELECT, no `;` between) | yes |
| g | `needSemicolon` branch (anonymous block, `/`)         | (existing) `anonymous block with forward slash` et al.                | no, unchanged |
| h | trailing `;` with hidden tokens before it at EOF      | `insert into t values('a',1) ;` (no second statement)                  | yes  |

**Five new cases.** Cell (f) input is `BEGIN NULL; END;\nSELECT 1 FROM dual`:
the anonymous block (needSemicolon, stop = `;`) is followed by a SELECT with
only `\n` between them. The loop starts at `;+1`, walks past `\n` (HIDDEN),
hits `SELECT` (DEFAULT, not SEMI) → bails. `prevStopTokenIndex` unchanged,
bookkeeping does not fire. This locks the bail-on-default-channel branch
with a non-`/` token (the existing forward-slash fixtures bail on
`FORWARD_SLASH`; cell (f) bails on a regular keyword). Cell (h) covers the
EOF bound of the loop and the bookkeeping firing without a downstream
consumer.

For non-`needSemicolon` statements (`INSERT`/`UPDATE`/`DELETE`/`SELECT`/DDL),
the trailing `;` is excluded from `Statement.Text` (existing behavior at
`split.go:71-77`). For `needSemicolon` statements (anonymous blocks, procedure
bodies, etc.), the trailing `;` is preserved.

### 6.2 Existing fixtures — Range value changes

Four existing fixtures will see numerical Range/Start.Column shifts (1 byte /
1 column) from the bookkeeping fix in §5 because they exercise the
"non-needSemicolon stmt with immediate `;` consumed by lookahead" path that
the new bookkeeping now correctly accounts for:

| fixture (line in `test_split.yaml`)                                     | what shifts                                  |
|-------------------------------------------------------------------------|----------------------------------------------|
| `multiple SELECT statements` (line 1)                                   | stmt 2 `Range.Start` 16→17; `Range.End` 33→34; `Start.Column` 17→18 |
| `multiple statements with newlines` (line 28)                           | stmt 2's `Range`/`Start.Column` shift by 1 byte/col |
| `SELECT statements separated by forward slash` (line 120)               | stmt 2 (after `/`): `Range`/`Start.Column` shift by 1 (the SELECT-then-`;` of stmt 1 triggers bookkeeping; the sql_plus_command branch then carries forward the shifted `byteOffsetStart`) |
| `position semantic: multi-statement with leading whitespace` (line 342) | stmt 2 Range/Start shift by 1                |

`Text`, `End.Line`/`End.Column`, and `Empty` are unchanged. Re-record via
`go test -args -record` and verify in diff that ONLY Range/Start.Column
shift, NEVER Text. Any Text change would indicate a bug elsewhere.

The four anonymous-block-with-`/` fixtures (lines 55, 79, 150, 172 region)
are unchanged — `prevStopTokenIndex` points at `;` (the parse-tree stop of
the needSemicolon body), loop walks past `\n` and bails on `/`, no
consumption, no bookkeeping update.

Single-statement fixtures (`ALTER TABLE`, `CALL procedure`, `DROP TABLESPACE`,
`CREATE TABLE`, leading-newlines/spaces position fixtures) are unchanged —
no second statement to be affected by `byteOffsetStart` advancement.

### 6.3 Driver-level integration tests — skipped

There is no existing testcontainer infrastructure for the Oracle driver
package targeting this code path. Building it for one bug isn't proportional.
The downstream chain `Statement.Text` → `conn.ExecContext` → go-ora performs
no transformation, so once the splitter emits clean text the rest is correct
by construction. §6.1's cell-coverage matrix is the load-bearing surface.

## 7. Audit (other engines / other splitters)

Cross-engine audit:

| splitter                                            | mechanism                                                     | status                                |
|-----------------------------------------------------|---------------------------------------------------------------|---------------------------------------|
| `plsql.SplitSQL`                                    | parse-tree walk + `prevStopTokenIndex` + 1-token lookahead    | **fixed in this PR**                  |
| `plsql.SplitSQLForCompletion`                       | parse-tree walk, no `leadingContent` computation              | not affected                          |
| `base.SplitSQLByLexer` (snowflake, …)               | token-stream walk, `;` always lands in current buffer         | not affected                          |
| `mysql.SplitSQL`, `tsql.SplitSQL`, `pg.SplitSQL`, `tidb.SplitSQL` | no `prevStopTokenIndex` / single-token-`;`-lookahead pattern | not affected            |
| `trino.splitByParser` (`trino/split.go:88-91`)      | parse-tree walk + 1-token lookahead AT FIRST GLANCE looks similar, BUT: `singleStatement: statement SEMICOLON_` (`TrinoParser.g4:38-40`) **requires** `;` in the parse tree; `singleStmt.GetStop()` is the `;` itself; `rangeEnd = endToken.GetStop() + 1` already extends past `;`. The 1-token lookahead handles a different case: a stray DOUBLE-`;` (empty statement absorption), not the BYT-9367 leak shape. | **not affected**           |
| `trino.splitByTokenizer` fallback                   | tokenizer-based, different mechanism                          | not affected (tokenizer in `tokenizer/tokenizer.go:381` walks character-by-character; `;` always terminates a buffer cleanly) |

## 8. Risk and rollback

* **Blast radius.** `plsql.SplitSQL` is shared across Oracle issue execution
  and SQL Editor (Admin and Query).
* **Existing test fixture re-records.** Four plsql fixtures (§6.2) will see
  1-byte/1-column numerical shifts in Range/Start.Column. No Text changes.
  The PR description and commit message must call out these specific fixtures
  and what shifted, so a reviewer can scan the diff and confirm the changes
  are mechanical re-records, not bugs.
* **Range correctness improvement.** The bookkeeping fix improves Range
  alignment for the existing immediate-`;` case (was off by 1 byte;
  now correct) — this is a quiet *improvement* shipped alongside the
  BYT-9367 leak fix, for free. Downstream consumers using
  `source[Range.Start:Range.End]` to reproduce Text will now get matching
  bytes for these fixtures (previously they got bytes shifted by 1).
* **Rollback.** Revert the one commit. No data, schema, or proto changes;
  no new dependencies; no callsite changes.

## 9. Out of scope

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
