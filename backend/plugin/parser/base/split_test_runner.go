package base

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// record is a flag to update expected results in YAML files.
// Usage: go test -run TestXxxSplitSQL -args -record
var record = flag.Bool("record", false, "update expected results in YAML test files")

// SplitTestCase represents a single test case in YAML.
type SplitTestCase struct {
	Description string            `yaml:"description"`
	Input       string            `yaml:"input"`
	Error       string            `yaml:"error,omitempty"`
	Result      []StatementResult `yaml:"result,omitempty"`
}

// StatementResult represents expected output for one statement.
type StatementResult struct {
	Text     string         `yaml:"text"`
	BaseLine int            `yaml:"baseline"`
	Start    PositionResult `yaml:"start"`
	End      PositionResult `yaml:"end"`
	Range    RangeResult    `yaml:"range"`
	Empty    bool           `yaml:"empty"`
}

// PositionResult represents a position in the YAML test data.
type PositionResult struct {
	Line   int32 `yaml:"line"`
	Column int32 `yaml:"column"`
}

// RangeResult represents a byte range in the YAML test data.
type RangeResult struct {
	Start int32 `yaml:"start"`
	End   int32 `yaml:"end"`
}

// SplitTestOptions configures how split tests are run.
type SplitTestOptions struct {
	// SplitFunc is the main split function to test (required).
	SplitFunc func(string) ([]Statement, error)
	// LexerSplitFunc is the lexer-based implementation (optional).
	LexerSplitFunc func(string) ([]Statement, error)
	// ParserSplitFunc is the parser-based implementation (optional).
	ParserSplitFunc func(string) ([]Statement, error)
}

// loadTestCases loads test cases from a YAML file.
// The path is relative to the caller's directory.
func loadTestCases(t *testing.T, relativePath string, callerDepth int) []SplitTestCase {
	t.Helper()

	// Get the caller's file path to resolve relative paths
	_, callerFile, _, ok := runtime.Caller(callerDepth)
	require.True(t, ok, "failed to get caller info")

	callerDir := filepath.Dir(callerFile)
	fullPath := filepath.Join(callerDir, relativePath)

	data, err := os.ReadFile(fullPath)
	require.NoError(t, err, "failed to read test file: %s", fullPath)

	var testCases []SplitTestCase
	err = yaml.Unmarshal(data, &testCases)
	require.NoError(t, err, "failed to parse YAML test file: %s", fullPath)

	return testCases
}

// assertStatementsEqual compares actual statements against expected results.
func assertStatementsEqual(t *testing.T, expected []StatementResult, actual []Statement, msgAndArgs ...any) {
	t.Helper()

	args := append([]any{"statement count mismatch"}, msgAndArgs...)
	require.Equal(t, len(expected), len(actual), args...)

	for i, exp := range expected {
		act := actual[i]
		msg := append([]any{"statement %d mismatch", i}, msgAndArgs...)

		require.Equal(t, exp.Text, act.Text, msg...)
		require.Equal(t, exp.BaseLine, act.BaseLine(), msg...)
		require.Equal(t, exp.Empty, act.Empty, msg...)

		require.NotNil(t, act.Start, msg...)
		require.Equal(t, exp.Start.Line, act.Start.Line, msg...)
		require.Equal(t, exp.Start.Column, act.Start.Column, msg...)

		require.NotNil(t, act.End, msg...)
		require.Equal(t, exp.End.Line, act.End.Line, msg...)
		require.Equal(t, exp.End.Column, act.End.Column, msg...)

		require.NotNil(t, act.Range, msg...)
		require.Equal(t, exp.Range.Start, act.Range.Start, msg...)
		require.Equal(t, exp.Range.End, act.Range.End, msg...)
	}
}

// RunSplitTests loads YAML test cases and runs them against a SplitSQL function.
// For engines with dual implementations (lexer and parser), it enforces that
// both produce identical results when both succeed.
//
// When run with -record flag, it updates the YAML file with actual results:
//
//	go test -run TestXxxSplitSQL -args -record
func RunSplitTests(t *testing.T, testDataPath string, opts SplitTestOptions) {
	t.Helper()

	require.NotNil(t, opts.SplitFunc, "SplitFunc is required")

	// Get the full path to the test file
	_, callerFile, _, ok := runtime.Caller(1)
	require.True(t, ok, "failed to get caller info")
	callerDir := filepath.Dir(callerFile)
	fullPath := filepath.Join(callerDir, testDataPath)

	// Caller depth 2: skip loadTestCases (1) and RunSplitTests (2) to get the actual test function
	testCases := loadTestCases(t, testDataPath, 2)
	require.NotEmpty(t, testCases, "no test cases found in %s", testDataPath)

	// Record mode: update expected results in YAML file
	if *record {
		recordTestResults(t, fullPath, testCases, opts)
		return
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			runSingleTest(t, tc, opts)
		})
	}
}

// recordTestResults runs all test cases and updates the YAML file with actual results.
func recordTestResults(t *testing.T, fullPath string, testCases []SplitTestCase, opts SplitTestOptions) {
	t.Helper()

	updated := make([]SplitTestCase, len(testCases))
	for i, tc := range testCases {
		updated[i] = SplitTestCase{
			Description: tc.Description,
			Input:       tc.Input,
		}

		result, err := opts.SplitFunc(tc.Input)
		if err != nil {
			updated[i].Error = err.Error()
			continue
		}

		updated[i].Result = make([]StatementResult, len(result))
		for j, stmt := range result {
			updated[i].Result[j] = StatementResult{
				Text:     stmt.Text,
				BaseLine: stmt.BaseLine(),
				Empty:    stmt.Empty,
			}
			if stmt.Start != nil {
				updated[i].Result[j].Start = PositionResult{
					Line:   stmt.Start.Line,
					Column: stmt.Start.Column,
				}
			}
			if stmt.End != nil {
				updated[i].Result[j].End = PositionResult{
					Line:   stmt.End.Line,
					Column: stmt.End.Column,
				}
			}
			if stmt.Range != nil {
				updated[i].Result[j].Range = RangeResult{
					Start: stmt.Range.Start,
					End:   stmt.Range.End,
				}
			}
		}
	}

	data, err := yaml.Marshal(updated)
	require.NoError(t, err, "failed to marshal updated test cases")

	err = os.WriteFile(fullPath, data, 0644)
	require.NoError(t, err, "failed to write updated test file: %s", fullPath)

	t.Logf("Updated %s with %d test cases", fullPath, len(updated))
}

func runSingleTest(t *testing.T, tc SplitTestCase, opts SplitTestOptions) {
	t.Helper()

	var lexerResult, parserResult []Statement
	var lexerErr, parserErr error

	// Run lexer implementation if provided
	if opts.LexerSplitFunc != nil {
		lexerResult, lexerErr = opts.LexerSplitFunc(tc.Input)
	}

	// Run parser implementation if provided
	if opts.ParserSplitFunc != nil {
		parserResult, parserErr = opts.ParserSplitFunc(tc.Input)
	}

	// MANDATORY: If both implementations succeed, results must be identical
	if opts.LexerSplitFunc != nil && opts.ParserSplitFunc != nil {
		if lexerErr == nil && parserErr == nil {
			require.Equal(t, len(lexerResult), len(parserResult),
				"lexer and parser produced different number of statements")
			for i := range lexerResult {
				require.Equal(t, lexerResult[i], parserResult[i],
					"lexer and parser produced different result for statement %d", i)
			}
		}
	}

	// Run the main split function
	result, err := opts.SplitFunc(tc.Input)

	// Check error expectation
	if tc.Error != "" {
		require.Error(t, err)
		require.Contains(t, err.Error(), tc.Error)
		return
	}

	require.NoError(t, err)
	assertStatementsEqual(t, tc.Result, result)
}
