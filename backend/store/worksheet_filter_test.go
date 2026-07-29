package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListWorksheetFilter(t *testing.T) {
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
		},
		{
			name:     "visibility filter",
			filter:   `visibility == "PROJECT_READ"`,
			wantSQL:  "(worksheet.visibility = $1)",
			wantArgs: []any{WorkSheetVisibility("PROJECT_READ")},
		},
		{
			name:     "visibility in filter",
			filter:   `visibility in ["PROJECT_READ", "PROJECT_WRITE"]`,
			wantSQL:  "(worksheet.visibility = ANY($1))",
			wantArgs: []any{[]string{"PROJECT_READ", "PROJECT_WRITE"}},
		},
		{
			name:        "starred unsupported",
			filter:      `starred == true`,
			wantErr:     true,
			errContains: `unsupported variable "starred"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetListWorksheetFilter(context.Background(), nil, "demo@example.com", tt.filter)

			if tt.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.errContains)
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
