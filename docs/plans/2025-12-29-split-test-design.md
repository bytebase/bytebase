# Split Test Design

## Overview

This document describes the standardized test design for SQL statement splitting across all parser engines.

## Key Semantics

| Field | Description |
|-------|-------------|
| `Start` | 1-based inclusive position (Line, Column) |
| `End` | 1-based exclusive position (Line, Column) |
| `Range` | Byte offset range for the statement |
| `Column` | Character offset (not byte offset) |
| `BaseLine` | Zero-based line offset for the statement |

## Test Infrastructure

### File Structure

```
backend/plugin/parser/base/
├── split_test_runner.go      # Shared test runner

backend/plugin/parser/<engine>/
├── split.go
├── split_test.go             # Uses shared runner
└── test-data/
    └── test_split.yaml       # Test cases
```

### YAML Schema

```yaml
- description: "Brief description"
  input: |
    SELECT 1;
    SELECT 2;
  error: ""  # Optional: expected error message
  result:
    - text: "SELECT 1;"
      baseline: 0
      start: {line: 1, column: 1}
      end: {line: 1, column: 10}
      range: {start: 0, end: 9}
      empty: false
```

### Shared Test Runner

```go
// SplitTestCase represents a single test case
type SplitTestCase struct {
    Description string            `yaml:"description"`
    Input       string            `yaml:"input"`
    Error       string            `yaml:"error,omitempty"`
    Result      []StatementResult `yaml:"result,omitempty"`
}

type StatementResult struct {
    Text     string         `yaml:"text"`
    BaseLine int            `yaml:"baseline"`
    Start    PositionResult `yaml:"start"`
    End      PositionResult `yaml:"end"`
    Range    RangeResult    `yaml:"range"`
    Empty    bool           `yaml:"empty"`
}

type PositionResult struct {
    Line   int32 `yaml:"line"`
    Column int32 `yaml:"column"`
}

type RangeResult struct {
    Start int32 `yaml:"start"`
    End   int32 `yaml:"end"`
}

type SplitTestOptions struct {
    SplitFunc       func(string) ([]Statement, error)
    LexerSplitFunc  func(string) ([]Statement, error) // Optional
    ParserSplitFunc func(string) ([]Statement, error) // Optional
}

// RunSplitTests loads YAML and runs tests with mandatory consistency check
func RunSplitTests(t *testing.T, testDataPath string, opts SplitTestOptions) {
    testCases := loadTestCases(t, testDataPath)

    for _, tc := range testCases {
        t.Run(tc.Description, func(t *testing.T) {
            var lexerResult, parserResult []Statement
            var lexerErr, parserErr error

            if opts.LexerSplitFunc != nil {
                lexerResult, lexerErr = opts.LexerSplitFunc(tc.Input)
            }
            if opts.ParserSplitFunc != nil {
                parserResult, parserErr = opts.ParserSplitFunc(tc.Input)
            }

            // MANDATORY: If both succeed, results must be identical
            if lexerErr == nil && parserErr == nil {
                require.Equal(t, lexerResult, parserResult,
                    "lexer and parser produced different results")
            }

            // Verify combined function
            result, err := opts.SplitFunc(tc.Input)
            assertExpectedResult(t, tc, result, err)
        })
    }
}
```

### Engine Test File

```go
// pg/split_test.go
func TestPGSplitSQL(t *testing.T) {
    base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
        SplitFunc: SplitSQL,
    })
}

// redshift/split_test.go (with dual implementation)
func TestRedshiftSplitSQL(t *testing.T) {
    base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
        SplitFunc:       SplitSQL,
        LexerSplitFunc:  splitByLexer,
        ParserSplitFunc: splitByParser,
    })
}
```

## Test Categories

### Core Categories (All Engines)

| Category | Description | Example |
|----------|-------------|---------|
| Single statement | Basic single SQL | `SELECT * FROM t;` |
| Multiple same line | Two statements on one line | `SELECT 1; SELECT 2;` |
| Multi-line statement | Statement spanning lines | `SELECT *\nFROM t;` |
| Leading whitespace | Statements with indentation | `    CREATE TABLE t(a int);` |
| Single-line comment | `--` style comments | `-- comment\nSELECT 1;` |
| Multi-line comment | `/* */` style comments | `/* comment */\nSELECT 1;` |
| Empty statement | Only comments, no SQL | `-- comment\n/* block */;` |
| Unicode characters | Non-ASCII in identifiers | `SELECT * FROM 表名;` |
| Windows line endings | CRLF handling | `SELECT 1;\r\nSELECT 2;` |
| Unterminated string | Error or graceful handling | `SELECT 'unclosed` |
| No trailing semicolon | Last statement without `;` | `SELECT 1; SELECT 2` |

### Engine-Specific Categories

| Engine | Category | Description |
|--------|----------|-------------|
| MySQL | Delimiter command | `DELIMITER ;;` for procedures |
| MySQL | Stored procedure | `CREATE PROCEDURE...BEGIN...END` |
| MySQL | Hash comment | `# comment` style |
| PostgreSQL | Dollar-quoted string | `$$body$$` or `$tag$body$tag$` |
| PostgreSQL | Stored procedure | `CREATE PROCEDURE...AS $$...$$` |
| TSQL | GO batch separator | `GO` between batches |
| TSQL | Stored procedure | `CREATE PROCEDURE...BEGIN...END` |
| Oracle/PLSQL | Anonymous block | `BEGIN...END;` |
| Oracle/PLSQL | Package | `CREATE PACKAGE...END;` |
| Spanner | BEGIN/END blocks | Scripting statements |
| Redshift | Stored procedure | Similar to PostgreSQL |

## Dual Implementation Consistency

For engines with both lexer-based and parser-based implementations:

| Lexer | Parser | Behavior |
|-------|--------|----------|
| Success | Success | **MUST be identical** - test fails if different |
| Fail | Success | OK - fallback to parser |
| Success | Fail | OK - lexer sufficient |
| Fail | Fail | Expected error case |

The test runner automatically enforces this consistency for every test case.

## Implementation Plan

1. Create `backend/plugin/parser/base/split_test_runner.go`
2. Create YAML test files for each engine
3. Update each engine's `split_test.go` to use shared runner
4. Run tests to verify existing behavior
5. Add missing test categories per engine
