package standard

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainStatementSeam(t *testing.T) {
	cases := []struct {
		name      string
		engine    storepb.Engine
		statement string
		format    base.ExplainFormat
		want      string
	}{
		{
			name:      "clickhouse json",
			engine:    storepb.Engine_CLICKHOUSE,
			statement: "SELECT 1",
			format:    base.ExplainFormatJSON,
			want:      "EXPLAIN json = 1, indexes = 1 SELECT 1",
		},
		{
			name:      "clickhouse default",
			engine:    storepb.Engine_CLICKHOUSE,
			statement: "SELECT 1",
			format:    base.ExplainFormatDefault,
			want:      "EXPLAIN SELECT 1",
		},
		{
			name:      "hive json",
			engine:    storepb.Engine_HIVE,
			statement: "SELECT 1",
			format:    base.ExplainFormatJSON,
			want:      "EXPLAIN FORMATTED SELECT 1",
		},
		{
			name:      "hive default",
			engine:    storepb.Engine_HIVE,
			statement: "SELECT 1",
			format:    base.ExplainFormatDefault,
			want:      "EXPLAIN SELECT 1",
		},
		{
			name:      "databricks formatted",
			engine:    storepb.Engine_DATABRICKS,
			statement: "SELECT 1",
			format:    base.ExplainFormatDefault,
			want:      "EXPLAIN FORMATTED SELECT 1",
		},
		{
			name:      "databricks preserves existing explain",
			engine:    storepb.Engine_DATABRICKS,
			statement: "EXPLAIN EXTENDED SELECT 1",
			format:    base.ExplainFormatJSON,
			want:      "EXPLAIN EXTENDED SELECT 1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := base.ExplainStatement(tc.engine, tc.statement, tc.format)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
