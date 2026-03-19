# PostgreSQL SQL Review: Migrate from ANTLR to Omni

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rewrite all 53 PostgreSQL SQL review advisors to use the omni AST instead of ANTLR parse trees, then remove the ANTLR dependency.

**Architecture:** Each advisor switches from walking ANTLR parse trees via `GenericChecker` to directly type-switching on omni `ast.Node` types. A new `OmniChecker` provides the common dispatch/advice infrastructure. The existing test YAML files validate output parity. Once all advisors pass, the ANTLR parsing path in `parsePgStatements` is removed.

**Tech Stack:** Go, omni parser (`github.com/bytebase/omni/pg/ast`), existing advisor framework

---

## Background

### Current Architecture (ANTLR)

Each advisor's `Check()` method:
1. Calls `base.GetANTLRAST(stmt.AST)` to extract ANTLR parse tree
2. Creates a `GenericChecker` with `Rule` implementations
3. Walks the ANTLR tree with `antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)`
4. Rules match on string `nodeType` (e.g., `"Createstmt"`) and type-assert to ANTLR contexts

Position tracking: ANTLR `ctx.GetStart().GetLine()` returns 1-based line within the statement. `baseLine` (0-based) is added by `AddAdvice()` to make it absolute.

### Target Architecture (Omni)

Each advisor's `Check()` method:
1. Calls `pgparser.GetOmniNode(stmt.AST)` to extract omni `ast.Node`
2. Type-switches directly on `*ast.CreateStmt`, `*ast.AlterTableStmt`, etc.
3. For deep inspection, uses `ast.Inspect(node, func)` on specific sub-trees
4. No reflection-based string matching needed

Position tracking: `OmniAST.StartPosition` is 1-based absolute. Sub-nodes with `Loc` fields carry byte offsets within statement text, convertible via `byteOffsetToRunePosition()`.

### Key Mappings: ANTLR to Omni

| ANTLR nodeType | Omni AST type | Notes |
|---|---|---|
| `Createstmt` | `*ast.CreateStmt` | `.Relation.Relname` for table name |
| `Altertablestmt` | `*ast.AlterTableStmt` | `.Cmds` list of `*ast.AlterTableCmd` |
| `Indexstmt` | `*ast.IndexStmt` | `.Concurrent` bool, `.Relation` |
| `Dropstmt` | `*ast.DropStmt` | `.RemoveType` for object type |
| `Selectstmt` | `*ast.SelectStmt` | `.WhereClause`, `.FromClause` |
| `Insertstmt` | `*ast.InsertStmt` | `.Cols` for column list |
| `Updatestmt` | `*ast.UpdateStmt` | `.WhereClause` |
| `Deletestmt` | `*ast.DeleteStmt` | `.WhereClause` |
| `Renamestmt` | `*ast.RenameStmt` | `.Newname`, `.RenameType` |
| `Variablesetstmt` | `*ast.VariableSetStmt` | `.Name`, `.Kind` |
| `TransactionStmt` | `*ast.TransactionStmt` | `.Kind` for COMMIT detection |
| `ColumnDef` (nested) | `*ast.ColumnDef` | `.Colname`, `.IsNotNull`, `.Constraints` |
| `Constraint` (nested) | `*ast.Constraint` | `.Contype` for PK/FK/UNIQUE/CHECK |

#### AlterTableCmd.Subtype mappings

| ANTLR pattern | Omni `AlterTableType` |
|---|---|
| `cmd.ADD_P() + cmd.ColumnDef()` | `ast.AT_AddColumn` |
| `cmd.ALTER() + cmd.SET() + cmd.NOT() + cmd.NULL_P()` | `ast.AT_SetNotNull` |
| `cmd.ALTER() + cmd.DROP() + cmd.NOT() + cmd.NULL_P()` | `ast.AT_DropNotNull` |
| `cmd.ADD_P() + cmd.Tableconstraint()` | `ast.AT_AddConstraint` |
| `cmd.ALTER() + cmd.TYPE_P()` | `ast.AT_AlterColumnType` |
| `cmd.DROP()` column | `ast.AT_DropColumn` |

#### Constraint.Contype mappings

| ANTLR pattern | Omni `ConstrType` |
|---|---|
| `PRIMARY() + KEY()` | `ast.CONSTR_PRIMARY` |
| `NOT() + NULL_P()` | `ast.CONSTR_NOTNULL` |
| `UNIQUE()` | `ast.CONSTR_UNIQUE` |
| `FOREIGN()` | `ast.CONSTR_FOREIGN` |
| `CHECK()` | `ast.CONSTR_CHECK` |
| `DEFAULT()` | `ast.CONSTR_DEFAULT` |

#### DropStmt.RemoveType mappings

| ANTLR pattern | Omni `ObjectType` |
|---|---|
| `ctx.Object_type_any_name().TABLE()` | `ast.OBJECT_TABLE` |
| `ctx.INDEX()` | `ast.OBJECT_INDEX` |

#### RenameStmt.RenameType

| ANTLR pattern | Omni `ObjectType` |
|---|---|
| `ctx.TABLE()` | `ast.OBJECT_TABLE` |

---

## Phase 1: Infrastructure

### Task 1: Create OmniChecker and OmniBaseRule

**Files:**
- Create: `bytebase/backend/plugin/advisor/pg/generic_checker_omni.go`

This file provides the new infrastructure parallel to `generic_checker.go`.

**Step 1: Write the new infrastructure file**

```go
package pg

import (
	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniRule defines the interface for omni-based SQL validation rules.
type OmniRule interface {
	// OnStatement is called for each top-level statement AST node.
	OnStatement(node ast.Node)

	// Name returns the rule name for logging/debugging.
	Name() string

	// GetAdviceList returns the accumulated advice from this rule.
	GetAdviceList() []*storepb.Advice
}

// OmniBaseRule provides common functionality for omni-based rules.
type OmniBaseRule struct {
	level      storepb.Advice_Status
	title      string
	adviceList []*storepb.Advice
	baseLine   int       // 0-based line offset for this statement
	stmtText   string    // original SQL text of this statement
}

// SetStatement sets the statement context for position calculations.
func (r *OmniBaseRule) SetStatement(baseLine int, stmtText string) {
	r.baseLine = baseLine
	r.stmtText = stmtText
}

// GetAdviceList returns the accumulated advice.
func (r *OmniBaseRule) GetAdviceList() []*storepb.Advice {
	return r.adviceList
}

// AddAdvice adds advice using line 1 of the current statement (most common case).
// The baseLine offset is added automatically.
func (r *OmniBaseRule) AddAdvice(advice *storepb.Advice) {
	if advice.StartPosition != nil {
		advice.StartPosition.Line += int32(r.baseLine)
	}
	r.adviceList = append(r.adviceList, advice)
}

// AddAdviceAbsolute adds advice with an already-absolute line number.
// Use this when you've already computed the absolute line (e.g., stored from a previous statement).
func (r *OmniBaseRule) AddAdviceAbsolute(advice *storepb.Advice) {
	r.adviceList = append(r.adviceList, advice)
}

// LocToLine converts an omni Loc byte offset to a 1-based line number
// relative to the current statement (suitable for AddAdvice which adds baseLine).
func (r *OmniBaseRule) LocToLine(loc ast.Loc) int32 {
	if loc.Start < 0 || r.stmtText == "" {
		return 1
	}
	pos := pgparser.ByteOffsetToRunePosition(r.stmtText, loc.Start)
	return pos.Line
}

// RunOmniRules iterates over parsed statements and dispatches each omni AST node to all rules.
// Returns combined advice from all rules. Skips statements without omni AST.
func RunOmniRules(stmts []base.ParsedStatement, rules []OmniRule) []*storepb.Advice {
	for _, stmt := range stmts {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		baseLine := stmt.BaseLine()
		for _, rule := range rules {
			if br, ok := rule.(interface{ SetStatement(int, string) }); ok {
				br.SetStatement(baseLine, stmt.Text)
			}
			rule.OnStatement(node)
		}
	}
	var allAdvice []*storepb.Advice
	for _, rule := range rules {
		allAdvice = append(allAdvice, rule.GetAdviceList()...)
	}
	return allAdvice
}
```

**Step 2: Export `byteOffsetToRunePosition`**

Modify: `bytebase/backend/plugin/parser/pg/omni.go`

Rename `byteOffsetToRunePosition` to `ByteOffsetToRunePosition` (export it) so advisors can use it.

**Step 3: Create omni-specific utilities**

Create: `bytebase/backend/plugin/advisor/pg/utils_omni.go`

```go
package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/ast"
)

// omniTableName extracts the table name from a RangeVar.
func omniTableName(rv *ast.RangeVar) string {
	if rv == nil {
		return ""
	}
	return rv.Relname
}

// omniSchemaName extracts the schema name from a RangeVar, defaulting to "public".
func omniSchemaName(rv *ast.RangeVar) string {
	if rv == nil || rv.Schemaname == "" {
		return "public"
	}
	return rv.Schemaname
}

// omniConstraintColumns extracts column names from a Constraint's Keys list.
func omniConstraintColumns(c *ast.Constraint) []string {
	if c == nil || c.Keys == nil {
		return nil
	}
	var cols []string
	for _, item := range c.Keys.Items {
		if s, ok := item.(*ast.String); ok {
			cols = append(cols, s.Str)
		}
	}
	return cols
}

// omniColumnConstraints iterates over a ColumnDef's constraint list.
func omniColumnConstraints(col *ast.ColumnDef) []*ast.Constraint {
	if col == nil || col.Constraints == nil {
		return nil
	}
	var result []*ast.Constraint
	for _, item := range col.Constraints.Items {
		if c, ok := item.(*ast.Constraint); ok {
			result = append(result, c)
		}
	}
	return result
}

// omniTableElements iterates over a CreateStmt's table elements,
// returning column defs and table constraints separately.
func omniTableElements(create *ast.CreateStmt) (cols []*ast.ColumnDef, cons []*ast.Constraint) {
	if create.TableElts == nil {
		return
	}
	for _, item := range create.TableElts.Items {
		switch n := item.(type) {
		case *ast.ColumnDef:
			cols = append(cols, n)
		case *ast.Constraint:
			cons = append(cons, n)
		}
	}
	return
}

// omniAlterTableCmds extracts AlterTableCmd items from an AlterTableStmt.
func omniAlterTableCmds(alter *ast.AlterTableStmt) []*ast.AlterTableCmd {
	if alter.Cmds == nil {
		return nil
	}
	var cmds []*ast.AlterTableCmd
	for _, item := range alter.Cmds.Items {
		if cmd, ok := item.(*ast.AlterTableCmd); ok {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

// omniIsRoleOrSearchPathSet checks if a VariableSetStmt is SET ROLE or SET search_path.
func omniIsRoleOrSearchPathSet(stmt *ast.VariableSetStmt) bool {
	if stmt == nil {
		return false
	}
	name := strings.ToLower(stmt.Name)
	return name == "role" || name == "search_path" ||
		stmt.Kind == ast.VAR_SET_ROLE
}

// omniTypeName extracts the type name string from a TypeName node.
func omniTypeName(tn *ast.TypeName) string {
	if tn == nil || tn.Names == nil {
		return ""
	}
	var parts []string
	for _, item := range tn.Names.Items {
		if s, ok := item.(*ast.String); ok {
			parts = append(parts, s.Str)
		}
	}
	// Return last part (skip "pg_catalog" prefix)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// omniDropObjects extracts object names from a DropStmt.
// Returns list of (schema, name) pairs.
func omniDropObjects(drop *ast.DropStmt) [][2]string {
	if drop.Objects == nil {
		return nil
	}
	var result [][2]string
	for _, item := range drop.Objects.Items {
		list, ok := item.(*ast.List)
		if !ok {
			continue
		}
		var parts []string
		for _, nameItem := range list.Items {
			if s, ok := nameItem.(*ast.String); ok {
				parts = append(parts, s.Str)
			}
		}
		switch len(parts) {
		case 1:
			result = append(result, [2]string{"public", parts[0]})
		case 2:
			result = append(result, [2]string{parts[0], parts[1]})
		}
	}
	return result
}
```

**Step 4: Verify compilation**

Run: `cd bytebase && go build ./backend/plugin/advisor/pg/...`
Expected: Compiles successfully

**Step 5: Commit**

```bash
git add bytebase/backend/plugin/advisor/pg/generic_checker_omni.go \
        bytebase/backend/plugin/advisor/pg/utils_omni.go \
        bytebase/backend/plugin/parser/pg/omni.go
git commit -m "feat(advisor/pg): add omni AST infrastructure for SQL review migration"
```

---

## Phase 2: Migrate Advisors

Each advisor migration follows the same pattern:
1. Add an `omniCheck()` method (or equivalent omni logic) to the advisor
2. In `Check()`, try omni first via `pgparser.GetOmniNode()`; fall back to ANTLR
3. Run tests to verify output parity

### Advisor Pattern Categories

Advisors fall into 5 patterns. We migrate one representative advisor per pattern first, then batch the rest.

---

### Task 2: Pattern A - Simple top-level matcher (NAMING_TABLE)

**Files:**
- Modify: `bytebase/backend/plugin/advisor/pg/advisor_naming_table.go`

This is the template for ~20 simple advisors that match on 1-2 top-level statement types.

**Step 1: Add omni handler methods to the rule struct**

Add these methods to `namingTableConventionRule`:

```go
func (r *namingTableConventionRule) checkOmniCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}
	r.checkTableName(tableName, 1) // line 1 = start of this statement
}

func (r *namingTableConventionRule) checkOmniRenameStmt(n *ast.RenameStmt) {
	if n.RenameType != ast.OBJECT_TABLE {
		return
	}
	r.checkTableName(n.Newname, 1)
}
```

**Step 2: Modify `checkTableName` to accept line number instead of antlr context**

Change signature from `checkTableName(tableName string, ctx antlr.ParserRuleContext)` to accept a line:

```go
func (r *namingTableConventionRule) checkTableName(tableName string, line int32) {
	if !r.format.MatchString(tableName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingTableConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, r.format),
			StartPosition: &storepb.Position{Line: line, Column: 0},
		})
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingTableConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			StartPosition: &storepb.Position{Line: line, Column: 0},
		})
	}
}
```

**Step 3: Update Check() to use omni**

Replace the statement loop in `Check()`:

```go
for _, stmt := range checkCtx.ParsedStatements {
	if stmt.AST == nil {
		continue
	}
	node, ok := pgparser.GetOmniNode(stmt.AST)
	if !ok {
		continue
	}
	rule.SetBaseLine(stmt.BaseLine())

	switch n := node.(type) {
	case *ast.CreateStmt:
		rule.checkOmniCreateStmt(n)
	case *ast.RenameStmt:
		rule.checkOmniRenameStmt(n)
	}
}
return rule.GetAdviceList(), nil
```

Now the rule struct embeds `OmniBaseRule` instead of `BaseRule`, remove `GenericChecker`, and remove ANTLR imports.

**Step 4: Run tests**

Run: `cd bytebase && go test -v -count=1 ./backend/plugin/advisor/pg/ -run TestPostgreSQLRules/NAMING_TABLE`
Expected: PASS

**Step 5: Commit**

```bash
git add bytebase/backend/plugin/advisor/pg/advisor_naming_table.go
git commit -m "feat(advisor/pg): migrate NAMING_TABLE to omni AST"
```

---

### Task 3: Pattern B - Stateful cross-statement (INDEX_CREATE_CONCURRENTLY)

**Files:**
- Modify: `bytebase/backend/plugin/advisor/pg/advisor_index_create_concurrently.go`

This is the template for stateful advisors that track state across multiple statements.

**Step 1: Switch to OmniBaseRule and add omni handlers**

```go
type indexCreateConcurrentlyRule struct {
	OmniBaseRule
	newlyCreatedTables map[string]bool
}

func (r *indexCreateConcurrentlyRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		tableName := omniTableName(n.Relation)
		if tableName != "" {
			r.newlyCreatedTables[tableName] = true
		}
	case *ast.IndexStmt:
		r.handleOmniIndexStmt(n)
	case *ast.DropStmt:
		r.handleOmniDropStmt(n)
	}
}

func (r *indexCreateConcurrentlyRule) handleOmniIndexStmt(n *ast.IndexStmt) {
	if n.Concurrent {
		return
	}
	tableName := omniTableName(n.Relation)
	if r.newlyCreatedTables[tableName] {
		return
	}
	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.CreateIndexUnconcurrently.Int32(),
		Title:   r.title,
		Content: "Creating indexes will block writes on the table, unless use CONCURRENTLY",
		StartPosition: &storepb.Position{Line: 1, Column: 0},
	})
}

func (r *indexCreateConcurrentlyRule) handleOmniDropStmt(n *ast.DropStmt) {
	if n.RemoveType != ast.OBJECT_INDEX {
		return
	}
	if !n.Concurrent {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.DropIndexUnconcurrently.Int32(),
			Title:   r.title,
			Content: "Droping indexes will block writes on the table, unless use CONCURRENTLY",
			StartPosition: &storepb.Position{Line: 1, Column: 0},
		})
	}
}
```

**Step 2: Update Check() to use RunOmniRules**

```go
func (*IndexConcurrentlyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	rule := &indexCreateConcurrentlyRule{
		OmniBaseRule:       OmniBaseRule{level: level, title: checkCtx.Rule.Type.String()},
		newlyCreatedTables: make(map[string]bool),
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}
```

**Step 3: Run tests**

Run: `cd bytebase && go test -v -count=1 ./backend/plugin/advisor/pg/ -run TestPostgreSQLRules/INDEX_CREATE_CONCURRENTLY`
Expected: PASS

**Step 4: Commit**

---

### Task 4: Pattern C - Deep expression walking (STATEMENT_WHERE_REQUIRE_SELECT)

**Files:**
- Modify: `bytebase/backend/plugin/advisor/pg/advisor_statement_where_required_select.go`

This is the template for advisors that inspect inside expressions.

**Step 1: Add omni handler**

With omni, `SelectStmt` directly has `WhereClause` and `FromClause` fields. No deep walking needed - this is simpler than ANTLR!

```go
func (r *statementWhereRequiredSelectRule) checkOmniSelect(n *ast.SelectStmt, stmtText string) {
	// Set operations (UNION etc) - check each side
	if n.Op != ast.SETOP_NONE {
		if n.Larg != nil {
			r.checkOmniSelect(n.Larg, stmtText)
		}
		if n.Rarg != nil {
			r.checkOmniSelect(n.Rarg, stmtText)
		}
		return
	}

	// No FROM = no WHERE needed (e.g., SELECT 1)
	if n.FromClause == nil || len(n.FromClause.Items) == 0 {
		return
	}

	if n.WhereClause != nil {
		return
	}

	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.StatementNoWhere.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
		StartPosition: &storepb.Position{Line: 1, Column: 0},
	})
}
```

**Step 2: Update Check()**

```go
for _, stmtInfo := range checkCtx.ParsedStatements {
	if stmtInfo.AST == nil {
		continue
	}
	node, ok := pgparser.GetOmniNode(stmtInfo.AST)
	if !ok {
		continue
	}
	rule := &statementWhereRequiredSelectRule{
		OmniBaseRule: OmniBaseRule{
			level: level, title: checkCtx.Rule.Type.String(),
		},
	}
	rule.SetStatement(stmtInfo.BaseLine(), stmtInfo.Text)

	if sel, ok := node.(*ast.SelectStmt); ok {
		rule.checkOmniSelect(sel, strings.TrimSpace(stmtInfo.Text))
	}
	adviceList = append(adviceList, rule.GetAdviceList()...)
}
```

Note: `SelectStmt` fields `FromClause` and `WhereClause` directly tell us what we need - no grammar rule traversal required.

**Step 3: Run tests, commit**

---

### Task 5: Pattern D - Complex stateful with metadata (COLUMN_NO_NULL)

**Files:**
- Modify: `bytebase/backend/plugin/advisor/pg/advisor_column_no_null.go`

This is the template for advisors that track column state across CREATE/ALTER and use metadata.

**Step 1: Add omni handlers**

```go
func (r *columnNoNullRule) onOmniStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleOmniCreate(n)
	case *ast.AlterTableStmt:
		r.handleOmniAlter(n)
	}
}

func (r *columnNoNullRule) handleOmniCreate(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}
	schema := omniSchemaName(n.Relation)

	cols, tableCons := omniTableElements(n)
	for _, col := range cols {
		r.addColumn(schema, tableName, col.Colname, int(r.LocToLine(col.Loc)))

		// Check column-level constraints
		if col.IsNotNull {
			r.removeColumn(schema, tableName, col.Colname)
			continue
		}
		for _, con := range omniColumnConstraints(col) {
			if con.Contype == ast.CONSTR_PRIMARY || con.Contype == ast.CONSTR_NOTNULL {
				r.removeColumn(schema, tableName, col.Colname)
				break
			}
		}
	}

	// Check table-level constraints (PRIMARY KEY)
	for _, con := range tableCons {
		if con.Contype == ast.CONSTR_PRIMARY {
			for _, colName := range omniConstraintColumns(con) {
				r.removeColumn(schema, tableName, colName)
			}
		}
	}
}

func (r *columnNoNullRule) handleOmniAlter(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	schema := omniSchemaName(n.Relation)

	for _, cmd := range omniAlterTableCmds(n) {
		switch ast.AlterTableType(cmd.Subtype) {
		case ast.AT_AddColumn:
			if colDef, ok := cmd.Def.(*ast.ColumnDef); ok {
				r.addColumn(schema, tableName, colDef.Colname, int(r.LocToLine(colDef.Loc)))
				if colDef.IsNotNull {
					r.removeColumn(schema, tableName, colDef.Colname)
				}
				for _, con := range omniColumnConstraints(colDef) {
					if con.Contype == ast.CONSTR_PRIMARY || con.Contype == ast.CONSTR_NOTNULL {
						r.removeColumn(schema, tableName, colDef.Colname)
						break
					}
				}
			}
		case ast.AT_SetNotNull:
			r.removeColumn(schema, tableName, cmd.Name)
		case ast.AT_DropNotNull:
			r.addColumn(schema, tableName, cmd.Name, 1)
		case ast.AT_AddConstraint:
			if con, ok := cmd.Def.(*ast.Constraint); ok && con.Contype == ast.CONSTR_PRIMARY {
				for _, colName := range omniConstraintColumns(con) {
					r.removeColumn(schema, tableName, colName)
				}
			}
		}
	}
}
```

**Step 2: Update Check(), run tests, commit**

---

### Task 6: Pattern E - Database-driven (STATEMENT_DML_DRY_RUN)

**Files:**
- Modify: `bytebase/backend/plugin/advisor/pg/advisor_statement_dml_dry_run.go`

This advisor uses `stmtInfo.Text` for EXPLAIN. With omni, we get `OmniAST.Text` directly.

**Step 1: Add omni handler**

```go
func (r *statementDMLDryRunRule) onOmniStatement(node ast.Node, stmtText string) {
	switch node.(type) {
	case *ast.VariableSetStmt:
		n := node.(*ast.VariableSetStmt)
		if omniIsRoleOrSearchPathSet(n) {
			r.preExecutions = append(r.preExecutions, strings.TrimSpace(stmtText))
		}
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
		r.checkDMLDryRunOmni(strings.TrimSpace(stmtText))
	}
}

func (r *statementDMLDryRunRule) checkDMLDryRunOmni(stmtText string) {
	if r.explainCount >= common.MaximumLintExplainSize {
		return
	}
	r.explainCount++

	_, err := advisor.Query(r.ctx, advisor.QueryContext{
		TenantMode:    r.tenantMode,
		PreExecutions: r.preExecutions,
	}, r.driver, storepb.Engine_POSTGRES, fmt.Sprintf("EXPLAIN %s", stmtText))

	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementDMLDryRunFailed.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("\"%s\" dry runs failed: %s", stmtText, err.Error()),
			StartPosition: &storepb.Position{Line: 1, Column: 0},
		})
	}
}
```

**Step 2: Update Check(), run tests, commit**

---

### Task 7-15: Batch migrate remaining advisors

Migrate all remaining advisors following the patterns established above. Group by pattern:

#### Task 7: Naming advisors (Pattern A)

Migrate these using the same pattern as NAMING_TABLE:

| File | Rule | Key omni types |
|---|---|---|
| `advisor_naming_column.go` | NAMING_COLUMN | `*ast.CreateStmt` → cols, `*ast.AlterTableStmt` → AT_AddColumn, AT_AlterColumnType |
| `advisor_naming_index_convention.go` | NAMING_INDEX_IDX | `*ast.IndexStmt` → `.Idxname`, `.Relation`, `.IndexParams` |
| `advisor_naming_primary_key_convention.go` | NAMING_INDEX_PK | `*ast.CreateStmt` → constraints, `*ast.AlterTableStmt` → AT_AddConstraint |
| `advisor_naming_unique_key_convention.go` | NAMING_INDEX_UK | Same as PK |
| `advisor_naming_foreign_key_convention.go` | NAMING_INDEX_FK | `*ast.Constraint{Contype: CONSTR_FOREIGN}` → `.Pktable`, `.FkAttrs`, `.PkAttrs` |
| `advisor_naming_fully_qualified.go` | NAMING_FULLY_QUALIFIED | Check `RangeVar.Schemaname` is not empty |

Run: `cd bytebase && go test -v -count=1 ./backend/plugin/advisor/pg/ -run "TestPostgreSQLRules/(NAMING_COLUMN|NAMING_INDEX_IDX|NAMING_INDEX_PK|NAMING_INDEX_UK|NAMING_INDEX_FK|NAMING_FULLY_QUALIFIED)"`

Commit after all naming rules pass.

#### Task 8: Column advisors (Pattern A/D)

| File | Rule | Pattern | Key omni types |
|---|---|---|---|
| `advisor_column_require_default.go` | COLUMN_REQUIRE_DEFAULT | A | `*ast.ColumnDef` → `.RawDefault`, `.Constraints` |
| `advisor_column_required.go` | COLUMN_REQUIRED | A | `*ast.CreateStmt` → check required column names present |
| `advisor_column_disallow_changing_type.go` | COLUMN_DISALLOW_CHANGE_TYPE | A | `*ast.AlterTableCmd{Subtype: AT_AlterColumnType}` |
| `advisor_column_default_disallow_volatile.go` | COLUMN_DEFAULT_DISALLOW_VOLATILE | A | `*ast.ColumnDef` → inspect `.RawDefault` for volatile functions |
| `advisor_column_maximum_character_length.go` | COLUMN_MAXIMUM_CHARACTER_LENGTH | A | `*ast.ColumnDef.TypeName.Typmods` for length |
| `advisor_column_type_disallow_list.go` | COLUMN_TYPE_DISALLOW_LIST | A | `omniTypeName(col.TypeName)` |

Run tests, commit.

#### Task 9: Comment/encoding advisors (Pattern A)

| File | Rule | Key omni types |
|---|---|---|
| `advisor_comment_convention.go` | SYSTEM_COMMENT_LENGTH | `*ast.CommentStmt` |
| `advisor_column_comment_convention.go` | COLUMN_COMMENT | `*ast.CommentStmt` for columns |
| `advisor_table_comment_convention.go` | TABLE_COMMENT | `*ast.CommentStmt` for tables |
| `advisor_encoding_allowlist.go` | SYSTEM_CHARSET_ALLOWLIST | builtin, may not need AST |
| `advisor_collation_allowlist.go` | SYSTEM_COLLATION_ALLOWLIST | builtin, may not need AST |

Run tests, commit.

#### Task 10: Index advisors (Pattern A/B)

| File | Rule | Key omni types |
|---|---|---|
| `advisor_index_key_number_limit.go` | INDEX_KEY_NUMBER_LIMIT | `*ast.IndexStmt` → `len(IndexParams.Items)` |
| `advisor_index_no_duplicate_column.go` | INDEX_NO_DUPLICATE_COLUMN | `*ast.IndexStmt` → check unique `.IndexParams` |
| `advisor_index_primary_key_type_allowlist.go` | INDEX_PRIMARY_KEY_TYPE_ALLOWLIST | `*ast.Constraint{CONSTR_PRIMARY}` + metadata |
| `advisor_index_total_number_limit.go` | INDEX_TOTAL_NUMBER_LIMIT | `*ast.IndexStmt` + metadata count |

Run tests, commit.

#### Task 11: Table advisors (Pattern A/D)

| File | Rule | Key omni types |
|---|---|---|
| `advisor_table_require_pk.go` | TABLE_REQUIRE_PK | Pattern D - track tables, validate against `FinalMetadata` |
| `advisor_table_no_fk.go` | TABLE_NO_FOREIGN_KEY | `*ast.Constraint{CONSTR_FOREIGN}` |
| `advisor_table_disallow_partition.go` | TABLE_DISALLOW_PARTITION | `*ast.CreateStmt.Partspec != nil` |
| `advisor_table_drop_naming_convention.go` | TABLE_DROP_NAMING_CONVENTION | `*ast.DropStmt{OBJECT_TABLE}` → name matching |

Run tests, commit.

#### Task 12: Statement DML advisors (Pattern A/C)

| File | Rule | Key omni types |
|---|---|---|
| `advisor_statement_where_required_update_delete.go` | STATEMENT_WHERE_REQUIRE_UPDATE_DELETE | `*ast.UpdateStmt.WhereClause`, `*ast.DeleteStmt.WhereClause` |
| `advisor_statement_no_select_all.go` | STATEMENT_SELECT_NO_SELECT_ALL | `*ast.SelectStmt.TargetList` → check for `*ast.ColumnRef` with `*ast.A_Star` |
| `advisor_statement_no_leading_wildcard_like.go` | STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE | `ast.Inspect` for `*ast.A_Expr{Kind: AEXPR_LIKE}` → check left operand |
| `advisor_statement_maximum_limit_value.go` | STATEMENT_MAXIMUM_LIMIT_VALUE | `*ast.SelectStmt.LimitCount` → extract integer |
| `advisor_insert_must_specify_column.go` | STATEMENT_INSERT_MUST_SPECIFY_COLUMN | `*ast.InsertStmt.Cols == nil` |
| `advisor_insert_row_limit.go` | STATEMENT_INSERT_ROW_LIMIT | `*ast.InsertStmt` → count VALUES rows |
| `advisor_insert_disallow_order_by_rand.go` | STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND | `*ast.InsertStmt.SelectStmt` → inspect sort clause |
| `advisor_statement_affected_row_limit.go` | STATEMENT_AFFECTED_ROW_LIMIT | Database-driven (Pattern E) |
| `advisor_statement_merge_alter_table.go` | STATEMENT_MERGE_ALTER_TABLE | Track consecutive ALTER TABLE on same table |

Run tests, commit.

#### Task 13: Statement DDL advisors (Pattern A)

| File | Rule | Key omni types |
|---|---|---|
| `advisor_statement_add_check_not_valid.go` | STATEMENT_ADD_CHECK_NOT_VALID | `*ast.AlterTableCmd{AT_AddConstraint}` → `*ast.Constraint{CONSTR_CHECK}` → `.SkipValidation` |
| `advisor_statement_add_fk_not_valid.go` | STATEMENT_ADD_FOREIGN_KEY_NOT_VALID | Same pattern with `CONSTR_FOREIGN` |
| `advisor_statement_check_set_role_variable.go` | STATEMENT_CHECK_SET_ROLE_VARIABLE | `*ast.VariableSetStmt` → check role/search_path |
| `advisor_statement_create_specify_schema.go` | STATEMENT_CREATE_SPECIFY_SCHEMA | `*ast.CreateStmt.Relation.Schemaname == ""` |
| `advisor_statement_disallow_add_column_with_default.go` | STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT | `AT_AddColumn` → `ColumnDef.RawDefault != nil` |
| `advisor_statement_disallow_add_not_null.go` | STATEMENT_DISALLOW_ADD_NOT_NULL | `AT_SetNotNull` |
| `advisor_statement_disallow_commit.go` | STATEMENT_DISALLOW_COMMIT | `*ast.TransactionStmt{Kind: TRANS_STMT_COMMIT}` |
| `advisor_statement_disallow_on_del_cascade.go` | STATEMENT_DISALLOW_ON_DEL_CASCADE | `*ast.Constraint{CONSTR_FOREIGN}` → `.FkDelaction == 'c'` |
| `advisor_statement_disallow_rm_tbl_cascade.go` | STATEMENT_DISALLOW_RM_TBL_CASCADE | `*ast.DropStmt{Behavior: DROP_CASCADE}` |
| `advisor_statement_non_transactional.go` | STATEMENT_NON_TRANSACTIONAL | Check for transaction wrapper |

Run tests, commit.

#### Task 14: Special advisors

| File | Rule | Notes |
|---|---|---|
| `advisor_statement_object_owner_check.go` | STATEMENT_OBJECT_OWNER_CHECK | Database-driven, check SET ROLE |
| `advisor_migration_compatibility.go` | SCHEMA_BACKWARD_COMPATIBILITY | Uses metadata diff, may not need AST changes |
| `advisor_builtin_prior_backup_check.go` | BUILTIN_PRIOR_BACKUP_CHECK | May not use AST at all |
| `advisor_builtin_walk_through_check.go` | BUILTIN_WALK_THROUGH_CHECK | Already uses omni catalog |
| `advisor_statement_dml_dry_run.go` | STATEMENT_DML_DRY_RUN | Pattern E (done in Task 6) |

Run tests, commit.

#### Task 15: Run full test suite

Run: `cd bytebase && go test -v -count=1 ./backend/plugin/advisor/pg/ -run TestPostgreSQLRules`
Expected: ALL tests pass

Commit if any final fixes needed.

---

## Phase 3: Switch and Cleanup

### Task 16: Remove ANTLR parsing from parsePgStatements

**Files:**
- Modify: `bytebase/backend/plugin/parser/pg/pg.go`

**Step 1: Remove ANTLR parsing call**

In `parsePgStatements`, remove:
- The call to `parseSinglePostgreSQL()`
- The `antlrResult`/`antlrErr` variables
- The `antlrAST` assignment on `OmniAST`
- The ANTLR-only fallback branch

The function simplifies to:
```go
func parsePgStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}
		omniStmts, omniErr := ParsePg(stmt.Text)
		if omniErr != nil {
			return nil, convertOmniError(omniErr, statement, stmt)
		}
		for _, os := range omniStmts {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &OmniAST{
					Node:          os.AST,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}
	return result, nil
}
```

**Step 2: Remove `antlrAST` field from OmniAST**

In `omni.go`, remove:
- The `antlrAST` field
- The `AsANTLRAST()` method
- Import of `antlr` packages

**Step 3: Run full test suite**

Run: `cd bytebase && go test -v -count=1 ./backend/plugin/advisor/pg/...`
Expected: PASS

**Step 4: Commit**

```bash
git commit -m "feat(parser/pg): remove ANTLR parsing path, omni-only for PostgreSQL"
```

### Task 17: Remove old ANTLR infrastructure

**Files:**
- Modify: `bytebase/backend/plugin/advisor/pg/generic_checker.go` - Remove old `Rule` interface, `GenericChecker`, `BaseRule`
- Modify: `bytebase/backend/plugin/advisor/pg/utils.go` - Remove ANTLR-specific utilities (`isTopLevel`, `getTextFromTokens`, `extractTableName` that use ANTLR types)
- Remove any remaining ANTLR imports from advisor files

**Step 1: Clean up `generic_checker.go`**

Remove the old `Rule`, `GenericChecker`, `BaseRule` types. Rename `generic_checker_omni.go` contents to `generic_checker.go` (or merge).

**Step 2: Clean up `utils.go`**

Remove functions that reference ANTLR types:
- `isTopLevel` (uses `antlr.Tree`)
- `getTextFromTokens` (uses `antlr.CommonTokenStream`)
- `extractTableName` / `extractSchemaName` (uses `parser.IQualified_nameContext`)
- `extractIntegerConstant` / `extractStringConstant` (uses parser types)
- `appendSessionPreExecutionStatements` (uses parser types)

Keep `getTemplateRegexp` and `normalizeSchemaName` (no ANTLR dependency).

**Step 3: Verify no ANTLR imports remain**

Run: `grep -r "github.com/bytebase/parser/postgresql" bytebase/backend/plugin/advisor/pg/`
Expected: No matches (or only in SDL files if they still use ANTLR)

Run: `grep -r "antlr4-go/antlr" bytebase/backend/plugin/advisor/pg/`
Expected: No matches

**Step 4: Run full test suite**

Run: `cd bytebase && go test -v -count=1 ./backend/plugin/advisor/pg/...`
Expected: PASS

**Step 5: Build**

Run: `cd bytebase && go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Successful build

**Step 6: Commit**

```bash
git commit -m "refactor(advisor/pg): remove ANTLR infrastructure after omni migration"
```

---

## Key Risks and Mitigations

1. **Position mismatch**: ANTLR lines are 1-based within statement, omni `Loc` is byte offset. Mitigation: `ByteOffsetToRunePosition()` + thorough test comparison.

2. **Omni parser strictness**: Omni may reject SQL that ANTLR accepts (reserved keywords as identifiers). Mitigation: Fix in omni parser if needed; existing ANTLR fallback catches these during migration.

3. **`isTopLevel` equivalent**: ANTLR advisors filter nested contexts. With omni, `GetOmniNode()` returns the top-level statement node directly, so nested filtering is unnecessary for most advisors. For advisors that use `ast.Inspect` to walk deeper, nested nodes are expected.

4. **SDL files**: The 3 SDL files (`sdl_check.go`, `sdl_integrity_check.go`, `sdl_drop_check.go`) may use different patterns. Evaluate separately.

5. **Test data**: Existing YAML test files are the source of truth. Any output difference is a bug in the migration.
