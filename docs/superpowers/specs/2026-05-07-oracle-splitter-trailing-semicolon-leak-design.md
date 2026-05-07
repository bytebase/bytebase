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
if nextIdx := prevStopTokenIndex + 1; nextIdx < len(tokens.GetAllTokens()); {
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

The splitter's contract should be: **after consuming statement N**,
`prevStopTokenIndex` points to the last token that "belongs" to N (including any
trailing `;`), so statement N+1's `leadingContent` starts cleanly at the first
non-belonging token.

The current code encodes a weaker assumption — that `;`, when present, sits
**immediately** after `stmt.GetStop()`. That holds for tightly written SQL
(`stmt;`) and breaks for any statement with hidden-channel tokens before `;`.

The corrected statement of intent: **scan forward through hidden-channel
tokens (whitespace, comments) to find the trailing `;`, if any; consume it and
stop. Otherwise (the next default-channel token is not `;`), do not advance.**

## 5. Fix

Replace the single-step lookahead at `split.go:108-113` with:

```go
prevStopTokenIndex = stmt.GetStop().GetTokenIndex()
for nextIdx := prevStopTokenIndex + 1; nextIdx < len(tokens.GetAllTokens()); nextIdx++ {
    next := tokens.Get(nextIdx)
    if next.GetTokenType() == parser.PlSqlParserSEMICOLON {
        prevStopTokenIndex = nextIdx
        break
    }
    if next.GetChannel() == antlr.TokenDefaultChannel {
        // Hit a real next-statement token; this stmt has no trailing ';'.
        break
    }
}
```

Properties:

* **Grammar-stable.** The skip predicate is `channel != DEFAULT`. We do not
  enumerate token types. `PlSqlLexer.g4:2621-2624` confirms whitespace,
  `--` comments, `/* */` comments, and `REMARK` comments all go to channel
  `HIDDEN`. If the grammar adds another hidden-channel token type later, no
  code change is needed.
* **Backward-compatible.** For `stmt;` (no hidden tokens), the first iteration
  matches `;` and breaks — identical to the old behavior.
* **Bounded.** The loop bails on the first default-channel token that isn't
  `;`, so it cannot consume a `;` that belongs to a later statement.
* **Local.** No signature change, no new helper, no callsite churn. The only
  observable behavior delta is that a leaked `;` no longer prefixes the next
  statement's `Text` and `Range.Start`.

`byteOffsetEnd` is computed before this lookahead (`split.go:82`), so it
already excludes the `;`. No change needed there.

## 6. Test plan

### 6.1 Splitter unit tests (Layer 1)

Add cases to `backend/plugin/parser/plsql/test-data/test_split.yaml`. Each
covers a distinct cell of the design gap:

| # | cell                                                  | input fixture                                                         | new? |
|---|-------------------------------------------------------|-----------------------------------------------------------------------|------|
| a | immediate `;`                                         | (existing) "multiple SELECT statements"                               | no   |
| b | whitespace before `;` (BYT-9367 exact)                | `insert into t values('a',1) ;\n\ninsert into t values('b',2) ;`     | yes  |
| c | comment before `;`                                    | `insert into t values('a',1) /* note */ ;\ninsert into t values('b',2) ;` | yes  |
| d | multi-newline / mixed whitespace before `;`           | `insert into t values('a',1)\n\n;\n\ninsert into t values('b',2);`    | yes  |
| e | no `;` at end of input                                | (existing) "multiple statements with newlines"                        | no   |
| f | hidden tokens then default-channel non-`;` (no sep.)  | `insert into t values('a',1)\ninsert into t values('b',2)`            | yes  |
| g | `needSemicolon` branch (anonymous block, `/`)         | (existing) "anonymous block with forward slash" et al.                | no   |

**Four new cases.** Each new case asserts the exact `Text`, `Range.Start`,
`Range.End`, `Start.Line`/`Column`, `End.Line`/`Column` for every emitted
statement. Cell (f) is the load-bearing guard for the loop's `break` on
default-channel: without it we don't test that the loop refuses to consume
non-existent `;`s.

For non-`needSemicolon` statements (`INSERT`/`UPDATE`/`DELETE`/`SELECT`/DDL),
the trailing `;` is excluded from `Statement.Text` (existing behavior at
`split.go:71-77`). For `needSemicolon` statements (anonymous blocks, procedure
bodies, etc.), the trailing `;` is preserved.

### 6.2 Driver-level integration test (Layer 2) — skipped

There is no existing Oracle driver integration test using `testcontainer`
(`backend/plugin/db/oracle/oracle_test.go` is `TestParseVersion` only). Building
new testcontainer scaffolding for one bug isn't proportional. The downstream
chain `Statement.Text` → `conn.ExecContext` → go-ora performs no transformation,
so once the splitter emits clean text the rest is correct by construction.
Layer 1's cell-coverage matrix is the load-bearing test surface.

If a future Oracle driver bug warrants such infra, the schema package's
`testcontainer.GetTestOracleContainer` pattern is the obvious starting point.

## 7. Audit (other engines / other splitters)

Confirmed the bug pattern is one-engine, one-function:

| splitter                        | mechanism                                                     | vulnerable? |
|---------------------------------|---------------------------------------------------------------|-------------|
| `plsql.SplitSQL`                | parse-tree walk + `prevStopTokenIndex` + 1-token lookahead    | **yes (this fix)** |
| `plsql.SplitSQLForCompletion`   | parse-tree walk, no `leadingContent` computation              | no          |
| `base.SplitSQLByLexer` (snowflake, …) | token-stream walk, `;` always lands in current buffer  | no          |
| `mysql.SplitSQL`, `tsql.SplitSQL`, `pg.SplitSQL`, `tidb.SplitSQL` | no `prevStopTokenIndex` / single-token-`;`-lookahead pattern | no |

No code changes triggered by the audit. Recorded so a future reader can see why
the fix is one-engine and not a sweep.

## 8. Risk and rollback

* **Blast radius.** `SplitSQL` is shared across Oracle issue execution and SQL
  Editor (Admin and Query). The behavior change is strictly "consume a trailing
  `;` we previously failed to consume." The immediate-`;` and no-`;` cases
  match the old behavior bit-for-bit (loop's first iteration hits the `;` or
  hits a default-channel token).
* **Rollback.** Revert the one commit. No data, schema, or proto changes; no
  new dependencies; no callsite changes.

## 9. Out of scope

* Defensive `leadingContent` rebuild (e.g., construct from default-channel
  tokens only, or strip a leading `;\s*`). Mentioned as a long-term hardening
  in the original Linear analysis (§7); intentionally not bundled. The
  channel-aware loop closes the documented design gap; further hardening would
  introduce logic not derivable from the gap as currently stated.
* Full plan→issue→rollout integration test in `backend/tests/`. The path is
  already exercised by other rollout tests (`transaction_mode_test.go`, etc.)
  and adds no marginal coverage of the splitter delta.
