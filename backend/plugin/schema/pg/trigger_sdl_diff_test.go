package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestTriggerDiffStructure(t *testing.T) {
	// Test that TriggerDiff has all required fields for SDL processing
	currentSDL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100)
		);

		CREATE FUNCTION audit_log() RETURNS TRIGGER AS $$
		BEGIN
			RAISE NOTICE 'Audit';
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER audit_trigger
		AFTER INSERT ON users
		FOR EACH ROW EXECUTE FUNCTION audit_log();
	`

	previousSDL := ""

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.Len(t, diff.TableChanges, 1, "Should have one table change")

	tableDiff := diff.TableChanges[0]
	require.Len(t, tableDiff.TriggerChanges, 1, "Should have one trigger change")

	triggerDiff := tableDiff.TriggerChanges[0]
	assert.Equal(t, schema.MetadataDiffActionCreate, triggerDiff.Action)
	assert.NotNil(t, triggerDiff.NewASTNode, "Should have AST node")
	// These fields should exist after we update the structure
	// assert.Equal(t, "public", triggerDiff.SchemaName)
	// assert.Equal(t, "users", triggerDiff.TableName)
	// assert.Equal(t, "audit_trigger", triggerDiff.TriggerName)
}

func TestCreateTriggerExtraction(t *testing.T) {
	tests := []struct {
		name             string
		sql              string
		expectedTriggers int
	}{
		{
			name: "Simple trigger",
			sql: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			expectedTriggers: 1,
		},
		{
			name: "Multiple triggers on same table",
			sql: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
				CREATE TRIGGER t2 BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			expectedTriggers: 2,
		},
		{
			name: "Trigger with schema-qualified table",
			sql: `
				CREATE SCHEMA app;
				CREATE TABLE app.users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION app.f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON app.users FOR EACH ROW EXECUTE FUNCTION app.f();
			`,
			expectedTriggers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.sql, "", nil, nil)
			require.NoError(t, err)

			totalTriggers := 0
			for _, tableDiff := range diff.TableChanges {
				totalTriggers += len(tableDiff.TriggerChanges)
			}

			assert.Equal(t, tt.expectedTriggers, totalTriggers)
		})
	}
}

func TestTriggerCommentExtraction(t *testing.T) {
	sql := `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
		COMMENT ON TRIGGER audit_trigger ON users IS 'Audit log trigger';
	`

	diff, err := GetSDLDiff(sql, "", nil, nil)
	require.NoError(t, err)

	// Check that comment change exists
	found := false
	for _, commentDiff := range diff.CommentChanges {
		if commentDiff.ObjectType == schema.CommentObjectTypeTrigger &&
			commentDiff.ObjectName == "audit_trigger" &&
			commentDiff.NewComment == "Audit log trigger" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should have trigger comment")
}

func TestDropTriggerMigration(t *testing.T) {
	previousSDL := `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`
	currentSDL := `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
	`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migration, err := generateMigration(diff)
	require.NoError(t, err)

	assert.Contains(t, migration, "DROP TRIGGER")
	assert.Contains(t, migration, "audit_trigger")
	assert.Contains(t, migration, "ON")
	assert.Contains(t, migration, "users")
}
