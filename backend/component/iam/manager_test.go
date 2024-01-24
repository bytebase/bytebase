package iam

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPartialEval(t *testing.T) {
	a := require.New(t)

	time20240201, err := time.Parse(time.RFC3339, "2024-02-01T00:00:00Z")
	a.NoError(err)

	testCases := []struct {
		name  string
		expr  string
		input map[string]any
		want  bool
	}{
		{
			name:  "simple false 1",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\")",
			input: map[string]any{"request.time": time20240201},
			want:  false,
		},
		{
			name:  "simple false 2",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\")",
			input: map[string]any{"request.time": time20240201.AddDate(0, 0, 1)},
			want:  false,
		},
		{
			name:  "simple true 1",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\")",
			input: map[string]any{"request.time": time20240201.AddDate(0, 0, -1)},
			want:  true,
		},
		{
			name:  "partial false 1",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\") && request.row_limit <= 1000",
			input: map[string]any{"request.time": time20240201},
			want:  false,
		},
		{
			name:  "partial false 2",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\") && (resource.database in [\"instances/dbdbdb/databases/db_106\",\"instances/dbdbdb/databases/db_108\",\"instances/dbdbdb/databases/db_103\"])",
			input: map[string]any{"request.time": time20240201},
			want:  false,
		},
		{
			name:  "partial true 1",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\") && request.row_limit <= 1000",
			input: map[string]any{"request.time": time20240201.AddDate(0, 0, -1)},
			want:  true,
		},
		{
			name:  "partial true 2",
			expr:  "request.time < timestamp(\"2024-02-01T00:00:00Z\") && (resource.database in [\"instances/dbdbdb/databases/db_106\",\"instances/dbdbdb/databases/db_108\",\"instances/dbdbdb/databases/db_103\"])",
			input: map[string]any{"request.time": time20240201.AddDate(0, 0, -1)},
			want:  true,
		},
	}

	for _, tc := range testCases {
		res, err := evalMemberCondition(tc.expr, tc.input)
		a.NoError(err)
		a.Equal(tc.want, res, tc.name)
	}

}
