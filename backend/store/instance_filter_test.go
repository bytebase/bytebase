package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListInstanceFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantSQL     string
		wantArgs    []any
		wantErr     bool
		errContains string
	}{
		{
			name:     "name contains",
			filter:   `name.contains("sample")`,
			wantSQL:  "(LOWER(instance.metadata->>'title') LIKE $1)",
			wantArgs: []any{"%sample%"},
		},
		{
			name:     "resource_id contains",
			filter:   `resource_id.contains("prod")`,
			wantSQL:  "(LOWER(instance.resource_id) LIKE $1)",
			wantArgs: []any{"%prod%"},
		},
		{
			name:     "host contains",
			filter:   `host.contains("127.0")`,
			wantSQL:  "(EXISTS (SELECT 1 FROM jsonb_array_elements(instance.metadata -> 'dataSources') AS ds WHERE ds ->> 'host' LIKE $1))",
			wantArgs: []any{"%127.0%"},
		},
		{
			name:     "port contains",
			filter:   `port.contains("543")`,
			wantSQL:  "(EXISTS (SELECT 1 FROM jsonb_array_elements(instance.metadata -> 'dataSources') AS ds WHERE ds ->> 'port' LIKE $1))",
			wantArgs: []any{"%543%"},
		},
		{
			name:        "matches is unsupported",
			filter:      `name.matches("sample")`,
			wantErr:     true,
			errContains: "unexpected function matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetListInstanceFilter(tt.filter)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, q)

			sql, args, err := q.ToSQL()
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, sql)
			require.Equal(t, tt.wantArgs, args)
		})
	}
}
