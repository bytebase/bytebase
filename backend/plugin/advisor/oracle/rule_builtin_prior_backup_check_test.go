package oracle

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

// TestOraclePriorBackupCheckAdapter covers Bytebase's BUILTIN_PRIOR_BACKUP_CHECK
// adapter rendering and case-sensitivity wiring for Oracle. Exhaustive LONG/LONG RAW
// and mixed-DML rule behavior lives in omni/oracle/v1review.
func TestOraclePriorBackupCheckAdapter(t *testing.T) {
	dbSchema := &metadatapb.DatabaseSchemaMetadata{
		Name: "DB",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "",
			Tables: []*metadatapb.TableMetadata{
				{Name: "T_LONG", Columns: []*metadatapb.ColumnMetadata{
					{Name: "ID", Type: "NUMBER"},
					{Name: "PAYLOAD", Type: "LONG"},
				}},
				{Name: "t_long", Columns: []*metadatapb.ColumnMetadata{
					{Name: "ID", Type: "NUMBER"},
					{Name: "NOTE", Type: "VARCHAR2"},
				}},
			},
		}},
	}

	run := func(t *testing.T, statement string, backupDBPresent bool) []*storepb.Advice {
		t.Helper()
		parsed, err := base.ParseStatements(storepb.Engine_ORACLE, statement)
		require.NoError(t, err)
		advices, err := advisor.Check(context.Background(), storepb.Engine_ORACLE, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, advisor.Context{
			DBType:                storepb.Engine_ORACLE,
			DBSchema:              dbSchema,
			EnablePriorBackup:     true,
			IsObjectCaseSensitive: true,
			Rule:                  &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, Level: storepb.SQLReviewRule_WARNING},
			ParsedStatements:      parsed,
			ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
				if backupDBPresent {
					return []string{"BBDATAARCHIVE"}, nil
				}
				return nil, nil
			},
		})
		require.NoError(t, err)
		return advices
	}

	t.Run("long column warning rendering and case-sensitive schema wiring", func(t *testing.T) {
		advices := run(t, "UPDATE t_long SET ID = 1 WHERE ID = 2;", true)
		require.Len(t, advices, 1)
		require.Equal(t, code.BuiltinPriorBackupCheck.Int32(), advices[0].Code)
		require.Equal(t, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK.String(), advices[0].Title)
		require.Contains(t, advices[0].Content, `"DB"."T_LONG"`)
		require.Contains(t, advices[0].Content, `"PAYLOAD"`)

		require.Empty(t, run(t, `UPDATE "t_long" SET ID = 1 WHERE ID = 2;`, true))
	})

	t.Run("mixed DDL and DML rendering", func(t *testing.T) {
		advices := run(t, "CREATE TABLE U (ID NUMBER);\nUPDATE T_LONG SET ID = 1 WHERE ID = 2;", true)
		require.Len(t, advices, 2)
		require.Equal(t, code.BuiltinPriorBackupCheck.Int32(), advices[0].Code)
		require.Contains(t, advices[0].Content, "mixed DDL and DML")
	})

	t.Run("missing backup database rendering", func(t *testing.T) {
		advices := run(t, "UPDATE T_LONG SET ID = 1 WHERE ID = 2;", false)
		require.Len(t, advices, 1)
		require.Equal(t, code.DatabaseNotExists.Int32(), advices[0].Code)
		require.Contains(t, advices[0].Content, `"BBDATAARCHIVE"`)
	})
}
