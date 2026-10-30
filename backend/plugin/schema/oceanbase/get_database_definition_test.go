package oceanbase

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestDefinitionGetters pins that each definition getter reaches this package for
// OceanBase, and that the database definition it writes parses back through the
// OceanBase parser. The DDL itself is pinned by the goldens in schema/mysql, which
// this package's copy mirrors.
func TestDefinitionGetters(t *testing.T) {
	table := &metadatapb.TableMetadata{
		Name:    "t",
		Columns: []*metadatapb.ColumnMetadata{{Name: "a", Type: "int"}},
	}

	for _, tc := range []struct {
		name string
		get  func() (string, error)
	}{
		{"table", func() (string, error) {
			return schema.GetTableDefinition(storepb.Engine_OCEANBASE, "", table, nil)
		}},
		{"view", func() (string, error) {
			return schema.GetViewDefinition(storepb.Engine_OCEANBASE, "", &metadatapb.ViewMetadata{Name: "v", Definition: "SELECT `a` FROM `t`"})
		}},
		{"function", func() (string, error) {
			return schema.GetFunctionDefinition(storepb.Engine_OCEANBASE, "", &metadatapb.FunctionMetadata{Name: "f", Definition: "CREATE FUNCTION `f`() RETURNS int RETURN 1"})
		}},
		{"procedure", func() (string, error) {
			return schema.GetProcedureDefinition(storepb.Engine_OCEANBASE, "", &metadatapb.ProcedureMetadata{Name: "p", Definition: "CREATE PROCEDURE `p`() BEGIN END"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, err := tc.get()
			require.NoError(t, err)
			require.NotEmpty(t, definition)
		})
	}

	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{table}}},
	}
	definition, err := schema.GetDatabaseDefinition(storepb.Engine_OCEANBASE, schema.GetDefinitionContext{}, metadata)
	require.NoError(t, err)
	require.NotEmpty(t, definition)

	_, err = schema.GetDatabaseMetadata(storepb.Engine_OCEANBASE, definition)
	require.NoError(t, err, "the definition this package writes must parse back")
}
