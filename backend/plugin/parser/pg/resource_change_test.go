package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	dbSchema := model.NewDatabaseSchema(&storepb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	statement :=
		`CREATE TABLE t1 (c1 INT);
						DROP TABLE t1;
						ALTER TABLE t1 ADD COLUMN c1 INT;
						ALTER TABLE t1 RENAME TO t2;
						COMMENT ON TABLE t1 IS 'comment';
						INSERT INTO t1 (c1) VALUES (1), (5);
						UPDATE t1 SET c1 = 5;
			`
	changedResources := model.NewChangedResources(dbSchema)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t1",
			Ranges: []*storepb.Range{
				{Start: 0, End: 25},
				{Start: 32, End: 46},
				{Start: 53, End: 86},
				{Start: 93, End: 121},
			},
		},
		true,
	)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name:   "t2",
			Ranges: []*storepb.Range{{Start: 93, End: 121}},
		},
		false,
	)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t1",
			Ranges: []*storepb.Range{
				{Start: 168, End: 204},
				{Start: 211, End: 232},
			},
		},
		false,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         1,
		SampleDMLS:       []string{"UPDATE t1 SET c1 = 5;"},
		InsertCount:      2,
	}

	nodes, err := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
	require.NoError(t, err)
	got, err := extractChangedResources("db", "public", dbSchema, nodes, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
