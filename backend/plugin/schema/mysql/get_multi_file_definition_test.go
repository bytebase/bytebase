package mysql

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestGetMultiFileDatabaseDefinition pins the registry dispatch and conversion
// from omni catalog.DefinitionFile to schema.MultiFileSchemaResult.
func TestGetMultiFileDatabaseDefinition(t *testing.T) {
	metaProto := &metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{
				Name:    "audit_log",
				Engine:  "InnoDB",
				Columns: []*metadatapb.ColumnMetadata{{Name: "id", Position: 1, Type: "int"}},
			}},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "recent_audit",
				Definition: "select `audit_log`.`id` AS `id` from `audit_log`",
			}},
		}},
	}

	result, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{
		SkipBackupSchema: true,
	}, metaProto)
	require.NoError(t, err)
	require.Len(t, result.Files, 2)

	byName := make(map[string]string, len(result.Files))
	for _, f := range result.Files {
		byName[f.Name] = f.Content
	}
	require.Contains(t, byName["tables/audit_log.sql"], "CREATE TABLE `audit_log`")
	require.Contains(t, byName["views/recent_audit.sql"], "CREATE OR REPLACE VIEW `recent_audit`")

	emptyResult, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{})
	require.NoError(t, err)
	require.Empty(t, emptyResult.Files)
}
