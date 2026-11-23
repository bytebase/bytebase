package standard

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateQuerySeam(t *testing.T) {
	require.True(t, base.HasQueryValidator(storepb.Engine_CLICKHOUSE))
	require.True(t, base.HasQueryValidator(storepb.Engine_HIVE))
	require.False(t, base.HasQueryValidator(storepb.Engine_DATABRICKS))

	cases := []struct {
		name     string
		engine   storepb.Engine
		sql      string
		readOnly bool
		allQuery bool
		wantErr  bool
	}{
		{
			name:     "clickhouse read-only select",
			engine:   storepb.Engine_CLICKHOUSE,
			sql:      "SELECT * FROM t WHERE a = 'insert'",
			readOnly: true,
			allQuery: true,
		},
		{
			name:     "clickhouse explain select",
			engine:   storepb.Engine_CLICKHOUSE,
			sql:      "EXPLAIN SELECT * FROM t;",
			readOnly: true,
			allQuery: true,
		},
		{
			name:   "clickhouse explain analyze",
			engine: storepb.Engine_CLICKHOUSE,
			sql:    "EXPLAIN ANALYZE SELECT * FROM t",
		},
		{
			name:   "clickhouse ddl",
			engine: storepb.Engine_CLICKHOUSE,
			sql:    "CREATE TABLE t (a Int32) ENGINE = Memory;",
		},
		{
			name:   "clickhouse multi-statement",
			engine: storepb.Engine_CLICKHOUSE,
			sql:    "SELECT 1; SELECT 2;",
		},
		{
			name:    "clickhouse syntax error",
			engine:  storepb.Engine_CLICKHOUSE,
			sql:     "SELECT 1 FROM",
			wantErr: true,
		},
		{
			name:     "hive read-only select",
			engine:   storepb.Engine_HIVE,
			sql:      "WITH c AS (SELECT 1 AS x) SELECT * FROM c",
			readOnly: true,
			allQuery: true,
		},
		{
			name:   "hive cte insert",
			engine: storepb.Engine_HIVE,
			sql:    "WITH c AS (SELECT 1 AS x) INSERT INTO t SELECT * FROM c",
		},
		{
			name:   "hive explain analyze",
			engine: storepb.Engine_HIVE,
			sql:    "EXPLAIN ANALYZE SELECT 1",
		},
		{
			name:   "hive multi-statement with write",
			engine: storepb.Engine_HIVE,
			sql:    "SELECT 1; DROP TABLE t;",
		},
		{
			name:    "hive syntax error",
			engine:  storepb.Engine_HIVE,
			sql:     "SELECT FROM",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			readOnly, allQuery, err := base.ValidateSQLForEditor(tc.engine, tc.sql)
			if tc.wantErr {
				require.Error(t, err)
				require.False(t, readOnly)
				require.False(t, allQuery)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.readOnly, readOnly)
			require.Equal(t, tc.allQuery, allQuery)
		})
	}
}
