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
