# Unified Statement Type Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Unify `SingleSQL` and `AST` into a single `Statement` type that contains both text and parsed tree.

**Architecture:** Introduce a new `Statement` type that combines fields from `SingleSQL` (text, positions, byte offsets) with the `AST` interface. The new `Parse()` function will return `[]Statement` with both text and AST populated. Existing `SplitMultiSQL()` and old `Parse()` remain unchanged during migration.

**Tech Stack:** Go, ANTLR4

---

## Phase 1: Introduce Statement Type

### Task 1: Create Statement type

**Files:**
- Create: `backend/plugin/parser/base/statement.go`

**Step 1: Create the Statement struct**

```go
package base

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Statement represents a single SQL statement with both text and parsed AST.
// This is the unified type that combines SingleSQL (text-based) and AST (tree-based).
type Statement struct {
	// Text content
	Text  string
	Empty bool

	// Position tracking (1-based)
	StartPosition *storepb.Position
	EndPosition   *storepb.Position

	// Byte offsets for execution tracking
	ByteOffsetStart int
	ByteOffsetEnd   int

	// Parsed tree (always present after Parse)
	AST AST
}
```

**Step 2: Verify file compiles**

Run: `go build ./backend/plugin/parser/base/...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add backend/plugin/parser/base/statement.go
git commit -m "feat(parser): add Statement type for unified SQL representation"
```

---

### Task 2: Add Statement helper functions

**Files:**
- Modify: `backend/plugin/parser/base/statement.go`

**Step 1: Add FilterEmptyStatements function**

```go
// FilterEmptyStatements removes empty statements from the list.
func FilterEmptyStatements(list []Statement) []Statement {
	var result []Statement
	for _, stmt := range list {
		if !stmt.Empty {
			result = append(result, stmt)
		}
	}
	return result
}

// FilterEmptyStatementsWithIndexes removes empty statements and returns original indexes.
func FilterEmptyStatementsWithIndexes(list []Statement) ([]Statement, []int32) {
	var result []Statement
	var originalIndex []int32
	for i, stmt := range list {
		if !stmt.Empty {
			result = append(result, stmt)
			originalIndex = append(originalIndex, int32(i))
		}
	}
	return result, originalIndex
}
```

**Step 2: Verify file compiles**

Run: `go build ./backend/plugin/parser/base/...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add backend/plugin/parser/base/statement.go
git commit -m "feat(parser): add Statement filter helper functions"
```

---

### Task 3: Add Statement conversion helpers

**Files:**
- Modify: `backend/plugin/parser/base/statement.go`

**Step 1: Add SingleSQLToStatement conversion**

```go
// SingleSQLToStatement converts a SingleSQL to a Statement without AST.
// The AST field will be nil. Use this for incremental migration.
func SingleSQLToStatement(sql SingleSQL) Statement {
	return Statement{
		Text:            sql.Text,
		Empty:           sql.Empty,
		StartPosition:   sql.Start,
		EndPosition:     sql.End,
		ByteOffsetStart: sql.ByteOffsetStart,
		ByteOffsetEnd:   sql.ByteOffsetEnd,
		AST:             nil,
	}
}

// SingleSQLsToStatements converts a slice of SingleSQL to Statements without AST.
func SingleSQLsToStatements(sqls []SingleSQL) []Statement {
	result := make([]Statement, len(sqls))
	for i, sql := range sqls {
		result[i] = SingleSQLToStatement(sql)
	}
	return result
}

// StatementToSingleSQL converts a Statement back to SingleSQL.
// The AST is discarded. Use this for backward compatibility.
func StatementToSingleSQL(stmt Statement) SingleSQL {
	return SingleSQL{
		Text:            stmt.Text,
		Empty:           stmt.Empty,
		Start:           stmt.StartPosition,
		End:             stmt.EndPosition,
		ByteOffsetStart: stmt.ByteOffsetStart,
		ByteOffsetEnd:   stmt.ByteOffsetEnd,
		// Note: BaseLine is not preserved as Statement uses 1-based StartPosition
	}
}
```

**Step 2: Verify file compiles**

Run: `go build ./backend/plugin/parser/base/...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add backend/plugin/parser/base/statement.go
git commit -m "feat(parser): add Statement conversion helpers for migration"
```

---

### Task 4: Add tests for Statement type

**Files:**
- Create: `backend/plugin/parser/base/statement_test.go`

**Step 1: Write tests**

```go
package base

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestFilterEmptyStatements(t *testing.T) {
	statements := []Statement{
		{Text: "SELECT 1", Empty: false},
		{Text: "", Empty: true},
		{Text: "SELECT 2", Empty: false},
		{Text: "-- comment", Empty: true},
	}

	result := FilterEmptyStatements(statements)

	require.Len(t, result, 2)
	require.Equal(t, "SELECT 1", result[0].Text)
	require.Equal(t, "SELECT 2", result[1].Text)
}

func TestFilterEmptyStatementsWithIndexes(t *testing.T) {
	statements := []Statement{
		{Text: "SELECT 1", Empty: false},
		{Text: "", Empty: true},
		{Text: "SELECT 2", Empty: false},
	}

	result, indexes := FilterEmptyStatementsWithIndexes(statements)

	require.Len(t, result, 2)
	require.Equal(t, []int32{0, 2}, indexes)
}

func TestSingleSQLToStatement(t *testing.T) {
	sql := SingleSQL{
		Text:            "SELECT 1",
		Empty:           false,
		Start:           &storepb.Position{Line: 1, Column: 1},
		End:             &storepb.Position{Line: 1, Column: 9},
		ByteOffsetStart: 0,
		ByteOffsetEnd:   8,
	}

	stmt := SingleSQLToStatement(sql)

	require.Equal(t, sql.Text, stmt.Text)
	require.Equal(t, sql.Empty, stmt.Empty)
	require.Equal(t, sql.Start, stmt.StartPosition)
	require.Equal(t, sql.End, stmt.EndPosition)
	require.Equal(t, sql.ByteOffsetStart, stmt.ByteOffsetStart)
	require.Equal(t, sql.ByteOffsetEnd, stmt.ByteOffsetEnd)
	require.Nil(t, stmt.AST)
}

func TestStatementToSingleSQL(t *testing.T) {
	stmt := Statement{
		Text:            "SELECT 1",
		Empty:           false,
		StartPosition:   &storepb.Position{Line: 1, Column: 1},
		EndPosition:     &storepb.Position{Line: 1, Column: 9},
		ByteOffsetStart: 0,
		ByteOffsetEnd:   8,
	}

	sql := StatementToSingleSQL(stmt)

	require.Equal(t, stmt.Text, sql.Text)
	require.Equal(t, stmt.Empty, sql.Empty)
	require.Equal(t, stmt.StartPosition, sql.Start)
	require.Equal(t, stmt.EndPosition, sql.End)
	require.Equal(t, stmt.ByteOffsetStart, sql.ByteOffsetStart)
	require.Equal(t, stmt.ByteOffsetEnd, sql.ByteOffsetEnd)
}
```

**Step 2: Run tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/base -run ^Test.*Statement.*$`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add backend/plugin/parser/base/statement_test.go
git commit -m "test(parser): add tests for Statement type"
```

---

### Task 5: Register ParseStatements function type

**Files:**
- Modify: `backend/plugin/parser/base/interface.go`

**Step 1: Add ParseStatementsFunc type and registry**

Add after the existing `parsers` variable declaration (around line 31):

```go
	statementParsers = make(map[storepb.Engine]ParseStatementsFunc)
```

Add after the `ParseFunc` type (around line 52):

```go
// ParseStatementsFunc is the interface for parsing SQL statements and returning []Statement.
// This is the new unified parsing function that returns complete Statement objects.
type ParseStatementsFunc func(statement string) ([]Statement, error)
```

**Step 2: Add registration and lookup functions**

Add after the existing `Parse` function (around line 274):

```go
func RegisterParseStatementsFunc(engine storepb.Engine, f ParseStatementsFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := statementParsers[engine]; dup {
		panic(fmt.Sprintf("RegisterParseStatementsFunc called twice %s", engine))
	}
	statementParsers[engine] = f
}

// ParseStatements parses the SQL statement and returns Statement objects with both text and AST.
// This is the new unified parsing function.
func ParseStatements(engine storepb.Engine, statement string) ([]Statement, error) {
	f, ok := statementParsers[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported for ParseStatements", engine)
	}
	return f(statement)
}
```

**Step 3: Verify file compiles**

Run: `go build ./backend/plugin/parser/base/...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add backend/plugin/parser/base/interface.go
git commit -m "feat(parser): add ParseStatements registry for unified parsing"
```

---

## Phase 2: Implement ParseStatements for MySQL

### Task 6: Implement MySQL ParseStatements

**Files:**
- Modify: `backend/plugin/parser/mysql/mysql.go`

**Step 1: Add ParseStatements registration in init()**

Modify the `init()` function to also register ParseStatements:

```go
func init() {
	base.RegisterParseFunc(storepb.Engine_MYSQL, parseMySQLForRegistry)
	base.RegisterParseFunc(storepb.Engine_MARIADB, parseMySQLForRegistry)
	base.RegisterParseFunc(storepb.Engine_OCEANBASE, parseMySQLForRegistry)
	base.RegisterParseStatementsFunc(storepb.Engine_MYSQL, parseMySQLStatements)
	base.RegisterParseStatementsFunc(storepb.Engine_MARIADB, parseMySQLStatements)
	base.RegisterParseStatementsFunc(storepb.Engine_OCEANBASE, parseMySQLStatements)
}
```

**Step 2: Add parseMySQLStatements function**

Add after the `toAST` function:

```go
// parseMySQLStatements is the ParseStatementsFunc for MySQL, MariaDB, and OceanBase.
// Returns []Statement with both text and AST populated.
func parseMySQLStatements(statement string) ([]base.Statement, error) {
	// First split to get SingleSQL with text and positions
	singleSQLs, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	parseResults, err := ParseMySQL(statement)
	if err != nil {
		return nil, err
	}

	// Combine: SingleSQL provides text/positions, ParseResult provides AST
	// Note: parseResults may have fewer items if some statements are empty
	var statements []base.Statement
	astIndex := 0
	for _, sql := range singleSQLs {
		stmt := base.Statement{
			Text:            sql.Text,
			Empty:           sql.Empty,
			StartPosition:   sql.Start,
			EndPosition:     sql.End,
			ByteOffsetStart: sql.ByteOffsetStart,
			ByteOffsetEnd:   sql.ByteOffsetEnd,
		}
		if !sql.Empty && astIndex < len(parseResults) {
			stmt.AST = &base.ANTLRAST{
				StartPosition: &storepb.Position{Line: int32(parseResults[astIndex].BaseLine) + 1},
				Tree:          parseResults[astIndex].Tree,
				Tokens:        parseResults[astIndex].Tokens,
			}
			astIndex++
		}
		statements = append(statements, stmt)
	}

	return statements, nil
}
```

**Step 3: Verify file compiles**

Run: `go build ./backend/plugin/parser/mysql/...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add backend/plugin/parser/mysql/mysql.go
git commit -m "feat(parser/mysql): implement ParseStatements for unified parsing"
```

---

### Task 7: Add tests for MySQL ParseStatements

**Files:**
- Modify: `backend/plugin/parser/mysql/mysql_test.go` (or create if not exists)

**Step 1: Write test**

```go
func TestParseMySQLStatements(t *testing.T) {
	statement := "SELECT 1; SELECT 2;"

	statements, err := base.ParseStatements(storepb.Engine_MYSQL, statement)
	require.NoError(t, err)

	// Filter empty statements for assertion
	statements = base.FilterEmptyStatements(statements)

	require.Len(t, statements, 2)

	// Check first statement
	require.Equal(t, "SELECT 1;", statements[0].Text)
	require.False(t, statements[0].Empty)
	require.NotNil(t, statements[0].AST)
	require.NotNil(t, statements[0].StartPosition)

	// Check second statement
	require.Contains(t, statements[1].Text, "SELECT 2")
	require.False(t, statements[1].Empty)
	require.NotNil(t, statements[1].AST)
}
```

**Step 2: Run tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/parser/mysql -run ^TestParseMySQLStatements$`
Expected: PASS

**Step 3: Commit**

```bash
git add backend/plugin/parser/mysql/mysql_test.go
git commit -m "test(parser/mysql): add tests for ParseStatements"
```

---

## Phase 2 Continued: Implement ParseStatements for Other Engines

### Task 8: Implement PostgreSQL ParseStatements

**Files:**
- Modify: `backend/plugin/parser/pg/pg.go`

Follow the same pattern as Task 6 for MySQL:
1. Add registration in `init()`
2. Add `parsePgStatements` function
3. Combine SplitSQL results with Parse results

**Step 1: Add to init()**

```go
base.RegisterParseStatementsFunc(storepb.Engine_POSTGRES, parsePgStatements)
base.RegisterParseStatementsFunc(storepb.Engine_COCKROACHDB, parsePgStatements)
base.RegisterParseStatementsFunc(storepb.Engine_RISINGWAVE, parsePgStatements)
```

**Step 2: Add parsePgStatements function** (similar pattern to MySQL)

**Step 3: Verify and commit**

---

### Task 9-15: Implement ParseStatements for remaining engines

Repeat the pattern for:
- TiDB (`backend/plugin/parser/tidb/tidb.go`)
- MSSQL (`backend/plugin/parser/tsql/tsql.go`)
- Oracle (`backend/plugin/parser/plsql/plsql.go`)
- Snowflake (`backend/plugin/parser/snowflake/snowflake.go`)
- Redshift (`backend/plugin/parser/redshift/redshift.go`)
- Other engines as needed

---

## Phase 3: Migrate One Consumer (Example)

### Task 16: Migrate sheet.GetASTsForChecks to use ParseStatements

**Files:**
- Modify: `backend/component/sheet/sheet.go`

This task shows the pattern for migrating consumers. The sheet manager currently:
1. Calls `base.Parse()` to get `[]AST`

After migration:
1. Call `base.ParseStatements()` to get `[]Statement`
2. Access `stmt.AST` when needed

**Note:** This is an example task. Full consumer migration would be a separate plan after Phase 1-2 are stable.

---

## Verification

### Final Task: Run full test suite

**Step 1: Run parser tests**

Run: `go test -v -count=1 ./backend/plugin/parser/...`
Expected: All tests PASS

**Step 2: Run linter**

Run: `golangci-lint run --allow-parallel-runners ./backend/plugin/parser/...`
Expected: No new issues

**Step 3: Build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds
