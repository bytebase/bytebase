package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListDatabaseFilterInstanceScope(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "project instance",
			filter:   `instance == "projects/project-a/instances/instance-a"`,
			wantSQL:  "(db.instance = $1 AND instance.project = $2)",
			wantArgs: []any{"instance-a", "project-a"},
		},
		{
			name:     "workspace instance",
			filter:   `instance == "instances/instance-a"`,
			wantSQL:  "(db.instance = $1 AND instance.project IS NULL)",
			wantArgs: []any{"instance-a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := GetListDatabaseFilter("default", tt.filter)
			require.NoError(t, err)

			sql, args, err := query.ToSQL()
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, sql)
			require.Equal(t, tt.wantArgs, args)
		})
	}
}
