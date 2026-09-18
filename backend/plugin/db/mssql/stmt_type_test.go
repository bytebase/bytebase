package mssql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetStmtType(t *testing.T) {
	for _, tc := range []struct {
		stmt    string
		want    stmtType
		wantErr string
	}{
		{stmt: "SELECT id FROM t", want: stmtTypeResultSetGenerating | stmtTypeRowCountGenerating},
		{stmt: "SELECT id INTO t2 FROM t", want: stmtTypeRowCountGenerating},
		{stmt: "WITH c AS (SELECT id FROM t) SELECT @id = id FROM c", want: stmtTypeUnknown},
		{stmt: "UPDATE t SET v = 1 OUTPUT inserted.id", want: stmtTypeResultSetGenerating | stmtTypeRowCountGenerating | stmtTypeOutput},
		{stmt: "DELETE FROM t OUTPUT deleted.id INTO @deleted WHERE id = 1", want: stmtTypeRowCountGenerating},
		{stmt: "INSERT INTO t VALUES (1)", want: stmtTypeRowCountGenerating},
		{stmt: "EXEC dbo.p @id = 1", want: stmtTypeProcedure},
		{stmt: "SET NOCOUNT ON", want: stmtTypeUnknown},
		{stmt: "SET NOEXEC ON", wantErr: "SET NOEXEC is not supported"},
		{stmt: "set parseonly on", wantErr: "SET PARSEONLY is not supported"},
		{stmt: "IF 1 = 1 SELECT 1", wantErr: "unsupported control flow statement"},
	} {
		t.Run(tc.stmt, func(t *testing.T) {
			got, err := getStmtType(tc.stmt)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
