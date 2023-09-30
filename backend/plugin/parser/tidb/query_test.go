package tidb

import (
	"testing"

	_ "github.com/pingcap/tidb/types/parser_driver"
	"github.com/stretchr/testify/require"
)

func TestMySQLExtractDatabaseList(t *testing.T) {
	tests := []struct {
		stmt string
		want []string
	}{
		{
			stmt: `
				SELECT * FROM t;
				SELECT * FROM db1.t;
				SELECT * FROM db2.t;
				SELECT * FROM db1.t;
			`,
			want: []string{"", "db1", "db2"},
		},
		{
			stmt: `
				SELECT 1;
			`,
			want: nil,
		},
		{
			stmt: `SELECT * FROM t, db1.t1;`,
			want: []string{"", "db1"},
		},
		{
			stmt: `select * from t join db1.t as t1 where t.a != t1.a;`,
			want: []string{"", "db1"},
		},
	}

	for _, test := range tests {
		res, err := ExtractDatabaseList(test.stmt, "")
		require.NoError(t, err)
		require.Equal(t, test.want, res)
	}
}
