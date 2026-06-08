package redshift

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	statement := `CREATE TABLE t1 (c1 INT);
ALTER TABLE t1 ADD COLUMN c2 INT;
INSERT INTO t1 VALUES (1);
UPDATE t1 SET c2 = 2;`

	dbMetadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_REDSHIFT, true /* caseSensitive */)
	changedResources := model.NewChangedResources(dbMetadata)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		true,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS: []string{
			"INSERT INTO t1 VALUES (1);",
			"UPDATE t1 SET c2 = 2;",
		},
		DMLCount:    2,
		InsertCount: 1,
	}

	stmts, err := base.ParseStatements(storepb.Engine_REDSHIFT, statement)
	require.NoError(t, err)
	got, err := base.ExtractChangedResources(storepb.Engine_REDSHIFT, "db", "public", dbMetadata, base.ExtractASTs(stmts), statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResourcesSearchPathDropAndSelectInto(t *testing.T) {
	statement := `SET search_path TO analytics;
CREATE TABLE unqualified(id INT);
SELECT * INTO copied_rows FROM public.rows;
DROP TABLE old_rows;
DELETE FROM copied_rows WHERE id = 1;`

	dbMetadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_REDSHIFT, true /* caseSensitive */)
	changedResources := model.NewChangedResources(dbMetadata)
	changedResources.AddTable(
		"db",
		"analytics",
		&storepb.ChangedResourceTable{
			Name: "unqualified",
		},
		false,
	)
	changedResources.AddTable(
		"db",
		"analytics",
		&storepb.ChangedResourceTable{
			Name: "copied_rows",
		},
		true,
	)
	changedResources.AddTable(
		"db",
		"analytics",
		&storepb.ChangedResourceTable{
			Name: "old_rows",
		},
		true,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS: []string{
			"DELETE FROM copied_rows WHERE id = 1;",
		},
		DMLCount:    1,
		InsertCount: 0,
	}

	stmts, err := base.ParseStatements(storepb.Engine_REDSHIFT, statement)
	require.NoError(t, err)
	got, err := base.ExtractChangedResources(storepb.Engine_REDSHIFT, "db", "public", dbMetadata, base.ExtractASTs(stmts), statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
