package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListAccessGrantFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantSQL     string
		wantArgs    []any
		wantErr     bool
		errContains string
	}{
		{
			name:     "empty filter",
			filter:   "",
			wantSQL:  "",
			wantArgs: nil,
			wantErr:  false,
		},
		{
			name:     "query contains",
			filter:   `query.contains("SELECT * FROM users")`,
			wantSQL:  "(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g') ILIKE $1)",
			wantArgs: []any{"%SELECT * FROM users%"},
			wantErr:  false,
		},
		{
			name:     "query contains normalizes whitespace",
			filter:   `query.contains("SELECT   *   FROM users")`,
			wantSQL:  "(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g') ILIKE $1)",
			wantArgs: []any{"%SELECT * FROM users%"},
			wantErr:  false,
		},
		{
			name:        "matches is unsupported",
			filter:      `query.matches("SELECT")`,
			wantErr:     true,
			errContains: "unsupported function matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetListAccessGrantFilter(tt.filter)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.filter == "" {
				require.Nil(t, q)
				return
			}

			require.NotNil(t, q)
			sql, args, err := q.ToSQL()
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, sql)
			require.Equal(t, tt.wantArgs, args)
		})
	}
}
