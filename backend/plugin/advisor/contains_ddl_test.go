package advisor_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Register parsers and statement type getters via init().
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

func TestContainsDDL(t *testing.T) {
	tests := []struct {
		name    string
		engine  storepb.Engine
		sql     string
		wantDDL bool
	}{
		{
			name:    "PostgreSQL: DML only",
			engine:  storepb.Engine_POSTGRES,
			sql:     "INSERT INTO t VALUES (1); UPDATE t SET a = 1; DELETE FROM t;",
			wantDDL: false,
		},
		{
			name:    "PostgreSQL: DDL only",
			engine:  storepb.Engine_POSTGRES,
			sql:     "CREATE TABLE t (id INT);",
			wantDDL: true,
		},
		{
			name:    "PostgreSQL: mixed DDL and DML",
			engine:  storepb.Engine_POSTGRES,
			sql:     "CREATE TABLE t (id INT); INSERT INTO t VALUES (1);",
			wantDDL: true,
		},
		{
			name:    "PostgreSQL: SET ROLE should not be treated as DDL",
			engine:  storepb.Engine_POSTGRES,
			sql:     "SET ROLE 'admin'; INSERT INTO t VALUES (1);",
			wantDDL: false,
		},
		{
			name:    "MySQL: DML only",
			engine:  storepb.Engine_MYSQL,
			sql:     "INSERT INTO t VALUES (1); UPDATE t SET a = 1;",
			wantDDL: false,
		},
		{
			name:    "MySQL: mixed DDL and DML",
			engine:  storepb.Engine_MYSQL,
			sql:     "CREATE TABLE t (id INT); INSERT INTO t VALUES (1);",
			wantDDL: true,
		},
		{
			// SET classifies as StatementType_SET (not UNSPECIFIED) since BYT-9832, but is
			// still a session-setting utility, not DDL — so a SET+DML script must NOT be
			// treated as containing DDL (otherwise the DML dry run would be skipped).
			name:    "MySQL: SET should not be treated as DDL",
			engine:  storepb.Engine_MYSQL,
			sql:     "SET @a = 1; INSERT INTO t VALUES (1);",
			wantDDL: false,
		},
		{
			// A session-mode SET (the exact form the SDL routine/event preamble emits) is
			// likewise not DDL.
			name:    "MySQL: SET sql_mode should not be treated as DDL",
			engine:  storepb.Engine_MYSQL,
			sql:     "SET sql_mode='ANSI_QUOTES'; INSERT INTO t VALUES (1);",
			wantDDL: false,
		},
		{
			name:    "TiDB: DML only",
			engine:  storepb.Engine_TIDB,
			sql:     "INSERT INTO t VALUES (1); DELETE FROM t;",
			wantDDL: false,
		},
		{
			name:    "TiDB: mixed DDL and DML",
			engine:  storepb.Engine_TIDB,
			sql:     "CREATE TABLE t (id INT); INSERT INTO t VALUES (1);",
			wantDDL: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stmts, err := base.ParseStatements(tc.engine, tc.sql)
			require.NoError(t, err)
			got := advisor.ContainsDDL(tc.engine, stmts)
			require.Equal(t, tc.wantDDL, got)
		})
	}
}
