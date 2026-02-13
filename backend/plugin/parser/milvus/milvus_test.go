package milvus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitSQL(t *testing.T) {
	stmts, err := SplitSQL("select * from c1;  show collections;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, "select * from c1;", stmts[0].Text)
	require.Equal(t, "  show collections;", stmts[1].Text)
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		stmt  string
		valid bool
	}{
		{stmt: "select * from c1", valid: true},
		{stmt: "show collections", valid: true},
		{stmt: "with t as (select 1) select * from t", valid: true},
		{stmt: "insert into c1 values (1)", valid: false},
		{stmt: "update c1 set a = 1", valid: false},
		{stmt: "drop collection c1", valid: false},
	}
	for _, tc := range tests {
		gotValid, gotAllQuery, err := validateQuery(tc.stmt)
		require.NoError(t, err, tc.stmt)
		require.Equal(t, tc.valid, gotValid, tc.stmt)
		require.Equal(t, tc.valid, gotAllQuery, tc.stmt)
	}
}

func TestParseStatements(t *testing.T) {
	stmts, err := parseStatements("select * from c1;")
	require.NoError(t, err)
	require.Len(t, stmts, 1)
	require.NotNil(t, stmts[0].AST)
}
