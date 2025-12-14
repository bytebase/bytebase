package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	statement :=
		`CREATE TABLE t1 (c1 INT);
						DROP TABLE t1;
						ALTER TABLE t1 ADD COLUMN c1 INT;
						ALTER TABLE t1 RENAME TO t2;
						COMMENT ON TABLE t1 IS 'comment';
						INSERT INTO t1 (c1) VALUES (1), (5);
						UPDATE t1 SET c1 = 5;
			`
	changedResources := model.NewChangedResources(dbMetadata)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		true,
	)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t2",
		},
		false,
	)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		false,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         1,
		SampleDMLS:       []string{"UPDATE t1 SET c1 = 5;"},
		InsertCount:      2,
	}

	asts, err := base.Parse(storepb.Engine_POSTGRES, statement)
	require.NoError(t, err)
	got, err := extractChangedResources("db", "public", dbMetadata, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
