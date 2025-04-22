package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Position represents a position in a document
type Position struct {
	Line   uint32
	Column uint32
}

func TestGetStatementRanges(t *testing.T) {
	testCases := []struct {
		name              string
		statement         string
		expectedRanges    int
		expectedPositions []Position
	}{
		{
			name:           "Single statement",
			statement:      "SELECT * FROM users;",
			expectedRanges: 1,
			// Skip detailed position testing as it depends on the implementation
			// of the statement splitting and parsing
		},
		{
			name:           "Multiple statements",
			statement:      "SELECT * FROM users; INSERT INTO orders VALUES (1, 2, 100.00);",
			expectedRanges: 2,
			// Skip detailed position testing
		},
		{
			name:           "Statements with comments",
			statement:      "-- This is a comment\nSELECT * FROM users;\n/* Block comment */\nDELETE FROM users WHERE id = 1;",
			expectedRanges: 2,
			// Skip detailed position testing
		},
		{
			name: "Multi-line statements",
			statement: `SELECT
    id,
    name
FROM
    users
WHERE
    id > 10;
    
INSERT INTO orders (id, user_id, total)
VALUES (1, 2, 100.00);`,
			expectedRanges: 2,
		},
		{
			name:           "Empty statement",
			statement:      "",
			expectedRanges: 0,
		},
		{
			name:           "Statement with quoted semicolons",
			statement:      "SELECT 'test;' as text; SELECT * FROM users;",
			expectedRanges: 2,
		},
		{
			name:           "Statement with semicolons in string literals",
			statement:      "SELECT 'Hello; World' FROM dual; SELECT 'Another; String' FROM dual;",
			expectedRanges: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			sCtx := base.StatementRangeContext{}

			// Call the statement range function
			ranges, err := GetStatementRanges(ctx, sCtx, tc.statement)
			require.NoError(t, err, "GetStatementRanges should not error")

			// Verify the number of ranges matches expected
			assert.Equal(t, tc.expectedRanges, len(ranges), "Number of statement ranges doesn't match")

			// Additional checks for multi-line statements
			if tc.name == "Multi-line statements" && len(ranges) >= 2 {
				assert.True(t, ranges[0].End.Line > ranges[0].Start.Line, "First statement should span multiple lines")
				assert.True(t, ranges[1].End.Line > ranges[1].Start.Line, "Second statement should span multiple lines")
			}
		})
	}
}
