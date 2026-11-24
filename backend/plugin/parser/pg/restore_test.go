package pg

import (
	"context"
	"math"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestRestoreSeam(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name:    "t",
				Columns: []*metadatapb.ColumnMetadata{{Name: "id"}, {Name: "a"}},
				Indexes: []*metadatapb.IndexMetadata{{Name: "t_pk", Expressions: []string{"id"}, Primary: true, Unique: true}},
			}},
		}},
	}
	rCtx := base.RestoreContext{
		InstanceID:              "i1",
		GetDatabaseMetadataFunc: buildSeamMetadataGetter(meta),
	}

	const script = "SET ROLE admin;\nDELETE FROM t WHERE id = 1;\nUPDATE t SET a = 2 WHERE id = 2;"
	for _, prefix := range []string{"instances/i1/databases/", "projects/p1/instances/i1/databases/"} {
		sql, err := base.GenerateRestoreSQL(context.Background(), storepb.Engine_POSTGRES, rCtx, script, &storepb.PriorBackupDetail_Item{
			SourceTable:   &storepb.PriorBackupDetail_Item_Table{Database: prefix + "db", Schema: "", Table: "t"},
			TargetTable:   &storepb.PriorBackupDetail_Item_Table{Database: prefix + "bb", Schema: "bb", Table: "b_t"},
			StartPosition: &storepb.Position{Line: 3, Column: 0},
			EndPosition:   &storepb.Position{Line: 3, Column: math.MaxInt32},
		})
		require.NoError(t, err)
		require.Contains(t, sql, "SET ROLE admin;")
		require.Contains(t, sql, "UPDATE t SET a = 2 WHERE id = 2")
		require.NotContains(t, sql, "DELETE FROM t")
		require.Contains(t, sql, `ON CONFLICT ON CONSTRAINT "t_pk" DO UPDATE SET "a" = EXCLUDED."a";`)
	}

	_, err := GenerateRestoreSQL(context.Background(), rCtx, script, nil)
	require.ErrorContains(t, err, "backup item is nil")

	_, err = GenerateRestoreSQL(context.Background(), base.RestoreContext{}, "DELETE FROM t;", &storepb.PriorBackupDetail_Item{
		SourceTable:   &storepb.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/db", Table: "t"},
		TargetTable:   &storepb.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/bb", Schema: "bb", Table: "b_t"},
		StartPosition: &storepb.Position{Line: 1, Column: 0},
		EndPosition:   &storepb.Position{Line: 1, Column: math.MaxInt32},
	})
	require.ErrorContains(t, err, "GetDatabaseMetadataFunc is required")
}

func TestExtractPrependStatement(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      string
		wantErr   bool
	}{
		{
			name:      "set role",
			statement: "SET role = 'admin';",
			want:      "SET role = 'admin';",
		},
		{
			name:      "set role without trailing semicolon",
			statement: "  SET role = 'admin'  ",
			want:      "SET role = 'admin';",
		},
		{
			name:      "set search_path mixed case",
			statement: "Set Search_Path = schema1, public;",
			want:      "Set Search_Path = schema1, public;",
		},
		{
			name:      "first matching statement wins",
			statement: "SET work_mem = '64MB'; SET search_path = app; SET role = 'admin';",
			want:      "SET search_path = app;",
		},
		{
			name:      "preserves internal whitespace and trailing comment",
			statement: "  SET\trole = \n'admin'  ; -- comment",
			want:      "SET\trole = \n'admin'  ;",
		},
		{
			name:      "non-matching statements return empty",
			statement: "SET work_mem = '64MB'; CREATE TABLE role (id int); SELECT 1;",
			want:      "",
		},
		{
			name:      "empty and whitespace",
			statement: "   \n\t  ",
			want:      "",
		},
		{
			name:      "invalid syntax returns error",
			statement: "SET role = ;",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractPrependStatement(tt.statement)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
