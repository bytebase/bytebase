package base

import (
	"context"
	"strings"
	"testing"

	omnianalysis "github.com/bytebase/omni/analysis"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestConvertOffsetBackupStatements(t *testing.T) {
	type offsetBackup struct {
		stmt, schema, src, bak string
		start, end             int
	}
	extract := func(b offsetBackup) (string, string, string, string, int, int) {
		return b.stmt, b.schema, b.src, b.bak, b.start, b.end
	}

	require.Nil(t, ConvertOffsetBackupStatements("UPDATE t SET a = 1;", []offsetBackup(nil), extract, false))

	sql := "UPDATE t1 SET a = 1;\nDELETE FROM t2;"
	// Pass items out of offset order to exercise single-pass ascending offset mapping.
	items := []offsetBackup{
		{stmt: "SELECT * FROM t2", schema: "public", src: "t2", bak: "bak_t2", start: 21, end: len(sql)},
		{stmt: "SELECT * FROM t1", schema: "public", src: "t1", bak: "bak_t1", start: 0, end: 20},
	}

	got1BasedEnd := ConvertOffsetBackupStatements(sql, items, extract, false)
	require.Len(t, got1BasedEnd, 2)
	require.Equal(t, &storepb.Position{Line: 2, Column: 0}, got1BasedEnd[0].StartPosition)
	require.Equal(t, &storepb.Position{Line: 2, Column: 16}, got1BasedEnd[0].EndPosition)
	require.Equal(t, &storepb.Position{Line: 1, Column: 0}, got1BasedEnd[1].StartPosition)
	require.Equal(t, &storepb.Position{Line: 1, Column: 21}, got1BasedEnd[1].EndPosition)

	got0BasedEnd := ConvertOffsetBackupStatements(sql, items, extract, true)
	require.Equal(t, &storepb.Position{Line: 2, Column: 15}, got0BasedEnd[0].EndPosition)
	require.Equal(t, &storepb.Position{Line: 1, Column: 20}, got0BasedEnd[1].EndPosition)
}

func TestBackupCatalogFuncsAndColumnClassifier(t *testing.T) {
	dbMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db1",
		Schemas: []*metadatapb.SchemaMetadata{{Name: ""}},
	}, nil, nil, storepb.Engine_MYSQL, false)
	wrap := func(d *UnnamedSchemaDatabase) *UnnamedSchemaDatabase { return d }

	t.Run("nil callbacks return nil closures", func(t *testing.T) {
		list, get := BackupCatalogFuncs(context.Background(), "i1", nil, nil, wrap)
		require.Nil(t, list)
		require.Nil(t, get)
	})

	t.Run("lookup returns wrapped metadata, nil, or raw error", func(t *testing.T) {
		wantErr := errors.New("boom")
		getDB := func(_ context.Context, instanceID, name string) (string, *model.DatabaseMetadata, error) {
			require.Equal(t, "i1", instanceID)
			switch name {
			case "db1":
				return name, dbMeta, nil
			case "missing":
				return name, nil, nil
			default:
				return "", nil, wantErr
			}
		}
		listDBs := func(_ context.Context, instanceID string) ([]string, error) {
			require.Equal(t, "i1", instanceID)
			return []string{"db1"}, nil
		}

		list, get := BackupCatalogFuncs(context.Background(), "i1", getDB, listDBs, wrap)
		names, err := list()
		require.NoError(t, err)
		require.Equal(t, []string{"db1"}, names)

		got, err := get("db1")
		require.NoError(t, err)
		require.Equal(t, "db1", got.Name())

		got, err = get("missing")
		require.NoError(t, err)
		require.Nil(t, got)

		_, err = get("err")
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("NewColumnClassifier and ClassifyRestoreColumns", func(t *testing.T) {
		classify := func(_ struct{}, _ bool, db, _ string) (string, error) {
			if db == "missing" {
				return "", &omnianalysis.ResourceNotFoundError{}
			}
			return "ok", nil
		}

		_, err := NewColumnClassifier(nil, "i1", struct{}{}, false, classify)("db1", "t1")
		require.EqualError(t, err, "GetDatabaseMetadataFunc is not set")

		dummyGet := func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, nil
		}
		_, err = NewColumnClassifier(dummyGet, "i1", struct{}{}, false, classify)("missing", "t1")
		require.EqualError(t, err, `failed to get database metadata for InstanceID "i1", Database "missing"`)

		val, err := NewColumnClassifier(dummyGet, "i1", struct{}{}, false, classify)("db1", "t1")
		require.NoError(t, err)
		require.Equal(t, "ok", val)

		_, err = ClassifyRestoreColumns(
			RestoreContext{InstanceID: "i1", GetDatabaseMetadataFunc: dummyGet},
			struct{}{},
			&storepb.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/missing", Table: "t1"},
			"missing",
			classify,
		)
		require.EqualError(t, err, `failed to classify columns for instances/i1/databases/missing.t1: failed to get database metadata for InstanceID "i1", Database "missing"`)
	})
}

func TestRestoreHelpers(t *testing.T) {
	t.Run("ValidateBackupItem", func(t *testing.T) {
		require.EqualError(t, ValidateBackupItem(nil), "backup item is nil")
		require.EqualError(t, ValidateBackupItem(&storepb.PriorBackupDetail_Item{}), "backup item source table is nil")
		require.EqualError(t, ValidateBackupItem(&storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{},
		}), "backup item target table is nil")
		require.NoError(t, ValidateBackupItem(&storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{},
			TargetTable: &storepb.PriorBackupDetail_Item_Table{},
		}))
	})

	t.Run("ParseBackupItemDatabases", func(t *testing.T) {
		item := &storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/src"},
			TargetTable: &storepb.PriorBackupDetail_Item_Table{Database: "projects/p1/instances/i1/databases/bak"},
		}
		src, dst, err := ParseBackupItemDatabases(item)
		require.NoError(t, err)
		require.Equal(t, "src", src)
		require.Equal(t, "bak", dst)

		badSrc := &storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{Database: "bad"},
			TargetTable: item.TargetTable,
		}
		_, _, err = ParseBackupItemDatabases(badSrc)
		require.ErrorContains(t, err, "failed to get source database ID for bad")

		badDst := &storepb.PriorBackupDetail_Item{
			SourceTable: item.SourceTable,
			TargetTable: &storepb.PriorBackupDetail_Item_Table{Database: "bad"},
		}
		_, _, err = ParseBackupItemDatabases(badDst)
		require.ErrorContains(t, err, "failed to get target database ID for bad")
	})

	t.Run("FormatRestoreComment", func(t *testing.T) {
		short := "UPDATE t SET a = 1;"
		require.Equal(t, short, FormatRestoreComment(short))

		long := strings.Repeat("a", maxRestoreCommentLength+10)
		got := FormatRestoreComment(long)
		require.Equal(t, strings.Repeat("a", maxRestoreCommentLength)+"...", got)
	})

	t.Run("positionInRange and FilterDMLsInRange", func(t *testing.T) {
		item := &storepb.PriorBackupDetail_Item{
			StartPosition: &storepb.Position{Line: 2, Column: 0},
			EndPosition:   &storepb.Position{Line: 3, Column: 10},
		}
		type span struct {
			id         string
			start, end *storepb.Position
		}
		spans := []span{
			{id: "before", start: &storepb.Position{Line: 1, Column: 0}, end: &storepb.Position{Line: 1, Column: 5}},
			{id: "inside", start: &storepb.Position{Line: 2, Column: 0}, end: &storepb.Position{Line: 3, Column: 10}},
			{id: "overflow", start: &storepb.Position{Line: 2, Column: 0}, end: &storepb.Position{Line: 3, Column: 11}},
		}
		got := FilterDMLsInRange(spans, item, func(s span) (*storepb.Position, *storepb.Position) {
			return s.start, s.end
		})
		require.Len(t, got, 1)
		require.Equal(t, "inside", got[0].id)
	})

	t.Run("ExtractSingleStatement", func(t *testing.T) {
		_, err := ExtractSingleStatement("UPDATE t SET a = 1;", nil, ExtractStatementOptions{})
		require.EqualError(t, err, "failed to extract single SQL: backup item is nil")

		item := &storepb.PriorBackupDetail_Item{
			SourceTable:   &storepb.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/db", Table: "t"},
			StartPosition: &storepb.Position{Line: 1, Column: 0},
			EndPosition:   &storepb.Position{Line: 2, Column: 20},
		}
		got, err := ExtractSingleStatement("ignored", item, ExtractStatementOptions{
			SplitSQL: func(string) ([]Statement, error) {
				return []Statement{
					{Text: "UPDATE t SET a = 1;", Start: &storepb.Position{Line: 1, Column: 0}},
					{Text: "UPDATE other SET a = 1;", Start: &storepb.Position{Line: 2, Column: 0}},
					{Text: "DELETE FROM t;", Start: &storepb.Position{Line: 2, Column: 10}},
				}, nil
			},
			Match: func(stmt, db, tbl string) (bool, error) {
				return db == "db" && strings.Contains(stmt, " "+tbl), nil
			},
			Normalize:     func(s string) string { return strings.TrimSuffix(s, ";") },
			JoinSeparator: "\n",
		})
		require.NoError(t, err)
		require.Equal(t, "UPDATE t SET a = 1\nDELETE FROM t", got)
	})

	t.Run("ResolveRestoreTableMetadata", func(t *testing.T) {
		dbMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: "db",
			Schemas: []*metadatapb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*metadatapb.TableMetadata{
						{
							Name: "t",
							Indexes: []*metadatapb.IndexMetadata{
								{Name: "t_pk", Primary: true, Expressions: []string{"id"}},
							},
						},
					},
				},
				{
					Name: "",
					Tables: []*metadatapb.TableMetadata{
						{Name: "T"},
					},
				},
			},
		}, nil, nil, storepb.Engine_POSTGRES, true)

		rCtx := RestoreContext{
			InstanceID: "i1",
			GetDatabaseMetadataFunc: func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
				if db == "unsynced" {
					return db, nil, nil
				}
				return db, dbMeta, nil
			},
		}

		// Default schema fallback ("public") and PK extraction.
		meta, err := ResolveRestoreTableMetadata(context.Background(), rCtx, "db", &storepb.PriorBackupDetail_Item_Table{Table: "t"}, "public", nil)
		require.NoError(t, err)
		require.Equal(t, "public", meta.Schema)
		require.Equal(t, "t", meta.Table.GetName())
		require.NotNil(t, meta.PrimaryKey)
		require.Equal(t, "t_pk", meta.PrimaryKey.GetName())

		// Unnamed schema mode (Oracle).
		meta, err = ResolveRestoreTableMetadata(context.Background(), rCtx, "db", &storepb.PriorBackupDetail_Item_Table{Table: "T"}, "", nil)
		require.NoError(t, err)
		require.Empty(t, meta.Schema)
		require.Nil(t, meta.PrimaryKey)

		// Default unsynced error.
		_, err = ResolveRestoreTableMetadata(context.Background(), rCtx, "unsynced", &storepb.PriorBackupDetail_Item_Table{Table: "T"}, "", nil)
		require.EqualError(t, err, `database metadata for database "unsynced" not found (database not synced)`)

		// Custom unsynced error.
		_, err = ResolveRestoreTableMetadata(context.Background(), rCtx, "unsynced", &storepb.PriorBackupDetail_Item_Table{Table: "t"}, "dbo", func(string) error {
			return errors.New("database metadata is nil")
		})
		require.EqualError(t, err, "database metadata is nil")

		// Missing schema / table error wording.
		_, err = ResolveRestoreTableMetadata(context.Background(), rCtx, "db", &storepb.PriorBackupDetail_Item_Table{Schema: "missing", Table: "t"}, "public", nil)
		require.EqualError(t, err, "schema metadata not found for missing")
		_, err = ResolveRestoreTableMetadata(context.Background(), rCtx, "db", &storepb.PriorBackupDetail_Item_Table{Table: "missing"}, "public", nil)
		require.EqualError(t, err, "table metadata not found for public.missing")
		_, err = ResolveRestoreTableMetadata(context.Background(), rCtx, "db", &storepb.PriorBackupDetail_Item_Table{Table: "missing"}, "", nil)
		require.EqualError(t, err, "no table metadata for db.missing")
	})
}
