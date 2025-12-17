# Advisor Statement Text Architecture Fix - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the bug where advisors extract wrong statement text by passing per-statement `ParsedStatement` (with `Text` field) instead of just ASTs.

**Architecture:** Add `ParsedStatements []base.ParsedStatement` field to `advisor.Context`, update `sql_review.go` to populate it, create `getParsedStatements()` helper in each engine's utils.go, then migrate all advisors to use `stmt.Text` directly instead of `extractStatementText()`.

**Tech Stack:** Go, ANTLR parser

**Worktree:** `/Users/h3n4l/OpenSource/bytebase/.worktrees/advisor-statement-text`

---

## Task 1: Add ParsedStatements Field to Context

**Files:**
- Modify: `backend/plugin/advisor/advisor.go:38-72`

**Step 1: Add the new field to Context struct**

In `backend/plugin/advisor/advisor.go`, add `ParsedStatements` field after line 51:

```go
// Context is the context for advisor.
type Context struct {
	DBSchema              *storepb.DatabaseSchemaMetadata
	EnableSDL             bool
	EnablePriorBackup     bool
	EnableGhost           bool
	ListDatabaseNamesFunc base.ListDatabaseNamesFunc
	InstanceID            string
	IsObjectCaseSensitive bool

	// SQL review rule special fields.
	AST              []base.AST
	Rule             *storepb.SQLReviewRule
	OriginalMetadata *model.DatabaseMetadata
	FinalMetadata    *model.DatabaseMetadata
	Driver           *sql.DB

	// ParsedStatements contains complete per-statement info including text.
	// Use this instead of AST + Statements for accessing statement text.
	ParsedStatements []base.ParsedStatement

	// CurrentDatabase is the current database.
	CurrentDatabase string
	// Statement is the original statement of AST, it is used for some PostgreSQL
	// advisors which need to check the token stream.
	Statements string
	// ... rest of fields unchanged
```

**Step 2: Build to verify no syntax errors**

Run: `go build ./backend/plugin/advisor/...`
Expected: Success (no output)

**Step 3: Commit**

```bash
git add backend/plugin/advisor/advisor.go
git commit -m "feat(advisor): add ParsedStatements field to Context

Adds ParsedStatements []base.ParsedStatement to advisor.Context to provide
per-statement text alongside AST, fixing the statement text extraction bug."
```

---

## Task 2: Populate ParsedStatements in sql_review.go

**Files:**
- Modify: `backend/plugin/advisor/sql_review.go:99-112`

**Step 1: Update SQLReviewCheck to populate ParsedStatements**

Change lines 99-112 in `sql_review.go`:

```go
// SQLReviewCheck checks the statements with sql review rules.
func SQLReviewCheck(
	ctx context.Context,
	sm *sheet.Manager,
	statements string,
	ruleList []*storepb.SQLReviewRule,
	checkContext Context,
) ([]*storepb.Advice, error) {
	stmts, parseResult := sm.GetStatementsForChecks(checkContext.DBType, statements)
	asts := base.ExtractASTs(stmts)

	builtinOnly := len(ruleList) == 0

	if !checkContext.NoAppendBuiltin {
		// Append builtin rules to the rule list.
		ruleList = append(ruleList, GetBuiltinRules(checkContext.DBType)...)
	}

	if asts == nil || len(ruleList) == 0 {
		return parseResult, nil
	}

	if !builtinOnly && checkContext.FinalMetadata != nil {
		switch checkContext.DBType {
		case storepb.Engine_TIDB, storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_POSTGRES, storepb.Engine_OCEANBASE:
			if advice := schema.WalkThrough(checkContext.DBType, checkContext.FinalMetadata, asts); advice != nil {
				return []*storepb.Advice{advice}, nil
			}
		default:
			// Other database types don't need walkthrough
		}
	}

	var errorAdvices, warningAdvices []*storepb.Advice
	for _, rule := range ruleList {
		if rule.Engine != storepb.Engine_ENGINE_UNSPECIFIED && rule.Engine != checkContext.DBType {
			continue
		}

		ruleType := rule.Type

		// Set per-rule fields
		checkContext.AST = asts
		checkContext.Statements = statements
		checkContext.Rule = rule
		checkContext.ParsedStatements = stmts  // ADD THIS LINE

		adviceList, err := Check(
```

**Step 2: Build to verify**

Run: `go build ./backend/plugin/advisor/...`
Expected: Success

**Step 3: Run existing tests to verify no regression**

Run: `go test -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/pg`
Expected: PASS

**Step 4: Commit**

```bash
git add backend/plugin/advisor/sql_review.go
git commit -m "feat(advisor): populate ParsedStatements in SQLReviewCheck

Pass the full ParsedStatement slice to advisors, making per-statement
text available without line-number-based extraction."
```

---

## Task 3: Add ParsedStatementInfo and getParsedStatements() to pg/utils.go

**Files:**
- Modify: `backend/plugin/advisor/pg/utils.go`

**Step 1: Add ParsedStatementInfo struct and getParsedStatements function**

Add after line 59 (after `getANTLRTree` function):

```go
// ParsedStatementInfo contains all info needed for checking a single statement.
type ParsedStatementInfo struct {
	Tree     antlr.Tree
	Tokens   *antlr.CommonTokenStream
	BaseLine int
	Text     string
}

// getParsedStatements extracts statement info from the advisor context.
// This is the preferred way to access statements - use stmtInfo.Text directly
// instead of extractStatementText().
func getParsedStatements(checkCtx advisor.Context) ([]ParsedStatementInfo, error) {
	if checkCtx.ParsedStatements == nil {
		// Fallback to old behavior for backward compatibility
		return getParsedStatementsFromAST(checkCtx)
	}

	var results []ParsedStatementInfo
	for _, stmt := range checkCtx.ParsedStatements {
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
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

// getParsedStatementsFromAST is the fallback when ParsedStatements is not available.
// Deprecated: Use getParsedStatements with ParsedStatements field instead.
func getParsedStatementsFromAST(checkCtx advisor.Context) ([]ParsedStatementInfo, error) {
	if checkCtx.AST == nil {
		return nil, errors.New("AST is not provided in context")
	}

	var results []ParsedStatementInfo
	for _, unifiedAST := range checkCtx.AST {
		antlrAST, ok := base.GetANTLRAST(unifiedAST)
		if !ok {
			return nil, errors.New("AST type mismatch: expected ANTLR-based parser result")
		}
		results = append(results, ParsedStatementInfo{
			Tree:     antlrAST.Tree,
			Tokens:   antlrAST.Tokens,
			BaseLine: base.GetLineOffset(antlrAST.StartPosition),
			Text:     "", // Not available in fallback mode
		})
	}
	return results, nil
}
```

**Step 2: Build to verify**

Run: `go build ./backend/plugin/advisor/pg/...`
Expected: Success

**Step 3: Commit**

```bash
git add backend/plugin/advisor/pg/utils.go
git commit -m "feat(advisor/pg): add getParsedStatements helper

Adds ParsedStatementInfo struct and getParsedStatements() function that
provides per-statement text directly, eliminating need for line-based
text extraction."
```

---

## Task 4: Migrate advisor_statement_where_required_select.go

**Files:**
- Modify: `backend/plugin/advisor/pg/advisor_statement_where_required_select.go`

**Step 1: Update the advisor to use getParsedStatements**

Replace the Check function and rule struct:

```go
// Check checks for WHERE clause requirement in SELECT statements.
func (*StatementWhereRequiredSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtInfos, err := getParsedStatements(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, stmtInfo := range stmtInfos {
		rule := &statementWhereRequiredSelectRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			statementText: stmtInfo.Text,
		}
		rule.SetBaseLine(stmtInfo.BaseLine)

		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtInfo.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type statementWhereRequiredSelectRule struct {
	BaseRule
	statementText string
}
```

**Step 2: Update the checkSelect function to use statementText directly**

Replace the checkSelect function:

```go
// checkSelect is a common function to check for WHERE clause requirement
func (r *statementWhereRequiredSelectRule) checkSelect(
	ctx antlr.ParserRuleContext,
	_ antlr.Token,
	_ antlr.Token,
	checkFunc func() (hasWhere bool, hasFrom bool),
) {
	// Check if this SELECT has a WHERE clause and FROM clause
	hasWhere, hasFrom := checkFunc()

	// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1
	if !hasFrom {
		return
	}

	// If there's a WHERE clause, all good
	if hasWhere {
		return
	}

	// Use the statement text directly
	stmtText := strings.TrimSpace(r.statementText)
	if stmtText == "" {
		stmtText = "<unknown statement>"
	}

	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.StatementNoWhere.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
```

**Step 3: Remove findTopLevelLine function (no longer needed)**

Delete the `findTopLevelLine` function (lines 141-155 approximately).

**Step 4: Update imports if needed**

Add `"strings"` to imports if not already present.

**Step 5: Build and test**

Run: `go build ./backend/plugin/advisor/pg/... && go test -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/pg -run "Test"`
Expected: PASS

**Step 6: Commit**

```bash
git add backend/plugin/advisor/pg/advisor_statement_where_required_select.go
git commit -m "refactor(advisor/pg): migrate where-required-select to use statement text

Use getParsedStatements() and per-statement Text field instead of
extractStatementText() with line numbers."
```

---

## Task 5: Migrate advisor_statement_where_required_update_delete.go

**Files:**
- Modify: `backend/plugin/advisor/pg/advisor_statement_where_required_update_delete.go`

**Step 1: Read current implementation**

Read the file to understand the current structure.

**Step 2: Update to use getParsedStatements pattern**

Follow the same pattern as Task 4:
- Replace `getANTLRTree` with `getParsedStatements`
- Loop over `stmtInfos`, create rule per statement with `stmtInfo.Text`
- Replace `extractStatementText(r.statementsText, ...)` with `r.statementText`

**Step 3: Build and test**

Run: `go build ./backend/plugin/advisor/pg/... && go test -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/pg`
Expected: PASS

**Step 4: Commit**

```bash
git add backend/plugin/advisor/pg/advisor_statement_where_required_update_delete.go
git commit -m "refactor(advisor/pg): migrate where-required-update-delete to use statement text"
```

---

## Task 6-24: Migrate Remaining PostgreSQL Advisors

Repeat the pattern from Task 4-5 for each of these files:

| Task | File |
|------|------|
| 6 | advisor_statement_no_select_all.go |
| 7 | advisor_statement_no_leading_wildcard_like.go |
| 8 | advisor_statement_non_transactional.go |
| 9 | advisor_statement_disallow_commit.go |
| 10 | advisor_statement_disallow_on_del_cascade.go |
| 11 | advisor_statement_affected_row_limit.go |
| 12 | advisor_statement_dml_dry_run.go |
| 13 | advisor_table_require_pk.go |
| 14 | advisor_table_no_fk.go |
| 15 | advisor_table_disallow_partition.go |
| 16 | advisor_table_comment_convention.go |
| 17 | advisor_naming_fully_qualified.go |
| 18 | advisor_naming_primary_key_convention.go |
| 19 | advisor_migration_compatibility.go |
| 20 | advisor_insert_row_limit.go |
| 21 | advisor_insert_must_specify_column.go |
| 22 | advisor_insert_disallow_order_by_rand.go |
| 23 | advisor_builtin_prior_backup_check.go |

**For each file:**
1. Read current implementation
2. Replace `getANTLRTree` with `getParsedStatements`
3. Update rule struct: `statementsText string` -> `statementText string`
4. Replace `extractStatementText(r.statementsText, line, line)` with `r.statementText`
5. Build and test
6. Commit with message: `refactor(advisor/pg): migrate <advisor-name> to use statement text`

---

## Task 25: Delete extractStatementText Function

**Files:**
- Modify: `backend/plugin/advisor/pg/utils.go`

**Step 1: Remove extractStatementText function**

Delete the `extractStatementText` function (approximately lines 110-133).

**Step 2: Build to verify no remaining usages**

Run: `go build ./backend/plugin/advisor/pg/...`
Expected: Success (if any file still uses it, you'll get a compile error - fix those first)

**Step 3: Commit**

```bash
git add backend/plugin/advisor/pg/utils.go
git commit -m "refactor(advisor/pg): remove extractStatementText function

No longer needed now that all advisors use per-statement Text directly."
```

---

## Task 26: Run Full Test Suite and Lint

**Step 1: Run all advisor tests**

Run: `go test -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/...`
Expected: PASS

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners ./backend/plugin/advisor/...`
Expected: No errors (or fix any that appear)

**Step 3: Build full project**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Success

---

## Task 27: Final Commit and PR Preparation

**Step 1: Review all changes**

Run: `git log --oneline main..HEAD`
Verify all commits are present.

**Step 2: Squash if needed (optional)**

If too many commits, consider interactive rebase to squash related changes.

**Step 3: Push branch**

```bash
git push -u origin fix/advisor-statement-text-architecture
```

---

## Notes for Implementation

1. **Pattern consistency:** Each migrated advisor should follow the exact same pattern - loop over `stmtInfos`, create rule with `stmtInfo.Text`

2. **Testing:** After each file migration, run `go test` to catch regressions immediately

3. **Edge cases:** Some advisors may have special logic (like `advisor_table_require_pk.go` which accumulates state). Preserve that logic while updating text access.

4. **Fallback:** The `getParsedStatementsFromAST` fallback ensures backward compatibility during migration if some code paths don't populate `ParsedStatements`.
