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
				Ranges: []base.Range{
					{
						Start: 0,
						End:   25,
					},
					{
						Start: 27,
						End:   41,
					},
					{
						Start: 43,
						End:   76,
					},
					{
						Start: 78,
						End:   100,
					},
				},
			},
			{
				Resource: base.SchemaResource{
					Database: "db",
					Table:    "t2",
				},
				Ranges: []base.Range{
					{
						Start: 78,
						End:   100,
					},
				},
			},
		},
		SampleDMLS: []string{
			"INSERT INTO t1 (c1) VALUES (1);",
		},
		DMLCount: 1,
	}

	asts, _ := ParseMySQL(statement)
	got, err := extractChangedResources("db", "", asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
