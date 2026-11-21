package mysql

import (
	"archive/zip"
	"bytes"
	"cmp"
	"io"
	"slices"
	"strings"
	"testing"

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
	const ddl = `
CREATE TABLE audit_log (
	id INT PRIMARY KEY AUTO_INCREMENT,
	msg VARCHAR(200) NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW recent_audit AS SELECT id, msg FROM audit_log;

CREATE FUNCTION audit_count() RETURNS INT DETERMINISTIC
BEGIN
	RETURN 1;
END;

CREATE PROCEDURE touch_audit()
BEGIN
	SELECT 1;
END;

CREATE TRIGGER audit_log_bi BEFORE INSERT ON audit_log FOR EACH ROW
SET NEW.msg = TRIM(NEW.msg);

CREATE EVENT purge_audit
	ON SCHEDULE EVERY 1 DAY
	DO DELETE FROM audit_log WHERE created_at < (NOW() - INTERVAL 30 DAY);
`

	metaProto, err := schema.GetDatabaseMetadata(storepb.Engine_MYSQL, ddl)
	require.NoError(t, err)

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
