package cosmosdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCosmosDBQuery(t *testing.T) {
	tests := []struct {
		name          string
		statement     string
		wantStatCount int
		wantErr       bool
	}{
		{
			name:          "Single SELECT statement",
			statement:     "SELECT * FROM c",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "SELECT with WHERE clause",
			statement:     "SELECT c.id, c.name FROM users c WHERE c.age > 18",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "SELECT with DISTINCT",
			statement:     "SELECT DISTINCT c.category FROM products c",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Empty statement",
			statement:     "",
			wantStatCount: 0,
			wantErr:       false,
		},
		{
			name:          "Only whitespace",
			statement:     "   \n  \t  ",
			wantStatCount: 0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseCosmosDBQuery(tt.statement)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatCount, len(results))

			// Verify each result has the required fields
			for i, result := range results {
				assert.NotNil(t, result.Tree, "Result %d should have a Tree", i)
				assert.NotNil(t, result.Tokens, "Result %d should have Tokens", i)
				assert.NotNil(t, result.StartPosition, "Result %d should have StartPosition", i)
				assert.Equal(t, int32(1), result.StartPosition.Line, "Result %d should have StartPosition.Line 1", i)
			}
		})
	}
}

func TestParseCosmosDBQueryErrors(t *testing.T) {
	tests := []struct {
		name      string
		statement string
	}{
		{
			name:      "Invalid SQL syntax",
			statement: "SELCT * FRM c",
		},
		{
			name:      "Unclosed string",
			statement: "SELECT * FROM c WHERE c.name = 'test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCosmosDBQuery(tt.statement)
			require.Error(t, err, "Expected error for invalid CosmosDB SQL")
		})
	}
}
