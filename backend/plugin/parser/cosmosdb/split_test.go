package cosmosdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitSQL(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantCount int
		wantEmpty bool
	}{
		{
			name:      "Single SELECT statement",
			statement: "SELECT * FROM c",
			wantCount: 1,
			wantEmpty: false,
		},
		{
			name:      "SELECT with WHERE",
			statement: "SELECT c.id FROM users c WHERE c.active = true",
			wantCount: 1,
			wantEmpty: false,
		},
		{
			name:      "Empty string",
			statement: "",
			wantCount: 0,
		},
		{
			name:      "Only whitespace",
			statement: "   \n  \t  ",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := SplitSQL(tt.statement)
			require.NoError(t, err)
			require.Equal(t, tt.wantCount, len(list))

			if tt.wantCount > 0 {
				require.Equal(t, tt.statement, list[0].Text)
				require.Equal(t, 0, list[0].GetBaseLine())
				require.Equal(t, tt.wantEmpty, list[0].Empty)
				require.NotNil(t, list[0].Start)
				require.NotNil(t, list[0].End)
			}
		})
	}
}
