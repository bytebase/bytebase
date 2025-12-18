# Eliminate ParseResult Type

## Summary

Remove the redundant `ParseResult` type from the parser infrastructure. All ANTLR-based parsers will return `*ANTLRAST` directly instead of `*ParseResult`.

## Background

Currently we have two nearly identical types:

```go
// ParseResult - used internally by parse functions
type ParseResult struct {
    Tree     antlr.Tree
    Tokens   *antlr.CommonTokenStream
    BaseLine int  // 0-based
}

// ANTLRAST - implements AST interface
type ANTLRAST struct {
    StartPosition *storepb.Position  // 1-based
    Tree          antlr.Tree
    Tokens        *antlr.CommonTokenStream
}
```

The only difference is `BaseLine` (0-based) vs `StartPosition.Line` (1-based). Every parser has a `toAST()` function that converts between them.

## Changes

### 1. Delete ParseResult

Remove from `backend/plugin/parser/base/base.go`:
```go
type ParseResult struct {
    Tree     antlr.Tree
    Tokens   *antlr.CommonTokenStream
    BaseLine int
}
```

### 2. Update Parse Functions

Change return types in 14 parser packages:

| Package | Functions |
|---------|-----------|
| pg | `ParsePostgreSQL`, `ParsePostgreSQLPLBlock`, `parseSinglePostgreSQL` |
| mysql | `ParseMySQL`, `parseInputStream` |
| tidb | `ANTLRParseTiDB`, `parseInputStream` |
| plsql | `ParsePLSQL` |
| tsql | `ParseTSQL`, `parseSingleTSQL` |
| snowflake | `ParseSnowSQL`, `parseSingleSnowSQL` |
| redshift | `ParseRedshift`, `parseSingleRedshift` |
| bigquery | `ParseBigQuerySQL`, `parseSingleBigQuerySQL` |
| spanner | `ParseSpannerGoogleSQL`, `parseSingleSpannerGoogleSQL` |
| doris | `ParseDorisSQL`, `parseSingleDorisSQL` |
| cassandra | `ParseCassandraSQL`, `parseSingleCassandraSQL` |
| trino | `ParseTrino`, `parseSingleTrino` |
| partiql | `ParsePartiQL`, `parseSinglePartiQL` |
| cosmosdb | `ParseCosmosDBQuery`, `parseSingleCosmosDBQuery` |

Return type changes from `[]*base.ParseResult` to `[]*base.ANTLRAST`.

### 3. Delete toAST() Functions

Remove from 9 packages: pg, mysql, plsql, tsql, snowflake, redshift, doris, cassandra, partiql.

### 4. Update Consumer Functions

Change parameter types:

- `pg/restore.go`: `doGenerate(..., tree *base.ANTLRAST, ...)`
- `mysql/restore.go`: `doGenerate(..., parseResult *base.ANTLRAST, ...)`
- `tsql/restore.go`: `doGenerate(..., tree *base.ANTLRAST, ...)`
- `mysql/backup.go`: `ExtractTables(..., parseResult *base.ANTLRAST, ...)`
- `tidb/backup.go`: `extractTables(..., parseResult *base.ANTLRAST, ...)`
- `doris/query_span_extractor.go`: `getAccessTables(..., parseResult *base.ANTLRAST, ...)`

### 5. Access Pattern Change

```go
// Before
baseLine := parseResult.BaseLine

// After
baseLine := base.GetLineOffset(ast.StartPosition)
```

## Result

- One less type to maintain
- No conversion layer between parsing and AST usage
- Consistent 1-based positioning throughout (via StartPosition)
- `GetLineOffset()` helper remains for consumers needing 0-based offset
