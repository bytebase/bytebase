package oracle

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

// TestPriorBackupLongColumnWarning pins the LONG/LONG RAW pre-flight warning:
// CREATE TABLE AS SELECT cannot copy those columns (ORA-00997), so prior
// backup fails at run time with no earlier signal unless the check warns.
func TestPriorBackupLongColumnWarning(t *testing.T) {
	// The real Oracle sync stores the connection schema's tables under an
	// EMPTY schema name (db/oracle/sync.go) — the fixture must match that
	// shape or the schema comparison is never exercised realistically.
	dbSchema := &storepb.DatabaseSchemaMetadata{
		Name: "DB",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{Name: "T_LONG", Columns: []*storepb.ColumnMetadata{
						{Name: "ID", Type: "NUMBER"},
						{Name: "PAYLOAD", Type: "LONG"},
					}},
					{Name: "T_LONG_RAW", Columns: []*storepb.ColumnMetadata{
						{Name: "ID", Type: "NUMBER"},
						{Name: "BLOB_ISH", Type: "LONG RAW"},
					}},
					{Name: "T_CLOB", Columns: []*storepb.ColumnMetadata{
						{Name: "ID", Type: "NUMBER"},
						{Name: "DOC", Type: "CLOB"},
					}},
				},
			},
		},
	}

	check := func(t *testing.T, statement string) []*storepb.Advice {
		t.Helper()
		parsed, err := base.ParseStatements(storepb.Engine_ORACLE, statement)
		require.NoError(t, err)
		checkCtx := advisor.Context{
			DBType:            storepb.Engine_ORACLE,
			DBSchema:          dbSchema,
			EnablePriorBackup: true,
			Rule:              &storepb.SQLReviewRule{Type: storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, Level: storepb.SQLReviewRule_WARNING},
			ParsedStatements:  parsed,
			ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
				return []string{"BBDATAARCHIVE"}, nil
			},
		}
		a := &StatementPriorBackupCheckAdvisor{}
		advices, err := a.Check(context.Background(), checkCtx)
		require.NoError(t, err)
		return advices
	}

	longAdvices := func(advices []*storepb.Advice) []*storepb.Advice {
		var result []*storepb.Advice
		for _, a := range advices {
			if strings.Contains(a.Content, "LONG") {
				result = append(result, a)
			}
		}
		return result
	}

	t.Run("long_column_warns_with_table_and_column", func(t *testing.T) {
		advices := longAdvices(check(t, "UPDATE T_LONG SET ID = 1 WHERE ID = 2;"))
		require.Len(t, advices, 1)
		require.Contains(t, advices[0].Content, `"T_LONG"`)
		require.Contains(t, advices[0].Content, `"PAYLOAD"`)
	})

	t.Run("long_raw_column_warns", func(t *testing.T) {
		advices := longAdvices(check(t, "DELETE FROM T_LONG_RAW WHERE ID = 2;"))
		require.Len(t, advices, 1)
		require.Contains(t, advices[0].Content, `"BLOB_ISH"`)
	})

	t.Run("clob_does_not_warn", func(t *testing.T) {
		// CLOB backups work via ROWID deduplication — no warning.
		require.Empty(t, longAdvices(check(t, "UPDATE T_CLOB SET ID = 1 WHERE ID = 2;")))
	})

	t.Run("one_warning_per_table_not_per_statement", func(t *testing.T) {
		advices := longAdvices(check(t,
			"UPDATE T_LONG SET ID = 1 WHERE ID = 1;\nUPDATE T_LONG SET ID = 2 WHERE ID = 2;"))
		require.Len(t, advices, 1)
	})

	t.Run("schema_qualified_dml_warns_too", func(t *testing.T) {
		// Schema-qualified DML normalizes to an explicit schema name; the
		// empty synced schema (= current schema) must still match.
		advices := longAdvices(check(t, "UPDATE DB.T_LONG SET ID = 1 WHERE ID = 2;"))
		require.Len(t, advices, 1)
		require.Contains(t, advices[0].Content, `"PAYLOAD"`)
	})

	t.Run("cross_owner_dml_stays_silent", func(t *testing.T) {
		// The empty synced schema holds only the CONNECTED schema's tables;
		// a different explicit owner must not borrow its metadata even when
		// a same-named table exists.
		require.Empty(t, longAdvices(check(t, "UPDATE OTHER_OWNER.T_LONG SET ID = 1 WHERE ID = 2;")))
	})

	t.Run("unknown_table_stays_silent", func(t *testing.T) {
		// Missing metadata must not produce false alarms.
		require.Empty(t, longAdvices(check(t, "UPDATE NO_SUCH_TABLE SET ID = 1 WHERE ID = 2;")))
	})
}
