package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
)

func TestExtractChangedResources(t *testing.T) {
	tests := []struct {
		statement string
		want      *base.ChangeSummary
	}{
		{
			statement: `CREATE TABLE t1 (c1 INT);
						DROP TABLE t1;
						ALTER TABLE t1 ADD COLUMN c1 INT;
						ALTER TABLE t1 RENAME TO t2;
						INSERT INTO t1 (c1) VALUES (1), (5);
						UPDATE t1 SET c1 = 5;
			`,
			want: &base.ChangeSummary{
				ResourceChanges: []*base.ResourceChange{
					{
						Resource: base.SchemaResource{
							Database: "db",
							Schema:   "public",
							Table:    "t1",
						},
						Ranges: []base.Range{
							{
								Start: 0,
								End:   25,
							},
							{
								Start: 32,
								End:   46,
							},
							{
								Start: 53,
								End:   86,
							},
							{
								Start: 93,
								End:   121,
							},
						},
						AffectTable: true,
					},
					{
						Resource: base.SchemaResource{
							Database: "db",
							Schema:   "public",
							Table:    "t2",
						},
						Ranges: []base.Range{
							{
								Start: 93,
								End:   121,
							},
						},
					},
				},
				DMLCount:    1,
				SampleDMLS:  []string{"UPDATE t1 SET c1 = 5;"},
				InsertCount: 2,
			},
		},
		{
			statement: `CREATE TABLE t1(a int);`,
			want: &base.ChangeSummary{
				ResourceChanges: []*base.ResourceChange{
					{
						Resource: base.SchemaResource{
							Database: "db",
							Schema:   "public",
							Table:    "t1",
						},
						Ranges: []base.Range{
							{
								Start: 0,
								End:   23,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		nodes, err := pgrawparser.Parse(pgrawparser.ParseContext{}, test.statement)
		require.NoError(t, err)
		got, err := extractChangedResources("db", "public", nodes, test.statement)
		require.NoError(t, err)
		require.Equal(t, test.want, got)
	}
}
