package mongodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseMongoShell(t *testing.T) {
	testCases := []struct {
		description     string
		statement       string
		wantStatements  int
		wantHasErrors   bool
		wantStartOffset int
		wantEndOffset   int
	}{
		{
			description:     "single find statement",
			statement:       `db.collection.find({})`,
			wantStatements:  1,
			wantHasErrors:   false,
			wantStartOffset: 0,
			wantEndOffset:   22,
		},
		{
			description:     "single find with filter",
			statement:       `db.users.find({ name: "John" })`,
			wantStatements:  1,
			wantHasErrors:   false,
			wantStartOffset: 0,
			wantEndOffset:   31,
		},
		{
			description:    "multiple statements",
			statement:      "db.users.find({});\ndb.products.insertOne({ name: \"test\" })",
			wantStatements: 2,
			wantHasErrors:  false,
		},
		{
			description:    "show databases command",
			statement:      "show dbs",
			wantStatements: 1,
			wantHasErrors:  false,
		},
		{
			description:    "show collections command",
			statement:      "show collections",
			wantStatements: 1,
			wantHasErrors:  false,
		},
		{
			description:    "aggregation pipeline",
			statement:      `db.orders.aggregate([{ $match: { status: "A" } }, { $group: { _id: "$cust_id", total: { $sum: "$amount" } } }])`,
			wantStatements: 1,
			wantHasErrors:  false,
		},
		{
			description:    "bracket notation for collection",
			statement:      `db["my-collection"].find({})`,
			wantStatements: 1,
			wantHasErrors:  false,
		},
		{
			description:    "method chaining",
			statement:      `db.users.find({}).sort({ name: 1 }).limit(10)`,
			wantStatements: 1,
			wantHasErrors:  false,
		},
		{
			description:   "syntax error - missing closing paren",
			statement:     `db.collection.find({`,
			wantHasErrors: true,
		},
		{
			description:    "empty input",
			statement:      "",
			wantStatements: 0,
			wantHasErrors:  false,
		},
		{
			description:    "whitespace only",
			statement:      "   \n\t  ",
			wantStatements: 0,
			wantHasErrors:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := ParseMongoShell(tc.statement)
			require.NotNil(t, result)

			if !tc.wantHasErrors {
				require.Len(t, result.Statements, tc.wantStatements, "statement count mismatch")
				require.Empty(t, result.Errors, "expected no errors")
			} else {
				require.NotEmpty(t, result.Errors, "expected errors")
			}

			if tc.wantStatements > 0 && tc.wantStartOffset != 0 && tc.wantEndOffset != 0 {
				require.Equal(t, tc.wantStartOffset, result.Statements[0].StartOffset)
				require.Equal(t, tc.wantEndOffset, result.Statements[0].EndOffset)
			}
		})
	}
}

func TestGetStatementRanges(t *testing.T) {
	testCases := []struct {
		description string
		statement   string
		wantRanges  int
	}{
		{
			description: "single statement",
			statement:   `db.collection.find({})`,
			wantRanges:  1,
		},
		{
			description: "multiple statements on separate lines",
			statement:   "db.users.find({});\ndb.products.find({})",
			wantRanges:  2,
		},
		{
			description: "multiline statement",
			statement: `db.users.aggregate([
  { $match: { status: "A" } },
  { $group: { _id: "$cust_id" } }
])`,
			wantRanges: 1,
		},
		{
			description: "empty input",
			statement:   "",
			wantRanges:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ranges, err := GetStatementRanges(context.Background(), base.StatementRangeContext{}, tc.statement)
			require.NoError(t, err)
			require.Len(t, ranges, tc.wantRanges)
		})
	}
}

func TestGetStatementRangesWithHindiCharacters(t *testing.T) {
	// Test with Hindi characters - ANTLR returns character (rune) offsets, not byte offsets
	// The Hindi collection name "मिलन-भेंट" has 9 runes but 25 bytes
	statement := "db[\"मिलन-भेंट\"].find()\ndb.coll.find()"

	ranges, err := GetStatementRanges(context.Background(), base.StatementRangeContext{}, statement)
	require.NoError(t, err)
	require.Len(t, ranges, 2)

	// First statement: db["मिलन-भेंट"].find()
	// Starts at line 0, char 0
	// Ends at line 0, char 22 (after the closing paren)
	require.Equal(t, uint32(0), ranges[0].Start.Line)
	require.Equal(t, uint32(0), ranges[0].Start.Character)
	require.Equal(t, uint32(0), ranges[0].End.Line)
	require.Equal(t, uint32(22), ranges[0].End.Character)

	// Second statement: db.coll.find()
	// Starts at line 1, char 0
	// Ends at line 1, char 14
	require.Equal(t, uint32(1), ranges[1].Start.Line)
	require.Equal(t, uint32(0), ranges[1].Start.Character)
	require.Equal(t, uint32(1), ranges[1].End.Line)
	require.Equal(t, uint32(14), ranges[1].End.Character)
}

func TestDiagnose(t *testing.T) {
	testCases := []struct {
		description        string
		statement          string
		wantHasDiagnostics bool
	}{
		{
			description:        "valid statement",
			statement:          `db.collection.find({})`,
			wantHasDiagnostics: false,
		},
		{
			description:        "syntax error - unclosed brace",
			statement:          `db.collection.find({`,
			wantHasDiagnostics: true,
		},
		{
			description:        "syntax error - invalid token",
			statement:          `db.collection.find(@@@@)`,
			wantHasDiagnostics: true,
		},
		{
			description:        "empty input",
			statement:          "",
			wantHasDiagnostics: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			diagnostics, err := Diagnose(context.Background(), base.DiagnoseContext{}, tc.statement)
			require.NoError(t, err)
			if tc.wantHasDiagnostics {
				require.NotEmpty(t, diagnostics, "expected diagnostics")
			} else {
				require.Empty(t, diagnostics, "expected no diagnostics")
			}
		})
	}
}
