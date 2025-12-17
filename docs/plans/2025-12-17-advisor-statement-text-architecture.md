# Advisor Statement Text Architecture Fix

## Problem

The current advisor architecture has a bug in how statement text is extracted for error messages:

1. `sm.GetStatementsForChecks` returns per-statement `ParsedStatement` objects (each with its own `Text` field)
2. `SQLReviewCheck` calls `ExtractASTs()` which **discards** the per-statement text
3. Advisors receive only `[]AST` and the full SQL text (`Statements string`)
4. Advisors try to extract statement text using ANTLR line numbers: `extractStatementText(fullText, lineNum, lineNum)`
5. **Bug:** ANTLR line numbers are relative to each statement (1-based), not the full text, causing wrong text extraction

## Solution

Pass `[]base.ParsedStatement` instead of `[]base.AST`. Each `ParsedStatement` already contains:
- `Text` - the individual statement's SQL
- `AST` - the parsed tree
- `BaseLine` - 0-based offset for position calculations

## Changes

### 1. `advisor.Context` Struct

**Before:**
```go
type Context struct {
    // ...
    AST        []base.AST   // Per-statement ASTs (loses text)
    Statements string       // Full SQL text (error-prone extraction)
    // ...
}
```

**After:**
```go
type Context struct {
    // ...
    ParsedStatements []base.ParsedStatement  // Complete per-statement info

    // Deprecated - kept temporarily for backward compatibility during migration
    AST        []base.AST
    Statements string
    // ...
}
```

### 2. `sql_review.go`

**Before:**
```go
stmts, parseResult := sm.GetStatementsForChecks(checkContext.DBType, statements)
asts := base.ExtractASTs(stmts)  // Loses statement text!
// ...
checkContext.AST = asts
checkContext.Statements = statements
```

**After:**
```go
stmts, parseResult := sm.GetStatementsForChecks(checkContext.DBType, statements)
// ...
checkContext.ParsedStatements = stmts
checkContext.AST = base.ExtractASTs(stmts)  // Keep for backward compat
checkContext.Statements = statements         // Keep for backward compat
```

### 3. Per-Engine `utils.go`

Update `getANTLRTree()` or create new helper:

```go
// ParsedStatementInfo contains all info needed for checking a single statement
type ParsedStatementInfo struct {
    Tree     antlr.Tree
    Tokens   *antlr.CommonTokenStream
    BaseLine int
    Text     string  // The statement's SQL text
}

// getParsedStatements extracts statement info from the advisor context
func getParsedStatements(checkCtx advisor.Context) ([]ParsedStatementInfo, error) {
    if checkCtx.ParsedStatements == nil {
        return nil, errors.New("ParsedStatements not provided")
    }

    var results []ParsedStatementInfo
    for _, stmt := range checkCtx.ParsedStatements {
        antlrAST, ok := base.GetANTLRAST(stmt.AST)
        if !ok {
            return nil, errors.New("AST type mismatch")
        }
        results = append(results, ParsedStatementInfo{
            Tree:     antlrAST.Tree,
            Tokens:   antlrAST.Tokens,
            BaseLine: stmt.BaseLine,
            Text:     stmt.Text,
        })
    }
    return results, nil
}
```

### 4. Individual Advisors

**Before (buggy):**
```go
func (*SomeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
    parseResults, err := getANTLRTree(checkCtx)
    rule := &someRule{
        statementsText: checkCtx.Statements,  // Full text
    }

    for _, parseResult := range parseResults {
        rule.SetBaseLine(parseResult.BaseLine)
        antlr.ParseTreeWalkerDefault.Walk(checker, parseResult.Tree)
    }
}

// In rule - buggy extraction:
stmtText := extractStatementText(r.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
```

**After (clean):**
```go
func (*SomeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
    stmtInfos, err := getParsedStatements(checkCtx)

    for _, stmtInfo := range stmtInfos {
        rule := &someRule{
            statementText: stmtInfo.Text,  // Per-statement text
        }
        rule.SetBaseLine(stmtInfo.BaseLine)
        antlr.ParseTreeWalkerDefault.Walk(checker, stmtInfo.Tree)
    }
}

// In rule - direct usage:
stmtText := r.statementText  // Already have it
```

### 5. Delete `extractStatementText`

Remove from `utils.go` - no longer needed.

## Files to Modify

| File | Change |
|------|--------|
| `backend/plugin/advisor/advisor.go` | Add `ParsedStatements` field to Context |
| `backend/plugin/advisor/sql_review.go` | Pass ParsedStatements to context |
| `backend/plugin/advisor/pg/utils.go` | Add `getParsedStatements()`, delete `extractStatementText()` |
| `backend/plugin/advisor/pg/advisor_*.go` (20+ files) | Use new pattern |
| `backend/plugin/advisor/mysql/utils.go` | Similar changes if applicable |
| `backend/plugin/advisor/tidb/utils.go` | Similar changes if applicable |
| Other engine utils.go files | Similar changes if applicable |

## Affected Advisors (PostgreSQL)

Files currently using `checkCtx.Statements` that need migration:
- advisor_statement_where_required_select.go
- advisor_statement_where_required_update_delete.go
- advisor_statement_no_select_all.go
- advisor_statement_no_leading_wildcard_like.go
- advisor_statement_non_transactional.go
- advisor_statement_disallow_commit.go
- advisor_statement_disallow_on_del_cascade.go
- advisor_statement_affected_row_limit.go
- advisor_statement_dml_dry_run.go
- advisor_table_require_pk.go
- advisor_table_no_fk.go
- advisor_table_disallow_partition.go
- advisor_table_comment_convention.go
- advisor_naming_fully_qualified.go
- advisor_naming_primary_key_convention.go
- advisor_migration_compatibility.go
- advisor_insert_row_limit.go
- advisor_insert_must_specify_column.go
- advisor_insert_disallow_order_by_rand.go
- advisor_builtin_prior_backup_check.go

## Migration Strategy

Single atomic PR:
1. Add `ParsedStatements` field to Context
2. Update `sql_review.go` to populate it
3. Add `getParsedStatements()` helper to each engine's utils.go
4. Update all advisors to use new pattern
5. Delete `extractStatementText()` function
6. Remove deprecated `AST` and `Statements` fields (or keep if needed elsewhere)

## Testing

- Existing advisor tests should continue to pass
- Focus on multi-statement SQL inputs where line offsets matter
- Verify error messages contain correct statement text
