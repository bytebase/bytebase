package mysql

import (
	"archive/zip"
	"bytes"
	"cmp"
	"io"
	"slices"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// concatMultiFile reproduces the declarative rollout concatenation from
// action/command/file.go: lexical path sort joined with "\n".
func concatMultiFile(result *schema.MultiFileSchemaResult) string {
	files := make([]schema.File, len(result.Files))
	copy(files, result.Files)
	slices.SortFunc(files, func(a, b schema.File) int { return cmp.Compare(a.Name, b.Name) })
	var b strings.Builder
	for i, f := range files {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(f.Content)
	}
	return b.String()
}

func TestGetMultiFileDatabaseDefinition(t *testing.T) {
	// Construct metadata directly rather than parsing DDL so generator branches
	// are exercised independently of parser extraction.
	metaProto := &metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{
				Name:      "audit_log",
				Engine:    "InnoDB",
				Charset:   "utf8mb4",
				Collation: "utf8mb4_0900_ai_ci",
				Columns: []*metadatapb.ColumnMetadata{
					{
						Name:     "id",
						Position: 1,
						Type:     "int",
						Default:  "AUTO_INCREMENT",
					},
					{
						Name:         "msg",
						Position:     2,
						Type:         "varchar(200)",
						CharacterSet: "utf8mb4",
						Collation:    "utf8mb4_0900_ai_ci",
					},
					{
						Name:     "created_at",
						Position: 3,
						Type:     "timestamp",
						Default:  "CURRENT_TIMESTAMP",
					},
				},
				Indexes: []*metadatapb.IndexMetadata{{
					Name:        "PRIMARY",
					Expressions: []string{"id"},
					KeyLength:   []int64{-1},
					Descending:  []bool{false},
					Type:        "BTREE",
					Unique:      true,
					Primary:     true,
					Visible:     true,
				}},
				Triggers: []*metadatapb.TriggerMetadata{{
					Name:   "audit_log_bi",
					Timing: "BEFORE",
					Event:  "INSERT",
					Body:   "SET NEW.msg = TRIM(NEW.msg)",
				}},
			}},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "recent_audit",
				Definition: "select `audit_log`.`id` AS `id`,`audit_log`.`msg` AS `msg` from `audit_log`",
			}},
			Functions: []*metadatapb.FunctionMetadata{{
				Name:       "audit_count",
				Definition: "CREATE FUNCTION `audit_count`() RETURNS int\n    DETERMINISTIC\nBEGIN\n\tRETURN 1;\nEND",
			}},
			Procedures: []*metadatapb.ProcedureMetadata{{
				Name:       "touch_audit",
				Definition: "CREATE PROCEDURE `touch_audit`()\nBEGIN\n\tSELECT 1;\nEND",
			}},
			Events: []*metadatapb.EventMetadata{{
				Name:       "purge_audit",
				Definition: "CREATE EVENT `purge_audit` ON SCHEDULE EVERY 1 DAY ON COMPLETION NOT PRESERVE ENABLE DO DELETE FROM audit_log WHERE created_at < (NOW() - INTERVAL 30 DAY)",
				TimeZone:   "SYSTEM",
			}},
		}},
	}

	result, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{
		SkipBackupSchema: true,
	}, metaProto)
	require.NoError(t, err)
	require.Len(t, result.Files, 6)

	byDir := map[string][]schema.File{}
	for _, f := range result.Files {
		require.True(t, strings.HasSuffix(f.Name, ".sql"), "file %q must end with .sql", f.Name)
		require.NotEmpty(t, strings.TrimSpace(f.Content), "file %q must be non-empty", f.Name)
		dir, _, ok := strings.Cut(f.Name, "/")
		require.True(t, ok, "file %q must be under a directory", f.Name)
		byDir[dir] = append(byDir[dir], f)
	}
	require.Len(t, byDir["tables"], 1)
	require.Len(t, byDir["views"], 1)
	require.Len(t, byDir["functions"], 1)
	require.Len(t, byDir["procedures"], 1)
	require.Len(t, byDir["triggers"], 1)
	require.Len(t, byDir["events"], 1)

	// Verify ZIP round-trip matches every emitted file entry-for-entry.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range result.Files {
		w, err := zw.Create(f.Name)
		require.NoError(t, err)
		_, err = w.Write([]byte(f.Content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, zr.File, len(result.Files))

	want := make(map[string]string, len(result.Files))
	for _, f := range result.Files {
		want[f.Name] = f.Content
	}
	for _, entry := range zr.File {
		rc, err := entry.Open()
		require.NoError(t, err)
		got, err := io.ReadAll(rc)
		require.NoError(t, rc.Close())
		require.NoError(t, err)
		require.Equal(t, want[entry.Name], string(got))
	}

	// Lexical concatenation places events/ before tables/; the SDL seam must
	// still accept it idempotently and agree with the single-file SDL dump.
	concat := concatMultiFile(result)
	require.Less(t, strings.Index(concat, "CREATE EVENT"), strings.Index(concat, "CREATE TABLE"))

	single, err := schema.GetDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{SDLFormat: true}, metaProto)
	require.NoError(t, err)

	diffVsSingle, err := mysqlDiffSDLMigration(concat, single, "8.0.46")
	require.NoError(t, err)
	require.Empty(t, diffVsSingle)

	selfDiff, err := mysqlDiffSDLMigration(concat, concat, "8.0.46")
	require.NoError(t, err)
	require.Empty(t, selfDiff)

	plan, err := mysqlDiffSDLMigration("", concat, "8.0.46")
	require.NoError(t, err)
	require.Contains(t, plan, "CREATE TABLE")
	require.Contains(t, plan, "EVENT")
	require.NotContains(t, plan, sdlDatabaseContext)
}
