package base

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestDatabaseQualifiersInStatement(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		defaultDB string
		want      []string
	}{
		{
			name:      "table alias column qualifier ignored",
			statement: "SELECT a. FROM users a",
			defaultDB: "appdb",
		},
		{
			name:      "join alias qualifiers in ON ignored",
			statement: "SELECT a2. FROM t1 JOIN t2 a2 ON t1.c1 = a2.c1",
			defaultDB: "appdb",
		},
		{
			name:      "update set and where table qualifiers ignored",
			statement: "UPDATE users u SET u.name = 'x' WHERE u.id = 1",
			defaultDB: "appdb",
		},
		{
			name:      "extract from inside scalar function ignored",
			statement: "SELECT EXTRACT(YEAR FROM u.created_at) FROM users u",
			defaultDB: "appdb",
		},
		{
			name:      "cross-db table and three-part column extracts db only",
			statement: "SELECT otherdb.t2. FROM otherdb.t2",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "cross-db in FROM with alias qualifier in SELECT extracts db only",
			statement: "SELECT u. FROM otherdb.users u",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "comma-joined cross-db table",
			statement: "SELECT a.id FROM users a, otherdb.orders o WHERE a.id = o.user_id",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "derived table subquery cross-db reference",
			statement: "SELECT sub.c1 FROM (SELECT a.c1 FROM otherdb.t2 a) sub WHERE sub.c1 = 1",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "default database excluded case-insensitively",
			statement: "SELECT * FROM APPDB.users u WHERE u.id = 1",
			defaultDB: "appdb",
		},
		{
			name:      "preserves backtick-quoted database casing",
			statement: "SELECT * FROM `OtherDB`.`users` u WHERE u.id = 1",
			defaultDB: "appdb",
			want:      []string{"OtherDB"},
		},
		{
			name:      "table qualifier in insert column list ignored while target db extracted",
			statement: "INSERT INTO otherdb.t2 (t2.c1) VALUES (1)",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "ddl create trigger on cross-db table",
			statement: "CREATE TRIGGER tr BEFORE INSERT ON otherdb.t2 FOR EACH ROW SET NEW.c1 = 1",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "join with incomplete qualifier in FROM and ON",
			statement: "SELECT a. FROM users a JOIN otherdb.orders o ON a. = o.user_id WHERE o.",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "desc in order by ignored",
			statement: "SELECT * FROM users u ORDER BY u.id DESC",
			defaultDB: "appdb",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := databaseQualifiersInStatement(tc.statement, tc.defaultDB)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestPopulateMySQLFamilyCatalog_SkipsEnumerationForQualifiers(t *testing.T) {
	listCalls := 0
	var loaded []string
	var registered []string

	cCtx := CompletionContext{
		DefaultDatabase: "appdb",
		ListDatabaseNames: func(_ context.Context, _ string) ([]string, error) {
			listCalls++
			return []string{"appdb", "otherdb", "unuseddb"}, nil
		},
		Metadata: func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
			loaded = append(loaded, db)
			return db, model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: db}, nil, nil, storepb.Engine_MYSQL, true), nil
		},
	}

	// Alias qualifier does not call ListDatabaseNames or load 'a'.
	PopulateMySQLFamilyCatalog(
		context.Background(),
		cCtx,
		"SELECT a. FROM users a",
		len("SELECT a."),
		func(string, int) bool { return false },
		func(name string) { registered = append(registered, name) },
		func(string, *metadatapb.DatabaseSchemaMetadata) {},
		func(string) {},
	)
	require.Zero(t, listCalls)
	require.Empty(t, registered)
	require.Equal(t, []string{"appdb"}, loaded)

	// Cross-db qualifier resolves directly without ListDatabaseNames.
	loaded = nil
	PopulateMySQLFamilyCatalog(
		context.Background(),
		cCtx,
		"SELECT otherdb.t2. FROM otherdb.t2",
		len("SELECT otherdb.t2."),
		func(string, int) bool { return false },
		func(name string) { registered = append(registered, name) },
		func(string, *metadatapb.DatabaseSchemaMetadata) {},
		func(string) {},
	)
	require.Zero(t, listCalls)
	require.Empty(t, registered)
	require.Equal(t, []string{"appdb", "otherdb"}, loaded)

	// Cursor expecting database_ref enumerates via ListDatabaseNames and registers unreferenced DBs only.
	loaded = nil
	PopulateMySQLFamilyCatalog(
		context.Background(),
		cCtx,
		"SELECT * FROM ",
		len("SELECT * FROM "),
		func(string, int) bool { return true },
		func(name string) { registered = append(registered, name) },
		func(string, *metadatapb.DatabaseSchemaMetadata) {},
		func(string) {},
	)
	require.Equal(t, 1, listCalls)
	require.Equal(t, []string{"otherdb", "unuseddb"}, registered)
	require.Equal(t, []string{"appdb"}, loaded)
}
