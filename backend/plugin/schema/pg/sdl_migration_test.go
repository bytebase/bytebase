package pg

import (
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

const eventTriggerFunctionSDL = `
CREATE FUNCTION audit_ddl() RETURNS event_trigger
LANGUAGE plpgsql AS $$
BEGIN
END;
$$;
`

const eventTriggerSDL = eventTriggerFunctionSDL + `
CREATE EVENT TRIGGER audit_ddl_start
	ON ddl_command_start
	WHEN TAG IN ('CREATE TABLE')
	EXECUTE FUNCTION audit_ddl();
`

const eventTriggerModifiedSDL = eventTriggerFunctionSDL + `
CREATE EVENT TRIGGER audit_ddl_start
	ON ddl_command_end
	WHEN TAG IN ('CREATE TABLE')
	EXECUTE FUNCTION audit_ddl();
`

const eventTriggerFunctionOnlySDL = eventTriggerFunctionSDL

func diffPostgresSDL(t *testing.T, fromSDL, toSDL string) string {
	t.Helper()
	sql, err := schema.DiffSDLMigration(storepb.Engine_POSTGRES, strings.TrimSpace(fromSDL), strings.TrimSpace(toSDL), "")
	require.NoError(t, err)
	return sql
}

// omniSDLMigration is a test helper that runs the omni SDL migration pipeline:
// from = LoadSDL(fromSDL), to = LoadSDL(toSDL), Diff, GenerateMigration.
func omniSDLMigration(t *testing.T, fromSDL, toSDL string) string {
	t.Helper()
	from, err := catalog.LoadSDL(strings.TrimSpace(fromSDL))
	require.NoError(t, err)
	to, err := catalog.LoadSDL(strings.TrimSpace(toSDL))
	require.NoError(t, err)
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return ""
	}
	plan := catalog.GenerateMigration(from, to, diff)
	return plan.SQL()
}

func TestOmniSDLMigration_CreateTable(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE,
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "users")
}

func TestOmniSDLMigration_DropTable(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE old_table (
			id INTEGER PRIMARY KEY,
			data TEXT
		);
	`, "")
	require.Contains(t, sql, "DROP TABLE")
	require.Contains(t, sql, "old_table")
}

func TestOmniSDLMigration_AddColumn(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255)
		);
	`)
	require.Contains(t, sql, "ALTER TABLE")
	require.Contains(t, sql, "email")
}

func TestOmniSDLMigration_DropColumn(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255)
		);
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`)
	require.Contains(t, sql, "ALTER TABLE")
	require.Contains(t, sql, "DROP COLUMN")
	require.Contains(t, sql, "email")
}

func TestOmniSDLMigration_CreateView(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT TRUE
		);
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT TRUE
		);
		CREATE VIEW active_users AS SELECT id, name FROM users WHERE active = TRUE;
	`)
	require.Contains(t, sql, "CREATE VIEW")
	require.Contains(t, sql, "active_users")
}

func TestOmniSDLMigration_DropView(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT TRUE
		);
		CREATE VIEW active_users AS SELECT id, name FROM users WHERE active = TRUE;
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT TRUE
		);
	`)
	require.Contains(t, sql, "DROP VIEW")
	require.Contains(t, sql, "active_users")
}

func TestOmniSDLMigration_RecordReturningFunctionView(t *testing.T) {
	sdl := `
CREATE FUNCTION public.record_pair(OUT id integer, OUT name text)
RETURNS record
LANGUAGE sql
AS $$
	SELECT 1::integer, 'alice'::text;
$$;

CREATE VIEW public.record_pair_view AS
SELECT *
FROM public.record_pair();
`
	sql := omniSDLMigration(t, sdl, sdl)
	require.Empty(t, sql)
}

func TestOmniSDLMigration_CreateFunction(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE FUNCTION add_numbers(a INTEGER, b INTEGER) RETURNS INTEGER
		LANGUAGE plpgsql AS $$
		BEGIN
			RETURN a + b;
		END;
		$$;
	`)
	require.Contains(t, sql, "CREATE FUNCTION")
	require.Contains(t, sql, "add_numbers")
}

func TestOmniSDLMigration_CreateSequence(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE SEQUENCE order_seq INCREMENT BY 1 MINVALUE 1 START WITH 1;
	`)
	require.Contains(t, sql, "CREATE SEQUENCE")
	require.Contains(t, sql, "order_seq")
}

func TestOmniSDLMigration_CreateEnum(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TYPE mood AS ENUM ('sad', 'ok', 'happy');
	`)
	require.Contains(t, sql, "CREATE TYPE")
	require.Contains(t, sql, "mood")
}

func TestOmniSDLMigration_AlterEnum_AddValue(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TYPE mood AS ENUM ('sad', 'ok', 'happy');
	`, `
		CREATE TYPE mood AS ENUM ('sad', 'ok', 'happy', 'ecstatic');
	`)
	require.Contains(t, sql, "ecstatic")
}

func TestOmniSDLMigration_CreateIndex(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL
		);
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL
		);
		CREATE INDEX idx_users_email ON users (email);
	`)
	require.Contains(t, sql, "CREATE INDEX")
	require.Contains(t, sql, "idx_users_email")
}

func TestOmniSDLMigration_CreateTrigger(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			updated_at TIMESTAMP
		);
		CREATE FUNCTION update_timestamp() RETURNS TRIGGER
		LANGUAGE plpgsql AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$;
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			updated_at TIMESTAMP
		);
		CREATE FUNCTION update_timestamp() RETURNS TRIGGER
		LANGUAGE plpgsql AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER trg_update_timestamp
			BEFORE UPDATE ON users
			FOR EACH ROW EXECUTE FUNCTION update_timestamp();
	`)
	require.Contains(t, sql, "CREATE TRIGGER")
	require.Contains(t, sql, "trg_update_timestamp")
}

func TestOmniSDLMigration_EventTriggerSourceSDL(t *testing.T) {
	sdl := `
		CREATE FUNCTION audit_ddl() RETURNS event_trigger
		LANGUAGE plpgsql AS $$
		BEGIN
		END;
		$$;

		CREATE EVENT TRIGGER audit_ddl_start
			ON ddl_command_start
			WHEN TAG IN ('CREATE TABLE')
			EXECUTE FUNCTION audit_ddl();
	`
	sql := omniSDLMigration(t, sdl, sdl)
	require.Empty(t, sql)
}

func TestDiffSDLMigration_EventTriggerChanges(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name:    "add event trigger",
			fromSDL: eventTriggerFunctionOnlySDL,
			toSDL:   eventTriggerSDL,
			contains: []string{
				`CREATE EVENT TRIGGER "audit_ddl_start" ON "ddl_command_start"`,
				`WHEN TAG IN ('CREATE TABLE')`,
				`EXECUTE FUNCTION "public"."audit_ddl"()`,
			},
		},
		{
			name:    "drop event trigger",
			fromSDL: eventTriggerSDL,
			toSDL:   eventTriggerFunctionOnlySDL,
			contains: []string{
				`DROP EVENT TRIGGER "audit_ddl_start"`,
			},
		},
		{
			name:    "modify event trigger",
			fromSDL: eventTriggerSDL,
			toSDL:   eventTriggerModifiedSDL,
			contains: []string{
				`DROP EVENT TRIGGER "audit_ddl_start"`,
				`CREATE EVENT TRIGGER "audit_ddl_start" ON "ddl_command_end"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := diffPostgresSDL(t, tt.fromSDL, tt.toSDL)
			for _, want := range tt.contains {
				require.Contains(t, sql, want)
			}
		})
	}
}

func TestOmniSDLMigration_Comment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY
		);
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY
		);
		COMMENT ON TABLE users IS 'User accounts';
	`)
	require.Contains(t, sql, "COMMENT ON TABLE")
	require.Contains(t, sql, "User accounts")
}

func TestOmniSDLMigration_NoChange(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`)
	require.Empty(t, sql)
}

func TestOmniSDLMigration_CreateSchema(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE SCHEMA analytics;
		CREATE TABLE analytics.events (
			id SERIAL PRIMARY KEY,
			event_name TEXT NOT NULL
		);
	`)
	require.Contains(t, sql, "CREATE SCHEMA")
	require.Contains(t, sql, "analytics")
	require.Contains(t, sql, "CREATE TABLE")
}

func TestOmniSDLMigration_AddConstraint(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			total DECIMAL(10,2)
		);
	`, `
		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			total DECIMAL(10,2),
			CONSTRAINT chk_total CHECK (total >= 0)
		);
	`)
	require.Contains(t, sql, "chk_total")
}

func TestOmniSDLMigration_AddForeignKey(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE categories (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			category_id INTEGER
		);
	`, `
		CREATE TABLE categories (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			category_id INTEGER,
			CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id)
		);
	`)
	require.Contains(t, sql, "fk_category")
	require.Contains(t, sql, "FOREIGN KEY")
}

func TestOmniSDLMigration_MultipleChanges(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
		CREATE TABLE old_table (
			id INTEGER PRIMARY KEY
		);
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255)
		);
		CREATE TABLE new_table (
			id SERIAL PRIMARY KEY,
			data TEXT
		);
	`)
	require.Contains(t, sql, "DROP TABLE")
	require.Contains(t, sql, "old_table")
	require.Contains(t, sql, "email")
	require.Contains(t, sql, "new_table")
}

// TestOmniFilter_BbdataarchiveSchemaFiltered verifies that objects in the
// bbdataarchive schema are excluded from migration output.
func TestOmniFilter_BbdataarchiveSchemaFiltered(t *testing.T) {
	// Source has tables in both public and bbdataarchive schemas.
	sourceMetadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
			{
				Name: "bbdataarchive",
				Tables: []*metadatapb.TableMetadata{
					{Name: "archived_users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
		},
	}
	// Target is empty — should only generate DROP for public, not bbdataarchive.
	targetSDL := ""

	sourceMeta := model.NewDatabaseMetadata(sourceMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, sourceMeta)
	require.NoError(t, err)

	sql, err := schema.DiffSDLMigration(storepb.Engine_POSTGRES, sourceSDL, targetSDL, "")
	require.NoError(t, err)

	// bbdataarchive objects must not appear in migration.
	require.NotContains(t, sql, "bbdataarchive")
	require.NotContains(t, sql, "archived_users")
	// public objects should be dropped.
	require.Contains(t, sql, "users")
}

// TestOmniFilter_SkipBackupSchemaInSDLGeneration verifies that MetadataToSDL
// excludes the backup schema entirely from generated SDL text.
func TestOmniFilter_SkipBackupSchemaInSDLGeneration(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "t1", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
			{
				Name: "bbdataarchive",
				Tables: []*metadatapb.TableMetadata{
					{Name: "backup_t1", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
		},
	}
	meta := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)

	sdl, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, meta)
	require.NoError(t, err)

	require.NotContains(t, sdl, "bbdataarchive")
	require.NotContains(t, sdl, "backup_t1")
	require.Contains(t, sdl, "t1")
}

// TestOmniFilter_SkipDumpObjectsExcluded verifies that objects marked with
// SkipDump (e.g., extension-created objects) are excluded from SDL generation.
func TestOmniFilter_SkipDumpObjectsExcluded(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "user_table", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
					{Name: "ext_table", SkipDump: true, Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
				Functions: []*metadatapb.FunctionMetadata{
					{Name: "my_func", Definition: "CREATE FUNCTION my_func() RETURNS void LANGUAGE sql AS $$ SELECT 1 $$;"},
					{Name: "ext_func", SkipDump: true, Definition: "CREATE FUNCTION ext_func() RETURNS void LANGUAGE sql AS $$ SELECT 1 $$;"},
				},
			},
		},
	}
	meta := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)

	sdl, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, meta)
	require.NoError(t, err)

	require.Contains(t, sdl, "user_table")
	require.Contains(t, sdl, "my_func")
	require.NotContains(t, sdl, "ext_table")
	require.NotContains(t, sdl, "ext_func")
}

// TestOmniFilter_ExtensionPreservedWhenArchiveFiltered verifies that extension
// operations are preserved even when bbdataarchive objects are filtered.
func TestOmniFilter_ExtensionPreservedWhenArchiveFiltered(t *testing.T) {
	sql := omniSDLMigration(t, "", `CREATE EXTENSION IF NOT EXISTS "citext";`)
	// Extension creation should not be filtered.
	require.Contains(t, sql, "citext")
}

// TestOmniFilter_NoChangesForIdenticalSchemas verifies that two identical
// schemas with backup objects produce no migration (backup objects are
// excluded from both sides consistently).
func TestOmniFilter_NoChangesForIdenticalSchemas(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{Name: "users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
			{
				Name: "bbdataarchive",
				Tables: []*metadatapb.TableMetadata{
					{Name: "old_users", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
			},
		},
	}

	meta := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, false)
	// Use MetadataToSDL to get the exact SDL that would be generated, then diff against itself.
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, meta)
	require.NoError(t, err)
	sql, err := schema.SDLMigration(storepb.Engine_POSTGRES, sourceSDL, meta, "")
	require.NoError(t, err)
	require.Empty(t, sql, "identical public schemas with backup objects should produce no migration")
}
