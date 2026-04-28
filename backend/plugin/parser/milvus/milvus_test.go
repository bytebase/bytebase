package milvus

import (
	"strings"
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

func TestSplitSQL_JSONPayload(t *testing.T) {
	sql := `search c1 with {"note":"keep;inside","data":[[0.1,0.2,0.3,0.4]],"params":{"expr":"id > 0; id < 10"}};show collections;`
	stmts, err := SplitSQL(sql)
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, `search c1 with {"note":"keep;inside","data":[[0.1,0.2,0.3,0.4]],"params":{"expr":"id > 0; id < 10"}};`, stmts[0].Text)
	require.Equal(t, "show collections;", strings.TrimSpace(stmts[1].Text))
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		stmt  string
		valid bool
	}{
		{stmt: "select * from c1", valid: true},
		{stmt: "show collections", valid: true},
		{stmt: `search c1 with {"data":[[0.1,0.2,0.3,0.4]],"annsField":"vector"}`, valid: true},
		{stmt: `hybrid search c1 with {"search":[{"data":[[0.1,0.2,0.3,0.4]],"annsField":"vector","limit":2}],"rerank":{"strategy":"rrf","params":{"k":60}}}`, valid: true},
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
