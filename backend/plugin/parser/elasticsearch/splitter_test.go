package elasticsearch

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type yamlPosition struct {
	Line   int32 `yaml:"line,omitempty"`
	Column int32 `yaml:"column,omitempty"`
}

type yamlRange struct {
	Start int32 `yaml:"start,omitempty"`
	End   int32 `yaml:"end,omitempty"`
}

type yamlStatement struct {
	Text  string        `yaml:"text,omitempty"`
	Start *yamlPosition `yaml:"start,omitempty"`
	End   *yamlPosition `yaml:"end,omitempty"`
	Range *yamlRange    `yaml:"range,omitempty"`
}

func TestSplitMultiSQL(t *testing.T) {
	type testCase struct {
		Description   string          `yaml:"description,omitempty"`
		Statement     string          `yaml:"statement,omitempty"`
		ExpectedCount int             `yaml:"expectedCount,omitempty"`
		Statements    []yamlStatement `yaml:"statements,omitempty"`
	}

	const (
		testDataPath = "test-data/splitter.yaml"
	)

	a := require.New(t)

	yamlFile, err := os.Open(testDataPath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yamlFile.Close())

	var testCases []testCase
	a.NoError(yaml.Unmarshal(byteValue, &testCases))

	for _, tc := range testCases {
		t.Run(tc.Description, func(_ *testing.T) {
			statements, err := SplitMultiSQL(tc.Statement)
			a.NoError(err)
			a.Len(statements, tc.ExpectedCount)

			for i, expected := range tc.Statements {
				if i >= len(statements) {
					break
				}
				stmt := statements[i]

				if expected.Text != "" {
					a.Equal(expected.Text, stmt.Text, "statement %d text", i)
				}

				if expected.Start != nil {
					a.NotNil(stmt.Start, "statement %d should have Start position", i)
					a.Equal(expected.Start.Line, stmt.Start.Line, "statement %d start line", i)
					a.Equal(expected.Start.Column, stmt.Start.Column, "statement %d start column", i)
				}

				if expected.End != nil {
					a.NotNil(stmt.End, "statement %d should have End position", i)
					a.Equal(expected.End.Line, stmt.End.Line, "statement %d end line", i)
					a.Equal(expected.End.Column, stmt.End.Column, "statement %d end column", i)
				}

				if expected.Range != nil {
					a.NotNil(stmt.Range, "statement %d should have Range", i)
					a.Equal(expected.Range.Start, stmt.Range.Start, "statement %d range start", i)
					a.Equal(expected.Range.End, stmt.Range.End, "statement %d range end", i)
				}
			}

			// Verify all statements have required fields
			for i, stmt := range statements {
				a.NotNil(stmt.Start, "statement %d should have Start", i)
				a.NotNil(stmt.End, "statement %d should have End", i)
				a.NotNil(stmt.Range, "statement %d should have Range", i)
				a.False(stmt.Empty, "statement %d should not be empty", i)
			}
		})
	}
}

// Keep this for compatibility with existing tests that use the function directly
func TestSplitMultiSQLDirect(t *testing.T) {
	tests := []struct {
		name           string
		statement      string
		expectedCount  int
		expectedStarts []*storepb.Position
	}{
		{
			name:          "empty statement",
			statement:     "",
			expectedCount: 0,
		},
		{
			name:          "single GET request",
			statement:     "GET _search",
			expectedCount: 1,
			expectedStarts: []*storepb.Position{
				{Line: 1, Column: 1},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statements, err := SplitMultiSQL(tc.statement)
			require.NoError(t, err)
			require.Len(t, statements, tc.expectedCount)

			if tc.expectedStarts != nil {
				for i, expectedStart := range tc.expectedStarts {
					require.NotNil(t, statements[i].Start, "statement %d should have Start position", i)
					require.Equal(t, expectedStart.Line, statements[i].Start.Line, "statement %d start line", i)
					require.Equal(t, expectedStart.Column, statements[i].Start.Column, "statement %d start column", i)
				}
			}
		})
	}
}
