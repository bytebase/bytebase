package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExtractMySQLChangedResources(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
	DROP TABLE t1;
	ALTER TABLE t1 ADD COLUMN c1 INT;
	RENAME TABLE t1 TO t2;
	INSERT INTO t1 (c1) VALUES (1);
	`
	want := &base.ChangeSummary{
		ResourceChanges: []*base.ResourceChange{
			{
				Resource: base.SchemaResource{
					Database: "db",
					Table:    "t1",
				},
				AffectTable: true,
			},
			{
				Resource: base.SchemaResource{
					Database: "db",
					Table:    "t2",
				},
			},
		},
		SampleDMLS: []string{
			"INSERT INTO t1 (c1) VALUES (1);",
		},
		DMLCount: 1,
	}

	asts, _ := ParseMySQL(statement)
	got, err := extractChangedResources("db", "", asts)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
