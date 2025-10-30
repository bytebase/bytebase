# BYT-7917: Complete Solution - Table Require PK False Positive Fix

**Design Version**: Complete Solution with Local State Tracking
**Date**: 2025-10-27
**Status**: Ready for Implementation

---

## 1. Executive Summary

### Problem
The `table.require-pk` advisor across all 6 database engines currently validates each statement independently, causing false positives when a primary key is added in a subsequent statement within the same migration script.

**Example:**
```sql
CREATE TABLE employees (id INT, name VARCHAR(100));  -- ❌ False positive warning
ALTER TABLE employees ADD PRIMARY KEY (id);           -- PK added here
```

### Solution Overview
Implement **two-phase validation with local state tracking**:
1. **Phase 1 (Collection)**: Track PK-affecting operations and maintain local PK state for newly created tables
2. **Phase 2 (Validation)**: Validate against `catalog.Final` state to determine actual violations

### Affected Engines
1. PostgreSQL (pgantlr) - New ANTLR-based implementation
2. MySQL - Existing two-phase implementation
3. TiDB - Existing two-phase implementation
4. MSSQL - Needs catalog support
5. Oracle - Needs catalog support
6. Snowflake - Needs catalog support

---

## 2. Technical Background

### Current Architecture

All engines follow a similar pattern but with different parsers:
- **PostgreSQL (pgantlr)**: ANTLR Listener pattern
- **MySQL**: ANTLR Listener pattern
- **TiDB**: AST Visitor pattern (TiDB parser)
- **MSSQL, Oracle, Snowflake**: ANTLR Listener pattern

### The Core Problem

When processing multi-statement scripts, advisors validate each statement as it's encountered:

```sql
-- Statement 1
CREATE TABLE t(id INT, name TEXT);
-- ❌ Advisor sees no PK, reports violation immediately

-- Statement 2
ALTER TABLE t ADD PRIMARY KEY (id);
-- ✅ Now table has PK, but too late - already reported
```

### Why This Is Complex

For newly created tables, `catalog.Origin` doesn't contain them (Origin represents database state BEFORE the script). We cannot rely solely on catalog queries.

**Example Challenge:**
```sql
Line 1: CREATE TABLE t(id INT PRIMARY KEY);
Line 2: ALTER TABLE t DROP CONSTRAINT t_pkey;
```

When processing Line 2, checking `catalog.Origin` for constraint information returns nothing because table `t` doesn't exist yet.

---

## 3. Detailed Design

### 3.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                     AST/Parse Tree                       │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │   Phase 1: Collection  │
         │   ─────────────────    │
         │  Walk all statements   │
         │  Track PK operations   │
         │  Build local PK state  │
         └───────────┬───────────┘
                     │
                     │  affectedTables map
                     │  localPKColumns map
                     │
                     ▼
         ┌───────────────────────┐
         │  Phase 2: Validation   │
         │  ──────────────────    │
         │  For each affected     │
         │  table, check          │
         │  catalog.Final state   │
         └───────────┬───────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │   Generate Advice      │
         │   with accurate line   │
         │   numbers              │
         └───────────────────────┘
```

### 3.2 Data Structures

```go
// PostgreSQL (pgantlr) Example
type tableRequirePKChecker struct {
    *parser.BasePostgreSQLParserListener

    adviceList     []*storepb.Advice
    level          storepb.Advice_Status
    title          string
    statementsText string
    catalog        *catalog.Finder

    // NEW: Track affected tables
    affectedTables map[string]*affectedTable  // key: "schema.table"

    // NEW: Local PK tracking for newly created tables
    localPKColumns map[string][]string  // key: "schema.table", value: PK columns

    // NEW: Cache Origin lookups
    originTableCache map[string]bool  // key: "schema.table", value: exists?
}

type affectedTable struct {
    schemaName string
    tableName  string
    lastLine   int      // Line of last PK-affecting operation
    lastStmt   string   // Statement text for error message
    isNewTable bool     // Created in this script?
}
```

### 3.3 Algorithm Flow

#### Phase 1: Collection

```
For each statement in script:

    If CREATE TABLE:
        Extract PK columns from:
            - Column-level: id INT PRIMARY KEY
            - Table-level: PRIMARY KEY (id)
            - Composite: PRIMARY KEY (id1, id2)

        Store in localPKColumns[table] = [columns]
        Record as affected (isNewTable=true)

    Else if ALTER TABLE DROP CONSTRAINT:
        Determine if constraint is PK:
            - If table is new: Check localPKColumns
            - If table exists: Query catalog.Origin.FindIndex()

        If constraint is PK:
            Remove from localPKColumns
            Record as affected with current line number

    Else if ALTER TABLE DROP COLUMN:
        Determine if column is in PK:
            - If table is new: Check localPKColumns
            - If table exists: Query catalog.Origin.FindPrimaryKey()

        If column in PK:
            Remove from localPKColumns (for composite PKs)
            Record as affected with current line number

    Else if ALTER TABLE ADD PRIMARY KEY:
        Extract PK columns
        Update localPKColumns[table]
        (Do NOT record as affected - this ADDS PK)
```

#### Phase 2: Validation

```
For each affected table:

    hasPK = catalog.Final.HasPrimaryKey(schemaName, tableName)

    If NOT hasPK:
        Generate advice:
            - Table name
            - Line number: affectedTable.lastLine
            - Statement text: affectedTable.lastStmt
            - Error: "Table requires PRIMARY KEY"
```

### 3.4 Helper Functions

#### 3.4.1 Table Key Generation
```go
func tableKey(schema, table string) string {
    return fmt.Sprintf("%s.%s", normalizeSchemaName(schema), table)
}
```

#### 3.4.2 Origin Existence Check (with caching)
```go
func (c *tableRequirePKChecker) existsInOrigin(schema, table string) bool {
    key := tableKey(schema, table)

    // Check cache
    if exists, cached := c.originTableCache[key]; cached {
        return exists
    }

    // Query catalog
    originTable := c.catalog.Origin.FindTable(&catalog.TableFind{
        SchemaName: normalizeSchemaName(schema),
        TableName:  table,
    })

    exists := originTable != nil
    c.originTableCache[key] = exists
    return exists
}
```

#### 3.4.3 Extract PK Columns from CREATE TABLE
```go
// PostgreSQL ANTLR example
func extractPKColumns(ctx *parser.CreatestmtContext) []string {
    var pkColumns []string

    if ctx.Opttableelementlist() == nil {
        return pkColumns
    }

    allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()

    for _, elem := range allElements {
        // Column-level: id INT PRIMARY KEY
        if elem.ColumnDef() != nil {
            columnDef := elem.ColumnDef()
            if hasColumnLevelPK(columnDef) {
                columnName := extractColumnName(columnDef)
                pkColumns = append(pkColumns, columnName)
            }
        }

        // Table-level: PRIMARY KEY (id, name)
        if elem.Tableconstraint() != nil {
            constraint := elem.Tableconstraint()
            if isPrimaryKeyConstraint(constraint) {
                cols := extractConstraintColumns(constraint)
                pkColumns = append(pkColumns, cols...)
            }
        }
    }

    return pkColumns
}
```

#### 3.4.4 Check if DROP CONSTRAINT Affects PK
```go
func (c *tableRequirePKChecker) isDropPKConstraint(
    schemaName, tableName, constraintName string,
) bool {
    key := tableKey(schemaName, tableName)

    // Case 1: New table - check local tracking
    if affected, exists := c.affectedTables[key]; exists && affected.isNewTable {
        // For new tables, we conservatively assume any DROP CONSTRAINT
        // might affect PK. Actual validation happens in Phase 2.
        return true
    }

    // Case 2: Existing table - query catalog.Origin
    _, index := c.catalog.Origin.FindIndex(&catalog.IndexFind{
        SchemaName: schemaName,
        TableName:  tableName,
        IndexName:  constraintName,
    })

    return index != nil && index.Primary()
}
```

#### 3.4.5 Check if DROP COLUMN Affects PK
```go
func (c *tableRequirePKChecker) isDropPKColumn(
    schemaName, tableName, columnName string,
) bool {
    key := tableKey(schemaName, tableName)

    // Case 1: New table - check local tracking
    if pkColumns, exists := c.localPKColumns[key]; exists {
        for _, col := range pkColumns {
            if col == columnName {
                return true
            }
        }
        return false
    }

    // Case 2: Existing table - query catalog.Origin
    pk := c.catalog.Origin.FindPrimaryKey(&catalog.PrimaryKeyFind{
        SchemaName: schemaName,
        TableName:  tableName,
    })

    if pk != nil {
        for _, col := range pk.ExpressionList() {
            if col == columnName {
                return true
            }
        }
    }

    return false
}
```

---

## 4. Implementation by Engine

### 4.1 PostgreSQL (pgantlr) - Complex Refactor

**Current State**: Per-statement validation, marked "WIP - needs debugging"

**File**: `backend/plugin/advisor/pgantlr/advisor_table_require_pk.go`

**Changes Required**:
1. Add data structures to `tableRequirePKChecker`
2. Refactor `EnterCreatestmt()` to track instead of validate
3. Refactor `EnterAltertablestmt()` to track PK-affecting operations
4. Add `validateFinalState()` method
5. Modify `Check()` to call validation after AST walk
6. Add helper functions for PK extraction and checking

**Complexity**: High (structural changes required)

**Test File**: `backend/plugin/advisor/pgantlr/test/table_require_pk.yaml`

### 4.2 MySQL - Simple Modification

**Current State**: Already has two-phase structure with local tracking

**File**: `backend/plugin/advisor/mysql/rule_table_require_pk.go`

**Changes Required**:
1. Modify `generateAdviceList()` method (lines 227-240)
2. Add catalog.Final validation alongside local state check

**Before:**
```go
func (r *TableRequirePKRule) generateAdviceList() {
    tableList := r.getTableList()
    for _, tableName := range tableList {
        if len(r.tables[tableName]) == 0 {  // Only checks local state
            r.AddAdvice(...)
        }
    }
}
```

**After:**
```go
func (r *TableRequirePKRule) generateAdviceList() {
    for tableName := range r.tables {
        // Check BOTH local state AND catalog.Final
        hasPKLocal := len(r.tables[tableName]) > 0
        hasPKFinal := r.catalog.Final.HasPrimaryKey(&catalog.PrimaryKeyFind{
            TableName: tableName,
        })

        // Only report if BOTH checks fail
        if !hasPKLocal && !hasPKFinal {
            r.AddAdvice(...)
        }
    }
}
```

**Complexity**: Low (single method modification)

**Test File**: `backend/plugin/advisor/mysql/test/table_require_pk.yaml`

### 4.3 TiDB - Same Pattern as MySQL

**Current State**: Already has two-phase structure with local tracking

**File**: `backend/plugin/advisor/tidb/advisor_table_require_pk.go`

**Changes Required**: Same as MySQL (modify validation logic)

**Complexity**: Low

**Test File**: `backend/plugin/advisor/tidb/test/table_require_pk.yaml`

### 4.4 MSSQL - Add Catalog Support

**Current State**: No catalog access, only local tracking

**File**: `backend/plugin/advisor/mssql/rule_table_require_pk.go`

**Changes Required**:
1. Add `catalog` field to `TableRequirePkRule` struct (line 68)
2. Pass `checkCtx.Catalog` in `NewTableRequirePkRule()` (line 42)
3. Modify `generateFinalAdvice()` to use catalog.Final

**Before:**
```go
func NewTableRequirePkRule(level storepb.Advice_Status, title string) *TableRequirePkRule {
    return &TableRequirePkRule{
        BaseRule: BaseRule{...},
        // No catalog
    }
}
```

**After:**
```go
func NewTableRequirePkRule(
    level storepb.Advice_Status,
    title string,
    catalog *catalog.Finder,  // NEW
) *TableRequirePkRule {
    return &TableRequirePkRule{
        BaseRule: BaseRule{...},
        catalog: catalog,  // NEW
    }
}

// In Check():
rule := NewTableRequirePkRule(level, string(checkCtx.Rule.Type), checkCtx.Catalog)
```

**Complexity**: Medium (requires plumbing catalog through)

**Test File**: `backend/plugin/advisor/mssql/test/table_require_pk.yaml`

### 4.5 Oracle - Same Pattern as MSSQL

**File**: `backend/plugin/advisor/oracle/rule_table_require_pk.go`

**Changes**: Add catalog support, modify final validation

**Complexity**: Medium

**Test File**: `backend/plugin/advisor/oracle/test/table_require_pk.yaml`

### 4.6 Snowflake - Same Pattern as MSSQL

**File**: `backend/plugin/advisor/snowflake/rule_table_require_pk.go`

**Changes**: Add catalog support, modify final validation

**Complexity**: Medium

**Test File**: `backend/plugin/advisor/snowflake/test/table_require_pk.yaml`

---

## 5. Test Cases

### Required Test Cases (All Engines)

```yaml
# Test 1: BYT-7917 - CREATE without PK, ADD PK (should PASS)
- statement: |-
    CREATE TABLE employees(id INT, name VARCHAR(100));
    ALTER TABLE employees ADD PRIMARY KEY (id);
  changeType: 1
  # Expected: No violation

# Test 2: CREATE with PK, DROP PK constraint (should FAIL)
- statement: |-
    CREATE TABLE book(id INT PRIMARY KEY, title TEXT);
    ALTER TABLE book DROP CONSTRAINT book_pkey;
  changeType: 1
  want:
    - status: 2
      code: 601
      title: table.require-pk
      content: 'Table "public"."book" requires PRIMARY KEY'
      startposition:
        line: 2  # Line of DROP CONSTRAINT

# Test 3: CREATE with PK, DROP column in PK (should FAIL)
- statement: |-
    CREATE TABLE author(id INT PRIMARY KEY, name TEXT);
    ALTER TABLE author DROP COLUMN id;
  changeType: 1
  want:
    - status: 2
      code: 601
      startposition:
        line: 2  # Line of DROP COLUMN

# Test 4: CREATE without PK, ADD PK, DROP PK (should FAIL)
- statement: |-
    CREATE TABLE product(id INT, name TEXT);
    ALTER TABLE product ADD PRIMARY KEY (id);
    ALTER TABLE product DROP CONSTRAINT product_pkey;
  changeType: 1
  want:
    - status: 2
      code: 601
      startposition:
        line: 3  # Line of DROP CONSTRAINT

# Test 5: CREATE without PK, ADD PK, DROP PK column (should FAIL)
- statement: |-
    CREATE TABLE customer(id INT, email TEXT);
    ALTER TABLE customer ADD PRIMARY KEY (id);
    ALTER TABLE customer DROP COLUMN id;
  changeType: 1
  want:
    - status: 2
      code: 601
      startposition:
        line: 3  # Line of DROP COLUMN

# Test 6: Composite PK - CREATE, DROP one column (should FAIL)
- statement: |-
    CREATE TABLE enrollment(
      student_id INT,
      course_id INT,
      PRIMARY KEY (student_id, course_id)
    );
    ALTER TABLE enrollment DROP COLUMN student_id;
  changeType: 1
  want:
    - status: 2
      code: 601
      startposition:
        line: 6  # Line of DROP COLUMN

# Test 7: Existing table (in catalog.Origin), DROP non-PK column (should PASS)
- statement: ALTER TABLE "tech_book" DROP COLUMN non_pk_column;
  changeType: 1
  # Expected: No violation (tech_book has PK, dropping non-PK column)

# Test 8: Existing table (in catalog.Origin), DROP PK constraint (should FAIL)
- statement: ALTER TABLE "tech_book" DROP CONSTRAINT "old_pk";
  changeType: 1
  want:
    - status: 2
      code: 601
```

### Test Coverage Matrix

| Scenario | CREATE in Script? | Has PK Initially? | Operation | Expected Result |
|----------|------------------|-------------------|-----------|-----------------|
| BYT-7917 | Yes | No | ADD PK | PASS ✅ |
| Edge 1 | Yes | Yes | DROP PK | FAIL ❌ |
| Edge 2 | Yes | Yes | DROP PK col | FAIL ❌ |
| Edge 3 | Yes | No | ADD then DROP PK | FAIL ❌ |
| Edge 4 | Yes | No | ADD PK, DROP col | FAIL ❌ |
| Edge 5 | Yes | Yes (composite) | DROP 1 col | FAIL ❌ |
| Edge 6 | No (in Origin) | Yes | DROP non-PK col | PASS ✅ |
| Edge 7 | No (in Origin) | Yes | DROP PK | FAIL ❌ |

---

## 6. Implementation Checklist

### Phase 1: Setup
- [ ] Rebase from `origin/main`
- [ ] Resolve any conflicts
- [ ] Update `backend/plugin/advisor/catalog/state.go` with `HasPrimaryKey()` helper (if not present)

### Phase 2: PostgreSQL (pgantlr)
- [ ] Read and understand current "WIP" implementation
- [ ] Add 8 test cases to `test/table_require_pk.yaml`
- [ ] Add data structures to `tableRequirePKChecker`
- [ ] Implement helper functions
- [ ] Refactor `EnterCreatestmt()`
- [ ] Refactor `EnterAltertablestmt()`
- [ ] Add `validateFinalState()` method
- [ ] Update `Check()` function
- [ ] Run tests: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/pgantlr -run TestPostgreSQLANTLRRules`
- [ ] Debug and fix failures
- [ ] Uncomment `advisor.SchemaRuleTableRequirePK` in `pgantlr_test.go`

### Phase 3: MySQL
- [ ] Add 8 test cases
- [ ] Modify `generateAdviceList()` method
- [ ] Run tests: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/mysql`
- [ ] Fix any failures

### Phase 4: TiDB
- [ ] Add 8 test cases
- [ ] Modify `generateAdviceList()` method
- [ ] Run tests: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/tidb`
- [ ] Fix any failures

### Phase 5: MSSQL
- [ ] Add 8 test cases (adapt syntax for T-SQL)
- [ ] Add catalog support
- [ ] Modify `generateFinalAdvice()` method
- [ ] Run tests
- [ ] Fix failures

### Phase 6: Oracle
- [ ] Add 8 test cases (adapt syntax for PL/SQL)
- [ ] Add catalog support
- [ ] Modify final validation
- [ ] Run tests
- [ ] Fix failures

### Phase 7: Snowflake
- [ ] Add 8 test cases (adapt syntax for Snowflake SQL)
- [ ] Add catalog support
- [ ] Modify final validation
- [ ] Run tests
- [ ] Fix failures

### Phase 8: Final Validation
- [ ] Run all advisor tests: `go test -v -count=1 github.com/bytebase/bytebase/backend/plugin/advisor/...`
- [ ] Run linting: `golangci-lint run --allow-parallel-runners`
- [ ] Fix linting issues (run multiple times until clean)
- [ ] Build project: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
- [ ] Manual smoke test (if possible)

---

## 7. Complexity Analysis

### Code Changes by Engine

| Engine | Current Lines | Estimated New Lines | Net Change | Complexity |
|--------|--------------|---------------------|------------|------------|
| PostgreSQL (pgantlr) | 204 | ~300 | +96 | High |
| MySQL | 279 | ~290 | +11 | Low |
| TiDB | 240 | ~250 | +10 | Low |
| MSSQL | 198 | ~230 | +32 | Medium |
| Oracle | 175 | ~207 | +32 | Medium |
| Snowflake | 217 | ~249 | +32 | Medium |
| **Total** | **1,313** | **~1,526** | **+213** | - |

### Performance Characteristics

- **Time Complexity**: O(n) where n = number of statements
  - Each statement processed once
  - Catalog lookups cached
- **Space Complexity**: O(t) where t = number of affected tables
  - Typically small (< 10 tables per script)
- **Catalog Query Impact**: Minimal
  - Origin lookups cached per table
  - Final state queried once per affected table

---

## 8. Risk Assessment

### Low Risk
- ✅ Self-contained changes (no external API modifications)
- ✅ Comprehensive test coverage
- ✅ Reuses existing catalog infrastructure
- ✅ No breaking changes to rule configuration

### Medium Risk
- ⚠️ PostgreSQL pgantlr has existing "WIP" issues to debug
- ⚠️ Need to handle edge cases (composite PKs, constraint names)
- ⚠️ Catalog.Final must accurately reflect post-script state

### Mitigation Strategies
1. **Incremental Rollout**: Fix engines one by one, test thoroughly
2. **Comprehensive Testing**: 8+ test cases per engine covering edge cases
3. **Code Review**: Focus on PK extraction and tracking logic
4. **Fallback**: Keep old implementation as reference during development

---

## 9. Timeline Estimate

| Task | Estimated Time | Priority |
|------|---------------|----------|
| Setup & Rebase | 30 min | High |
| PostgreSQL (pgantlr) | 4-5 hours | Critical |
| MySQL | 1.5 hours | High |
| TiDB | 1.5 hours | High |
| MSSQL | 2 hours | Medium |
| Oracle | 2 hours | Medium |
| Snowflake | 2 hours | Medium |
| Testing & Polish | 1.5 hours | High |
| **Total** | **15-17 hours** | - |

---

## 10. Success Criteria

### Functional Requirements
- ✅ BYT-7917 scenario (CREATE without PK, ADD PK) passes for all engines
- ✅ All edge cases handled correctly (8 test scenarios)
- ✅ Existing test cases continue to pass
- ✅ Line numbers point to exact PK-affecting statement

### Non-Functional Requirements
- ✅ No performance degradation
- ✅ Code passes `golangci-lint` without errors
- ✅ No breaking changes to rule configuration
- ✅ Consistent behavior across all 6 engines

### Deliverables
1. Updated implementations for all 6 engines
2. 8+ test cases per engine (48+ total new tests)
3. All tests passing
4. Linting clean
5. Project builds successfully
6. PostgreSQL pgantlr version enabled (migration unblocked)

---

## 11. Future Enhancements

### Potential Improvements
1. **More Granular Line Numbers**: Point to exact constraint definition, not just statement
2. **Better Error Messages**: Include column names in composite PK violations
3. **Transaction Boundaries**: Consider rollback scenarios
4. **Performance Optimization**: Batch catalog queries if needed

### Not In Scope
- ❌ Validating PK column types (separate rule)
- ❌ Suggesting which column should be PK
- ❌ Handling temporary tables differently
- ❌ Cross-database PK consistency checks

---

## 12. References

### Related Issues
- BYT-7917: Table require PK false positive when PK added in subsequent statement

### Related Files
- `backend/plugin/advisor/catalog/state.go` - Catalog infrastructure
- `backend/plugin/advisor/pgantlr/advisor_table_require_pk.go` - PostgreSQL impl
- `backend/plugin/advisor/mysql/rule_table_require_pk.go` - MySQL impl
- `backend/plugin/advisor/tidb/advisor_table_require_pk.go` - TiDB impl
- `backend/plugin/advisor/mssql/rule_table_require_pk.go` - MSSQL impl
- `backend/plugin/advisor/oracle/rule_table_require_pk.go` - Oracle impl
- `backend/plugin/advisor/snowflake/rule_table_require_pk.go` - Snowflake impl

### Documentation
- Google Style Guide: https://google.github.io/styleguide/go/
- Bytebase Development SOP: `CLAUDE.md`

---

**Document End**
