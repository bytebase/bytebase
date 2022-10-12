package mysql

import (
	"testing"

	"github.com/pingcap/tidb/parser"
	"github.com/stretchr/testify/require"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

func TestValidateStmtNodes(t *testing.T) {
	tests := []struct {
		stmt    string
		wantErr bool
	}{
		{
			stmt:    "CREATE TABLE t(a INT PRIMARY KEY);",
			wantErr: true,
		},
		{
			stmt:    "CREATE TABLE t(a INT, CONSTRAINT PRIMARY KEY(a));",
			wantErr: false,
		},
		{
			stmt:    "CREATE TABLE t(a INT UNIQUE);",
			wantErr: true,
		},
		{
			stmt:    "CREATE TABLE t(a INT, CONSTRAINT UNIQUE KEY(a));",
			wantErr: true,
		},
		{
			stmt:    "CREATE TABLE t(a INT, CONSTRAINT UNIQUE KEY uniq_idx_a(a));",
			wantErr: false,
		},
		{
			stmt:    "CREATE TABLE t(a INT, INDEX(a));",
			wantErr: true,
		},
		{
			stmt:    "CREATE TABLE t(a INT, INDEX idx_a(a));",
			wantErr: false,
		},
	}

	a := require.New(t)
	for _, test := range tests {
		nodes, _, err := parser.New().Parse(test.stmt, "", "")
		a.NoError(err)
		got := validateStmtNodes(nodes)
		if test.wantErr {
			a.Errorf(got, "stmt", test.stmt)
		} else {
			a.NoErrorf(got, "stmt", test.stmt)
		}
	}
}
