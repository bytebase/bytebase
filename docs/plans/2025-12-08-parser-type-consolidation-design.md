# Parser Type Consolidation Design

## Problem Statement

The parser infrastructure has redundant types that represent similar concepts:

| Current Type | Purpose | Usage |
|--------------|---------|-------|
| `SingleSQL` | Result of splitting SQL | 245 occurrences (59 files) |
| `Statement` | Unified text + AST | 35 occurrences (12 files) |
| `ParseResult` | Parser output (Tree + Tokens + BaseLine) | 167 occurrences (37 files) |

### Issues

1. **Type redundancy**: `SingleSQL` and `Statement` are nearly identical
2. **Double splitting**: `parsePgStatements` calls `SplitSQL` twice (once directly, once via `ParsePostgreSQL`)
3. **Unclear contracts**: Current `Statement.AST` can be nil, requiring runtime checks
4. **Conversion overhead**: Multiple conversion functions exist (`SingleSQLToStatement`, `StatementToSingleSQL`, etc.)

## Design

### New Types

```go
// Statement is the result of splitting SQL (text + positions, no AST)
type Statement struct {
    Text            string
    Empty           bool
    BaseLine        int                // 0-based offset (to be addressed separately)
    StartPosition   *storepb.Position  // 1-based position
    EndPosition     *storepb.Position
    ByteOffsetStart int
    ByteOffsetEnd   int
}

// ParsedStatement is the result of parsing SQL (Statement + AST)
// AST is guaranteed to be non-nil after successful parsing
type ParsedStatement struct {
    Statement        // embedded - access fields directly
    AST       AST
}
```

### Design Rationale

- **Two distinct types**: `Statement` for split-only, `ParsedStatement` for parsed results
- **Type system enforcement**: If you have a `ParsedStatement`, AST is guaranteed present (no nil checks)
- **Embedding**: `ParsedStatement` embeds `Statement`, allowing direct field access (`p.Text`, `p.StartPosition`)
- **BaseLine kept**: The 0-based `BaseLine` field is retained for backward compatibility; the 0-based vs 1-based position confusion will be addressed in a separate change

### Interface Changes

```go
// Before
type SplitMultiSQLFunc func(string) ([]SingleSQL, error)
type ParseFunc func(string) ([]AST, error)
type ParseStatementsFunc func(string) ([]Statement, error)

// After
type SplitMultiSQLFunc func(string) ([]Statement, error)
type ParseStatementsFunc func(string) ([]ParsedStatement, error)
// ParseFunc deprecated - use ParseStatementsFunc + ExtractASTs
```

### Type Deprecation Plan

| Current Type | Action | Replacement |
|--------------|--------|-------------|
| `SingleSQL` | Deprecated, then removed | `Statement` |
| `ParseResult` | Deprecated, then removed | `ParsedStatement` |
| `Statement` (current) | Merged/aligned | `Statement` (new) |
| `AST` interface | Keep | - |
| `ANTLRAST` | Keep | - |

### Helper Functions

```go
// ExtractASTs extracts ASTs from ParsedStatements for backward compatibility
func ExtractASTs(stmts []ParsedStatement) []AST

// FilterEmptyStatements removes empty statements from the list
func FilterEmptyStatements(stmts []Statement) []Statement

// FilterEmptyParsedStatements removes empty parsed statements from the list
func FilterEmptyParsedStatements(stmts []ParsedStatement) []ParsedStatement
```

## Migration Plan

### Phase 1: Add New Types (Non-breaking)

1. Define `Statement` and `ParsedStatement` in `backend/plugin/parser/base/`
2. Add type alias for backward compatibility:
   ```go
   // Deprecated: use Statement instead
   type SingleSQL = Statement
   ```
3. Add helper functions (`ExtractASTs`, `FilterEmptyStatements`, etc.)
4. Existing code continues to work via type alias

### Phase 2: Update Splitters

1. Change `SplitMultiSQLFunc` signature to return `[]Statement`
2. Update each engine's `split.go` implementation:
   - `backend/plugin/parser/pg/split.go`
   - `backend/plugin/parser/mysql/split.go`
   - `backend/plugin/parser/tsql/split.go`
   - `backend/plugin/parser/plsql/split.go`
   - `backend/plugin/parser/tidb/split.go`
   - `backend/plugin/parser/snowflake/split.go`
   - `backend/plugin/parser/redshift/split.go`
   - `backend/plugin/parser/bigquery/split.go`
   - `backend/plugin/parser/trino/split.go`
   - `backend/plugin/parser/spanner/split.go`
   - `backend/plugin/parser/doris/split.go`
   - `backend/plugin/parser/cassandra/split.go`
   - `backend/plugin/parser/partiql/split.go`
   - `backend/plugin/parser/cosmosdb/split.go`
   - `backend/plugin/parser/standard/split.go`
3. Callers using `SingleSQL` still work via type alias

### Phase 3: Update Parsers

1. Change `ParseStatementsFunc` signature to return `[]ParsedStatement`
2. Update each engine's parser implementation:
   - `backend/plugin/parser/pg/pg.go`
   - `backend/plugin/parser/mysql/mysql.go`
   - `backend/plugin/parser/tsql/tsql.go`
   - `backend/plugin/parser/plsql/plsql.go`
   - `backend/plugin/parser/tidb/tidb.go`
   - `backend/plugin/parser/snowflake/snowflake.go`
   - `backend/plugin/parser/redshift/redshift.go`
   - `backend/plugin/parser/doris/doris.go`
   - `backend/plugin/parser/cassandra/cassandra.go`
   - `backend/plugin/parser/partiql/partiql.go`
   - `backend/plugin/parser/cockroachdb/cockroachdb.go`
3. Deprecate `ParseFunc`:
   - Add wrapper implementation that calls `ParseStatementsFunc` + `ExtractASTs`
   - Mark as deprecated in documentation
4. Remove `ParseResult` type (replaced by `ParsedStatement`)

### Phase 4: Migrate Callers

1. Replace `SingleSQL` references with `Statement` across codebase
2. Replace `ParseResult` usage with `ParsedStatement`
3. Update advisor code to use new types
4. Key directories to update:
   - `backend/plugin/advisor/` (heaviest usage of BaseLine)
   - `backend/plugin/db/` (execution code)
   - `backend/component/sheet/`

### Phase 5: Cleanup

1. Remove type alias `SingleSQL`
2. Remove `ParseResult` type completely
3. Remove `ParseFunc` and related registry code
4. Remove conversion functions:
   - `SingleSQLToStatement`
   - `SingleSQLsToStatements`
   - `StatementToSingleSQL`
5. Update tests

## Future Work

After this consolidation is complete, a separate change should address:

- **Position convention unification**: Standardize on 1-based positions throughout, potentially removing `BaseLine` or renaming it to `LineOffset` for clarity
