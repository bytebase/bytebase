package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestExtractChangedResources(t *testing.T) {
	dbSchema := model.NewDBSchema(&storepb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{})
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
