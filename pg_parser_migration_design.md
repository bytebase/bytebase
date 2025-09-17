# PostgreSQL Parser Migration Design

## 1. Background

Bytebase currently uses two PostgreSQL parsers:
- **pg_query_go**: CGO-based parser using libpg_query (PostgreSQL's official parser)
- **ANTLR v4**: Pure Go parser already used for other SQL dialects

Goal: Migrate all PostgreSQL parsing to ANTLR v4 for consistency and to eliminate CGO dependencies.

## 2. Current State Analysis

### AST Usage Patterns

**AST Caching is ONLY used for:**
1. **SQL Review/Advisors** - 52 advisor rules that analyze SQL statements
2. **Statement Reports** - Getting statement types and affected rows count
3. **Syntax Checking** - Initial parse validation before running advisors

**AST is NOT cached for:**
- Query validation (parses fresh each time)
- Schema diffing (parses fresh each time)
- Restore SQL generation (parses fresh each time)
- Any other feature implementations

### AST Caching Architecture

```
component/sheet/Manager
├── astCache: LRU cache with 3-minute TTL
├── GetASTsForChecks(): Used ONLY by SQL review & statement reports
└── syntaxCheck(): Actual parsing logic
```

**Cache details:**
- Cache key: Statement hash + engine type
- Cache capacity: 8 items with 3-minute TTL
- Cache value: `any` type (can be []ast.Node or ANTLR tree)
- Thread-safe with mutex locking

### Components Using pg_query_go

1. **SQL Review System** (`plugin/advisor/pg/*.go`)
   - 52 advisor rules using cached AST via `GetASTsForChecks()`
   - Access AST via: `stmts, ok := checkCtx.AST.([]ast.Node)`
   - Implement visitor pattern on legacy AST nodes

2. **Statement Reports** (`runner/plancheck/statement_report_executor.go`)
   - Uses cached AST via `GetASTsForChecks()`
   - Gets statement types and affected rows

3. **Query Validation** (`plugin/parser/pg/query.go`)
   - Direct parsing with `pgrawparser.Parse()` (NO caching)
   - Validates if query is read-only

4. **Schema Differ** (`plugin/parser/pg/differ.go`)
   - Direct parsing with `pgrawparser.Parse()` (NO caching)
   - Will be replaced separately (not part of this migration)

5. **Restore SQL Generation** (`plugin/parser/pg/restore.go`)
   - Mixed usage: ANTLR for main logic, pgrawparser for extracting SET ROLE
   - Direct parsing (NO caching)

6. **Query Span Extractor** (`plugin/parser/pg/query_span_extractor.go`)
   - Mixed usage with pgquery.ParseToJSON()
   - Direct parsing (NO caching)

## 3. Migration Strategy

### Two Different Approaches

**For SQL Review (Cached AST):**
1. **Parse with BOTH parsers and cache BOTH ASTs simultaneously**
2. **Migrated advisors use ANTLR AST, non-migrated use legacy AST**
3. **Gradually migrate advisors one by one**

**For Other Components (Non-cached):**
1. **Directly replace implementation with ANTLR**
2. **No dual AST needed - they parse fresh each time**
3. **Simple replacement, no gradual migration needed**

### Phase 1: SQL Review Dual AST Caching (Week 1)

**Objective:** Set up dual AST caching ONLY for SQL Review

1. Update cache Result structure for SQL Review:
   ```go
   // component/sheet/sheet.go
   type Result struct {
       sync.Mutex
       legacyAST  []ast.Node                    // pg_query_go AST
       antlrAST   *PostgreSQLParser.RootContext  // ANTLR AST
       advices    []*storepb.Advice
   }
   ```

2. Update `postgresSyntaxCheck` to parse with BOTH parsers (ONLY for caching):
   ```go
   func postgresSyntaxCheck(statement string) (any, []*storepb.Advice) {
       // Parse with pg_query_go (legacy)
       legacyNodes, legacyErr := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
       if legacyErr != nil {
           return nil, convertLegacyError(legacyErr)
       }
       
       // Parse with ANTLR
       antlrTree, antlrErr := antlrparser.Parse(statement)
       if antlrErr != nil {
           // Log warning but don't fail - use legacy during migration
           slog.Warn("ANTLR parse failed", "error", antlrErr)
       }
       
       // Return both ASTs in a wrapper (for SQL Review only)
       return &DualAST{
           Legacy: legacyNodes,
           ANTLR:  antlrTree,
       }, nil
   }
   ```

3. SQL Review advisors access the appropriate AST:
   ```go
   // Each advisor checks which AST to use
   func (a *insertMustSpecifyColumn) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
       if dualAST, ok := checkCtx.AST.(*DualAST); ok {
           if a.useANTLR && dualAST.ANTLR != nil {
               return a.checkANTLR(dualAST.ANTLR)
           }
           return a.checkLegacy(dualAST.Legacy)
       }
       // Backward compatibility
       return a.checkLegacy(checkCtx.AST.([]ast.Node))
   }
   ```

### Phase 2: Direct Component Replacement (Week 2)

**Objective:** Replace non-cached components directly with ANTLR

1. **Query Validation** - Direct replacement:
   ```go
   // plugin/parser/pg/query.go
   func validateQuery(statement string) (bool, bool, error) {
       // REMOVE: stmtList, err := pgrawparser.Parse(...)
       // ADD: Direct ANTLR parsing
       tree, err := antlrparser.Parse(statement)
       if err != nil {
           return false, false, convertToSyntaxError(statement, err)
       }
       
       // Reimplement validation logic using ANTLR visitor
       validator := &QueryValidator{}
       tree.Accept(validator)
       return validator.isValid, validator.hasExecute, nil
   }
   ```

2. **Query Span Extractor** - Direct replacement:
   ```go
   // plugin/parser/pg/query_span_extractor.go
   func ExtractQuerySpans(statement string) ([]Span, error) {
       // REMOVE: pgquery.ParseToJSON(...)
       // ADD: Use ANTLR directly
       tree, err := antlrparser.Parse(statement)
       if err != nil {
           return nil, err
       }
       
       extractor := &SpanExtractor{}
       tree.Accept(extractor)
       return extractor.spans, nil
   }
   ```

3. **Restore SQL** - Partial replacement:
   ```go
   // Only replace the SET ROLE extraction part
   func getPrependStatements(statement string) (string, error) {
       // Replace pgrawparser.Parse with ANTLR
       tree, err := antlrparser.Parse(statement)
       if err != nil {
           return "", err
       }
       
       // Find SET ROLE using ANTLR visitor
       roleExtractor := &SetRoleExtractor{}
       tree.Accept(roleExtractor)
       return roleExtractor.roleStatement, nil
   }
   ```

### Phase 3: SQL Review Advisor Migration (Week 3-4)

**Objective:** Migrate 52 SQL Review advisors to use cached ANTLR AST

1. **Advisor migration approach** (using dual AST cache):
   ```go
   // plugin/advisor/pg/advisor_insert_must_specify_column.go
   type insertMustSpecifyColumn struct {
       useANTLR bool  // Migration flag for gradual rollout
   }
   
   func (a *insertMustSpecifyColumn) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
       // SQL Review uses DualAST from cache
       if dualAST, ok := checkCtx.AST.(*DualAST); ok {
           if a.useANTLR && dualAST.ANTLR != nil {
               return a.checkANTLR(dualAST.ANTLR)
           }
           return a.checkLegacy(dualAST.Legacy)
       }
       
       // Fallback for tests or old code paths
       if nodes, ok := checkCtx.AST.([]ast.Node); ok {
           return a.checkLegacy(nodes)
       }
       
       return nil, errors.New("no valid AST available")
   }
   ```

2. **Migration process per advisor**:
   - Implement ANTLR visitor version
   - Test against legacy implementation
   - Enable `useANTLR` flag when ready
   - Monitor for discrepancies
   - Keep legacy code until all advisors migrated

3. **Statement Report migration**:
   ```go
   // Also needs to handle DualAST
   func GetSQLSummaryReport(...) {
       asts, _ := sheetManager.GetASTsForChecks(engine, statement)
       if dualAST, ok := asts.(*DualAST); ok {
           if useANTLR {
               // Use ANTLR AST
               sqlTypes, err = pg.GetStatementTypesANTLR(dualAST.ANTLR)
           } else {
               // Use legacy AST
               sqlTypes, err = pg.GetStatementTypes(dualAST.Legacy)
           }
       }
   }
   ```

### Phase 4: Testing and Validation (Throughout)

1. **Parallel Testing Strategy**:
   - Run both parsers on same input
   - Compare advisor outputs
   - Measure performance differences
   - Validate error handling

2. **Test Coverage**:
   - Unit tests for each reimplemented component
   - Integration tests with real SQL statements
   - Performance benchmarks
   - Edge case validation

### Phase 5: Switchover and Cleanup (End of Week 4)

1. **Gradual Switchover**:
   - Feature flag to toggle between implementations
   - Monitor in staging environment
   - Gradual rollout to production

2. **Remove pg_query_go dependencies**:
   - Remove `github.com/pganalyze/pg_query_go/v5` from go.mod
   - Remove `plugin/parser/pg/legacy` package entirely
   - Remove pgrawparser wrapper code
   - Remove deparse functionality (only used for logging)

## 4. Implementation Details

### Advisor Reimplementation Pattern

Each advisor will follow this pattern:

```go
// advisor_example.go
func (a *ExampleAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
    switch ast := checkCtx.AST.(type) {
    case []ast.Node:
        // Legacy implementation
        return a.checkLegacy(ast)
    case *PostgreSQLParser.RootContext:
        // ANTLR implementation
        return a.checkANTLR(ast)
    default:
        return nil, errors.New("unsupported AST type")
    }
}

func (a *ExampleAdvisor) checkANTLR(tree *PostgreSQLParser.RootContext) ([]*storepb.Advice, error) {
    visitor := &ExampleANTLRVisitor{
        adviceList: []*storepb.Advice{},
    }
    tree.Accept(visitor)
    return visitor.adviceList, nil
}
```

### Dual AST Caching Benefits

This approach provides several advantages:

1. **Gradual Migration**: Each advisor can be migrated independently
2. **Easy Rollback**: Just flip the `useANTLR` flag back to false
3. **A/B Testing**: Can compare outputs between parsers
4. **No Big Bang**: No need to migrate all 52 advisors at once
5. **Performance Monitoring**: Can measure performance difference per advisor

### Cache Performance Considerations

- **Memory**: Will temporarily use ~2x memory (both ASTs cached)
- **Parse Time**: Initial parse takes longer (parsing twice)
- **Cache Hit**: After first parse, performance identical to current
- **Migration Complete**: Remove dual parsing, return to normal memory/CPU usage

## 5. Migration Summary

### Components Using Dual AST Cache (Gradual Migration)
**Only SQL Review components use the dual AST caching strategy:**
1. **52 SQL Review Advisors** - Migrate one by one with flags
2. **Statement Report** - Update to handle DualAST

### Components Using Direct Replacement (Simple Migration)
**These don't use caching, just replace implementation:**
1. **Query Validation** (`query.go`) - Direct ANTLR replacement
2. **Query Span Extractor** (`query_span_extractor.go`) - Direct ANTLR replacement  
3. **Restore SQL** (`restore.go`) - Replace SET ROLE extraction only

### Out of Scope
1. **Schema Differ** - Will be replaced separately
2. **Deparse functionality** - Remove (only for logging)

## 6. Risk Mitigation

1. **Dual Implementation Risk**:
   - Risk: Maintaining two implementations temporarily
   - Mitigation: Clear separation, shared test suites

2. **Feature Parity Risk**:
   - Risk: ANTLR implementation might miss edge cases
   - Mitigation: Comprehensive testing, gradual rollout

3. **Performance Risk**:
   - Risk: ANTLR might be slower for complex queries
   - Mitigation: Profiling, optimization, caching

## 7. Success Criteria

- All 52 PostgreSQL advisors reimplemented with ANTLR
- Feature parity with pg_query_go implementation
- Performance within acceptable range (≤120% parsing time)
- Zero CGO dependencies for PostgreSQL
- Clean codebase without legacy parser code

## 8. Implementation Steps

1. **Week 1: Infrastructure**
   - Implement dual AST caching in `sheet.Manager`
   - Create DualAST wrapper structure
   - Set up ANTLR PostgreSQL parser
   - Update `postgresSyntaxCheck` to parse with both parsers

2. **Week 2: First Advisors**
   - Pick 3-5 simple advisors as pilot
   - Implement ANTLR versions
   - Add migration flags
   - Test dual-path execution

3. **Week 3-4: Mass Migration**
   - Migrate remaining advisors in batches
   - Run A/B tests comparing outputs
   - Monitor performance metrics
   - Fix any discrepancies

4. **Final Cleanup**
   - Remove all legacy implementations
   - Remove pg_query_go dependency
   - Simplify cache back to single AST
   - Performance optimization