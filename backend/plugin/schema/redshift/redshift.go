package redshift

import (
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/redshift/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.Register(schema.Engine{
		Definitions: schema.Definitions{
			Database: GetDatabaseDefinition,
			Table:    GetTableDefinition,
			View:     GetViewDefinition,
		},
	}, storepb.Engine_REDSHIFT)
}

// GetTableDefinition renders one table as DDL, matching DatabaseDefinition
// without the section banner.
func GetTableDefinition(schemaName string, table *metadatapb.TableMetadata, _ []*metadatapb.SequenceMetadata) (string, error) {
	return catalog.GetTableDefinition(schemaName, table)
}

// GetViewDefinition renders one view as DDL, matching DatabaseDefinition
// without the section banner.
func GetViewDefinition(schemaName string, view *metadatapb.ViewMetadata) (string, error) {
	return catalog.GetViewDefinition(schemaName, view)
}

// GetDatabaseDefinition renders a Redshift database snapshot as DDL.
func GetDatabaseDefinition(_ schema.GetDefinitionContext, to *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(to)
}
