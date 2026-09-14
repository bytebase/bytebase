package cockroachdb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestGetStatementTypes(t *testing.T) {
	for _, tc := range []struct {
		statement string
		want      []storepb.StatementType
	}{
		{statement: `CREATE DATABASE d`, want: []storepb.StatementType{storepb.StatementType_CREATE_DATABASE}},
		{statement: `CREATE TABLE t (id INT PRIMARY KEY)`, want: []storepb.StatementType{storepb.StatementType_CREATE_TABLE}},
		{statement: `CREATE TABLE t2 AS SELECT * FROM t`, want: []storepb.StatementType{storepb.StatementType_CREATE_TABLE}},
		{statement: `CREATE VIEW v AS SELECT * FROM t`, want: []storepb.StatementType{storepb.StatementType_CREATE_VIEW}},
		{statement: `CREATE MATERIALIZED VIEW mv AS SELECT * FROM t`, want: []storepb.StatementType{storepb.StatementType_CREATE_VIEW}},
		{statement: `CREATE INDEX ON t (c)`, want: []storepb.StatementType{storepb.StatementType_CREATE_INDEX}},
		{statement: `CREATE SEQUENCE s`, want: []storepb.StatementType{storepb.StatementType_CREATE_SEQUENCE}},
		{statement: `CREATE SCHEMA app`, want: []storepb.StatementType{storepb.StatementType_CREATE_SCHEMA}},
		{statement: `CREATE FUNCTION f() RETURNS INT LANGUAGE SQL AS 'SELECT 1'`, want: []storepb.StatementType{storepb.StatementType_CREATE_FUNCTION}},
		{statement: `CREATE PROCEDURE p() LANGUAGE SQL AS 'SELECT 1'`, want: []storepb.StatementType{storepb.StatementType_CREATE_FUNCTION}},
		{statement: `CREATE TRIGGER tr BEFORE INSERT ON t FOR EACH ROW EXECUTE FUNCTION f()`, want: []storepb.StatementType{storepb.StatementType_CREATE_TRIGGER}},
		{statement: `CREATE EXTENSION IF NOT EXISTS pg_trgm`, want: []storepb.StatementType{storepb.StatementType_CREATE_EXTENSION}},
		{statement: `CREATE TYPE e AS ENUM ('a', 'b')`, want: []storepb.StatementType{storepb.StatementType_CREATE_TYPE}},
		{statement: `DROP DATABASE d`, want: []storepb.StatementType{storepb.StatementType_DROP_DATABASE}},
		{statement: `DROP TABLE t`, want: []storepb.StatementType{storepb.StatementType_DROP_TABLE}},
		{statement: `DROP MATERIALIZED VIEW mv`, want: []storepb.StatementType{storepb.StatementType_DROP_TABLE}},
		{statement: `DROP VIEW v`, want: []storepb.StatementType{storepb.StatementType_DROP_VIEW}},
		{statement: `DROP INDEX t@idx`, want: []storepb.StatementType{storepb.StatementType_DROP_INDEX}},
		{statement: `DROP SEQUENCE s`, want: []storepb.StatementType{storepb.StatementType_DROP_SEQUENCE}},
		{statement: `DROP SCHEMA app`, want: []storepb.StatementType{storepb.StatementType_DROP_SCHEMA}},
		{statement: `DROP TYPE e`, want: []storepb.StatementType{storepb.StatementType_DROP_TYPE}},
		{statement: `DROP TRIGGER tr ON t`, want: []storepb.StatementType{storepb.StatementType_DROP_TRIGGER}},
		{statement: `DROP FUNCTION f`, want: []storepb.StatementType{storepb.StatementType_DROP_FUNCTION}},
		{statement: `TRUNCATE t`, want: []storepb.StatementType{storepb.StatementType_TRUNCATE}},
		{statement: `ALTER DATABASE d RENAME TO d2`, want: []storepb.StatementType{storepb.StatementType_ALTER_DATABASE}},
		{statement: `ALTER DATABASE d OWNER TO u`, want: []storepb.StatementType{storepb.StatementType_ALTER_DATABASE}},
		{statement: `ALTER DATABASE d PRIMARY REGION "us-east1"`, want: []storepb.StatementType{storepb.StatementType_ALTER_DATABASE}},
		{statement: `ALTER DATABASE d SURVIVE REGION FAILURE`, want: []storepb.StatementType{storepb.StatementType_ALTER_DATABASE}},
		{statement: `ALTER DATABASE d CONFIGURE ZONE USING num_replicas = 5`, want: []storepb.StatementType{storepb.StatementType_ALTER_DATABASE}},
		{statement: `ALTER DATABASE d SET timezone = 'UTC'`, want: []storepb.StatementType{storepb.StatementType_ALTER_DATABASE}},
		{statement: `ALTER ROLE u SET timezone = 'UTC'`},
		{statement: `ALTER TABLE t CONFIGURE ZONE USING num_replicas = 5`, want: []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{statement: `ALTER INDEX t@idx CONFIGURE ZONE USING num_replicas = 5`, want: []storepb.StatementType{storepb.StatementType_ALTER_INDEX}},
		{statement: `ALTER RANGE default CONFIGURE ZONE USING num_replicas = 5`},
		{statement: `ALTER TABLE t ADD COLUMN c2 INT`, want: []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{statement: `ALTER TABLE t RENAME COLUMN c TO c2`, want: []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{statement: `ALTER TABLE t SET LOCALITY REGIONAL BY ROW`, want: []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{statement: `ALTER TABLE t SET SCHEMA app`, want: []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{statement: `ALTER VIEW v OWNER TO u`, want: []storepb.StatementType{storepb.StatementType_ALTER_VIEW}},
		{statement: `ALTER SEQUENCE s OWNER TO u`, want: []storepb.StatementType{storepb.StatementType_ALTER_SEQUENCE}},
		{statement: `ALTER INDEX t@idx NOT VISIBLE`, want: []storepb.StatementType{storepb.StatementType_ALTER_INDEX}},
		{statement: `ALTER SEQUENCE s RESTART WITH 100`, want: []storepb.StatementType{storepb.StatementType_ALTER_SEQUENCE}},
		{statement: `ALTER TYPE e ADD VALUE 'c'`, want: []storepb.StatementType{storepb.StatementType_ALTER_TYPE}},
		{statement: `ALTER TABLE t RENAME TO t2`, want: []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{statement: `ALTER VIEW v RENAME TO v2`, want: []storepb.StatementType{storepb.StatementType_ALTER_VIEW}},
		{statement: `ALTER MATERIALIZED VIEW mv RENAME TO mv2`, want: []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{statement: `ALTER SEQUENCE s RENAME TO s2`, want: []storepb.StatementType{storepb.StatementType_RENAME_SEQUENCE}},
		{statement: `ALTER INDEX t@idx RENAME TO idx2`, want: []storepb.StatementType{storepb.StatementType_RENAME_INDEX}},
		{statement: `ALTER SCHEMA app RENAME TO app2`, want: []storepb.StatementType{storepb.StatementType_RENAME_SCHEMA}},
		{statement: `COMMENT ON TABLE t IS 'comment'`, want: []storepb.StatementType{storepb.StatementType_COMMENT}},
		{statement: `COMMENT ON COLUMN t.c IS 'comment'`, want: []storepb.StatementType{storepb.StatementType_COMMENT}},
		{statement: `INSERT INTO t VALUES (1)`, want: []storepb.StatementType{storepb.StatementType_INSERT}},
		{statement: `UPSERT INTO t VALUES (1)`, want: []storepb.StatementType{storepb.StatementType_INSERT}},
		{statement: `UPDATE t SET c = 1`, want: []storepb.StatementType{storepb.StatementType_UPDATE}},
		{statement: `DELETE FROM t WHERE id = 1`, want: []storepb.StatementType{storepb.StatementType_DELETE}},
		{
			statement: `SET application_name = 'x'; CREATE TABLE t (id INT PRIMARY KEY); SELECT 1; INSERT INTO t VALUES (1)`,
			want:      []storepb.StatementType{storepb.StatementType_CREATE_TABLE, storepb.StatementType_INSERT},
		},
		{statement: `WITH d AS (DELETE FROM t RETURNING id) SELECT count(*) FROM d`, want: []storepb.StatementType{storepb.StatementType_DELETE}},
		{statement: `SELECT * FROM [UPDATE t SET c = 1 RETURNING id]`, want: []storepb.StatementType{storepb.StatementType_UPDATE}},
		{
			statement: `WITH d AS (DELETE FROM t RETURNING *) INSERT INTO t2 SELECT * FROM d`,
			want:      []storepb.StatementType{storepb.StatementType_INSERT, storepb.StatementType_DELETE},
		},
		{
			statement: `CREATE TABLE t_moved AS SELECT * FROM [DELETE FROM t RETURNING *]`,
			want:      []storepb.StatementType{storepb.StatementType_CREATE_TABLE, storepb.StatementType_DELETE},
		},
		{statement: `EXPLAIN ANALYZE DELETE FROM t WHERE id = 1`, want: []storepb.StatementType{storepb.StatementType_DELETE}},
		{statement: `EXPLAIN ANALYZE (VERBOSE) UPSERT INTO t VALUES (1)`, want: []storepb.StatementType{storepb.StatementType_INSERT}},
		{statement: `EXPLAIN DELETE FROM t WHERE id = 1`},
		{statement: `SELECT * FROM t`},
	} {
		t.Run(tc.statement, func(t *testing.T) {
			got, err := GetStatementTypes(parseASTs(t, tc.statement))
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
