package taskcheck

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAffectedRowsCountForMysql(t *testing.T) {
	tests := []struct {
		name string
		want int64
		res  []any
	}{
		{
			name: "delete simple",
			want: 123,
			res: []any{
				[]string{
					"id",
					"select_type",
					"table",
					"partitions",
					"type",
					"possible_keys",
					"key",
					"key_len",
					"ref",
					"rows",
					"filtered",
					"Extra",
				},
				[]string{
					"UNSIGNED BIGINT",
					"VARCHAR",
					"VARCHAR",
					"MEDIUMTEXT",
					"VARCHAR",
					"VARCHAR",
					"VARCHAR",
					"VARCHAR",
					"VARCHAR",
					"UNSIGNED BIGINT",
					"DOUBLE",
					"VARCHAR",
				},
				[]any{
					[]any{1, "SIMPLE", "<subquery2>", nil, "ALL", nil, nil, nil, nil, nil, 100.0, "Using temporary"},
					[]any{1, "DELETE", "ha", nil, "ALL", nil, nil, nil, nil, 123, 100.0, "Using where"},
					[]any{2, "MATERIALIZED", "haha", nil, "ALL", nil, nil, nil, nil, 5423, 100.0, "Using where"},
					[]any{2, "MATERIALIZED", "hahaha", nil, "ALL", nil, nil, nil, nil, 54321, 100.0, "Using where"},
					[]any{2, "MATERIALIZED", "hahahaha", nil, "ALL", nil, nil, nil, nil, 856, 100.0, "Range checked for each record (index map: 0x2)"},
				},
			},
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getAffectedRowsCountForMysql(tc.res)
			a.NoError(err)
			a.Equal(tc.want, got)
		})
	}
}
