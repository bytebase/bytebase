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
			name:      "insert with priority/ignore modifiers keeps target state before INTO",
			statement: "INSERT LOW_PRIORITY IGNORE INTO otherdb.",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "replace with delayed modifier keeps target state before INTO",
			statement: "REPLACE DELAYED INTO otherdb.",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "drop view preserves object list across comma",
			statement: "DROP VIEW appdb.v1, otherdb.",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "drop temporary table preserves object list across comma",
			statement: "DROP TEMPORARY TABLE IF EXISTS appdb.t1, otherdb.",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "analyze table maintenance statement preserves list across comma and modifier",
			statement: "ANALYZE NO_WRITE_TO_BINLOG TABLE appdb.t1, otherdb.",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "check optimize repair checksum flush preserve list across comma",
			statement: "CHECK TABLE appdb.t1, db2.; OPTIMIZE LOCAL TABLE appdb.t1, db3.; REPAIR TABLE appdb.t1, db4.; CHECKSUM TABLE appdb.t1, db5.; FLUSH TABLES appdb.t1, db6.",
			defaultDB: "appdb",
			want:      []string{"db2", "db3", "db4", "db5", "db6"},
		},
		{
			name:      "on duplicate key update and for update do not treat qualifiers as databases",
			statement: "INSERT INTO appdb.t1 (id) VALUES (1) ON DUPLICATE KEY UPDATE id = a.id; SELECT * FROM t1 u FOR UPDATE OF u.",
			defaultDB: "appdb",
		},
		{
			name:      "grant and revoke extract target db on ON and ignore role after FROM",
			statement: "GRANT SELECT, INSERT ON otherdb.t1 TO u; REVOKE UPDATE ON third_db.t2 FROM r.user",
			defaultDB: "appdb",
			want:      []string{"otherdb", "third_db"},
		},
		{
			name:      "show full columns and delete with modifiers extract db",
			statement: "SHOW FULL COLUMNS FROM otherdb.; DELETE LOW_PRIORITY IGNORE FROM third_db.t1 WHERE id = u.id",
			defaultDB: "appdb",
			want:      []string{"otherdb", "third_db"},
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
		{
			name:      "truncate without TABLE keyword extracts db while TRUNCATE function call does not",
			statement: "TRUNCATE otherdb.; SELECT TRUNCATE(a.amount, 2) FROM users a",
			defaultDB: "appdb",
			want:      []string{"otherdb"},
		},
		{
			name:      "drop sequence comma list and tidb import into extract dbs",
			statement: "DROP SEQUENCE IF EXISTS db1.s1, otherdb.; IMPORT INTO third_db.t1 FROM 's3://bucket/path'",
			defaultDB: "appdb",
			want:      []string{"db1", "otherdb", "third_db"},
		},
		{
			name:      "cache index and load index into cache extract table lists",
			statement: "CACHE INDEX db1.t1, db2. IN hot_cache; LOAD INDEX INTO CACHE db3.t1 INDEX (idx1), db4.",
			defaultDB: "appdb",
			want:      []string{"db1", "db2", "db3", "db4"},
		},
		{
			name:      "function return expression does not treat qualifier as database",
			statement: "CREATE FUNCTION fn() RETURNS INT RETURN a.x",
			defaultDB: "appdb",
		},
		{
			name:      "dollar and unicode identifiers preserved as single database qualifiers",
			statement: "SELECT sales$archive.t1., café$2. FROM café$2.users u JOIN users a$1 ON a$1.id = u.id",
			defaultDB: "appdb",
			want:      []string{"sales$archive", "café$2"},
		},
		{
			name:      "parenthesized table references in UPDATE and DELETE USING extract dbs",
			statement: "UPDATE LOW_PRIORITY (otherdb.t1 JOIN db2.t2 ON t1.id = t2.id) SET t1.c = a.c; DELETE FROM db3.t1 USING (db4.t1 JOIN db5.t2 ON t1.id = t2.id) WHERE a.id = 1",
			defaultDB: "appdb",
			want:      []string{"otherdb", "db2", "db3", "db4", "db5"},
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

	// Case-folded qualifier resolves to its canonical stored name via lazy enumeration on miss.
	caseFoldListCalls := 0
	var caseFoldLoaded []string
	PopulateMySQLFamilyCatalog(
		context.Background(),
		CompletionContext{
			DefaultDatabase: "appdb",
			ListDatabaseNames: func(_ context.Context, _ string) ([]string, error) {
				caseFoldListCalls++
				return []string{"appdb", "db_b"}, nil
			},
			Metadata: func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
				if db != "appdb" && db != "db_b" {
					return "", nil, nil
				}
				return db, model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: db}, nil, nil, storepb.Engine_MYSQL, true), nil
			},
		},
		"SELECT DB_B.orders. FROM DB_B.orders",
		len("SELECT DB_B.orders."),
		func(string, int) bool { return false },
		func(string) {},
		func(name string, _ *metadatapb.DatabaseSchemaMetadata) { caseFoldLoaded = append(caseFoldLoaded, name) },
		func(string) {},
	)
	require.Equal(t, 1, caseFoldListCalls)
	require.Equal(t, []string{"appdb", "db_b"}, caseFoldLoaded)
}

func TestCompleteMySQLFamily_PreservesDistinctDefinitionsAndUTF16Caret(t *testing.T) {
	got := CompleteMySQLFamily(
		CompletionContext{Scene: SceneTypeAll},
		"SELECT '😀', ",
		1,
		len("SELECT '")+2+len("', "), // UTF-16 code units for supplementary emoji 😀 = 2
		func(statement string, pos int) (string, int) {
			return statement, pos
		},
		func(string) bool { return false },
		func(stmt string, pos int) []Candidate {
			require.Equal(t, "SELECT '😀', ", stmt)
			require.Equal(t, len("SELECT '😀', "), pos)
			return []Candidate{
				{Type: CandidateTypeColumn, Text: "id", Definition: "appdb.t2.id | int"},
				{Type: CandidateTypeColumn, Text: "id", Definition: "appdb.t1.id | int"},
				{Type: CandidateTypeFunction, Text: "COUNT()"},
				{Type: CandidateTypeNone, Text: "ignored"},
			}
		},
	)

	require.Equal(t, []Candidate{
		{Type: CandidateTypeColumn, Text: "id", Definition: "appdb.t1.id | int"},
		{Type: CandidateTypeColumn, Text: "id", Definition: "appdb.t2.id | int"},
		{Type: CandidateTypeFunction, Text: "COUNT()"},
	}, got)
}

func TestDollarAndUnicodeIdentifiersConsistentAcrossHelpers(t *testing.T) {
	require.Equal(t, len("sales$archive"), extractIdentPrefixLen("USE sales$archive", len("USE sales$archive")))
	require.Equal(t, len("sales$"), extractIdentPrefixLen("SELECT * FROM sales$", len("SELECT * FROM sales$")))
	require.Equal(t, len("café$2"), extractIdentPrefixLen("USE café$2", len("USE café$2")))
	require.Equal(t, 1, extractIdentPrefixLen("sales$archive.t", len("sales$archive.t")))

	require.True(t, MySQLStatementReferencesDatabase("SELECT * FROM sales$archive.t1", "sales$archive"))
	require.False(t, MySQLStatementReferencesDatabase("SELECT * FROM sales$archive.t1", "archive"))
	require.False(t, MySQLStatementReferencesDatabase("SELECT * FROM café.t1", "fé"))

	require.False(t, identifierNeedsQuoting("sales$archive", nil))
	require.True(t, identifierNeedsQuoting("1$bad", nil))
}
