# AST Cache Dependencies - PostgreSQL Migration to ANTLR

## Summary

You're correct - we need to focus on what **accesses the AST through the cache**. Here's the complete picture:

## Components Using AST from Cache

### 1. Catalog WalkThrough System ❌ BLOCKER

**File**: `backend/plugin/advisor/catalog/walk_through_for_pg.go:18-29`

```go
func (d *DatabaseState) pgWalkThrough(pgAst any) error {
    nodeList, ok := pgAst.([]ast.Node)  // ❌ Expects pg_query_go AST
    if !ok {
        return errors.Errorf("invalid ast type %T", pgAst)
    }
    for _, node := range nodeList {
        if err := d.pgChangeState(node); err != nil {
            return err
        }
    }
    return nil
}
```

**What it does**:
- Builds virtual schema state by walking through DDL statements
- Validates table/column existence
- Checks foreign key references
- Detects view dependencies

**Impact**: **MUST be migrated** before activating ANTLR advisors

---

### 2. Legacy PostgreSQL Advisors ✅ ALREADY MIGRATED

**Location**: `backend/plugin/advisor/pg/` (55 files)

All legacy advisors receive AST via `ctx.AST` and expect `[]ast.Node` type:

```go
func (*SomeAdvisor) Check(...) {
    stmtList, ok := checkCtx.AST.([]ast.Node)  // Type assertion to legacy AST
    if !ok {
        return nil, errors.Errorf("failed to convert to Node")
    }
    // ... use stmtList
}
```

**Status**: ✅ All 55 advisors already migrated to ANTLR (completed in previous work)

---

## Call Chain

```
sheet.Manager.GetASTsForChecks()
    ↓ (returns cached AST)
advisor.SQLReviewCheck()
    ↓ (splits to 2 paths)
    ├─> finder.WalkThrough(asts)  ❌ NEEDS pg_query_go AST
    │       ↓
    │   catalog.DatabaseState.pgWalkThrough()
    │       ↓
    │   Expects: []ast.Node (from pg_query_go)
    │
    └─> advisor.Check(ctx.AST = asts)  ✅ ANTLR advisors ready
            ↓
        Individual advisors receive AST
```

## The ONLY Blocker

**The catalog walkthrough system** is the **ONLY** feature that relies on pg_query_go AST via the cache.

### Where it's called:
**File**: `backend/plugin/advisor/sql_review.go:542`

```go
func SQLReviewCheck(...) {
    asts, parseResult := sm.GetASTsForChecks(checkContext.DBType, statements)

    finder := checkContext.Catalog.GetFinder()
    if !builtinOnly {
        switch checkContext.DBType {
        case storepb.Engine_POSTGRES:  // ← PostgreSQL enters here
            if err := finder.WalkThrough(asts); err != nil {  // ← Calls catalog walkthrough
                return convertWalkThroughErrorToAdvice(err)
            }
        }
    }

    // Then passes asts to advisors...
}
```

## Migration Required

### What needs to be done:

**Create ANTLR-based catalog walkthrough**:

File: `backend/plugin/advisor/catalog/walk_through_for_pg_antlr.go` (new file)

```go
package catalog

import (
    "github.com/antlr4-go/antlr/v4"
    parser "github.com/bytebase/parser/postgresql"
)

// pgAntlrWalkThrough walks through ANTLR parse tree
func (d *DatabaseState) pgAntlrWalkThrough(tree any) error {
    root, ok := tree.(parser.IRootContext)
    if !ok {
        return errors.Errorf("invalid ANTLR tree type %T", tree)
    }

    // Walk through all statements in the tree
    listener := &catalogBuildingListener{
        databaseState: d,
    }
    antlr.ParseTreeWalkerDefault.Walk(listener, root)

    if listener.err != nil {
        return listener.err
    }
    return nil
}

type catalogBuildingListener struct {
    *parser.BasePostgreSQLParserListener
    databaseState *DatabaseState
    err           error
}

// EnterCreatestmt handles CREATE TABLE
func (l *catalogBuildingListener) EnterCreatestmt(ctx *parser.CreatestmtContext) {
    // Build table state from ANTLR context
    // Similar logic to pgCreateTable() but using ANTLR nodes
}

// EnterAltertablestmt handles ALTER TABLE
func (l *catalogBuildingListener) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
    // Build alter table state from ANTLR context
}

// ... more Enter methods for other DDL statements
```

Then update the dispatcher in `walk_through.go:212`:

```go
case storepb.Engine_POSTGRES:
    // Check if AST is ANTLR tree or legacy ast.Node
    if _, ok := ast.(parser.IRootContext); ok {
        // ANTLR tree - use new walkthrough
        if err := d.pgAntlrWalkThrough(ast); err != nil {
            return err
        }
    } else {
        // Legacy ast.Node - use old walkthrough
        if err := d.pgWalkThrough(ast); err != nil {
            return err
        }
    }
```

## No Other Dependencies

After checking all usages of `GetASTsForChecks()`:

1. ✅ `backend/runner/plancheck/statement_report_executor.go` - Only checks syntax errors, ignores AST
2. ✅ `backend/api/v1/release_service_check.go` - Only checks syntax errors, passes AST to advisors
3. ✅ `backend/plugin/advisor/catalog/walk_through_test.go` - Test only

None of these files use the pg_query_go AST structure directly - they either:
- Ignore the AST completely (only check syntax errors)
- Pass it to advisors (which we've migrated)
- Pass it to catalog walkthrough (which we need to migrate)

## Conclusion

**Answer to your question**:

> "Is there any features relies on pg_query_go AST via AST cache?"

**YES, exactly ONE feature**: The **catalog walkthrough system** (`backend/plugin/advisor/catalog/walk_through_for_pg.go`)

This is the ONLY blocker. Once we migrate the catalog walkthrough to support ANTLR trees, we can:

1. Switch the Parse function registration to return ANTLR trees
2. Disable legacy PostgreSQL advisor registrations
3. Activate the new ANTLR-based advisors

## Next Steps

1. Implement ANTLR-based catalog walkthrough
2. Add dual-mode support (ANTLR + legacy) in WalkThrough dispatcher
3. Test catalog building with ANTLR trees
4. Switch Parse registration to ANTLR
5. Deactivate legacy advisors
