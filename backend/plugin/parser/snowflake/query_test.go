package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditor(t *testing.T) {
	tests := []struct {
		statement   string
		valid       bool
		gotAllQuery bool
	}{
		{statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;", valid: true, gotAllQuery: true},
		{statement: "SET max_execution_time = 1000; select * from t", valid: true, gotAllQuery: false},
		{statement: "UPDATE t1 SET c1 = 1;", valid: false, gotAllQuery: false},
	}
	for _, test := range tests {
		gotValid, gotAllQuery, err := base.ValidateSQLForEditor(storepb.Engine_SNOWFLAKE, test.statement)
		require.NoError(t, err, test.statement)
		require.Equal(t, test.valid, gotValid, test.statement)
		require.Equal(t, test.gotAllQuery, gotAllQuery, test.statement)
	}
}
