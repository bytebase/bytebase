package clickhouse

import (
	"github.com/bytebase/omni/clickhouse/deparse"
	metadatapb "github.com/bytebase/omni/metadata"

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
	}, storepb.Engine_CLICKHOUSE)
}

func GetDatabaseDefinition(_ schema.GetDefinitionContext, to *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return deparse.DatabaseDefinition(to)
}

func GetTableDefinition(tableName string, table *metadatapb.TableMetadata, _ []*metadatapb.SequenceMetadata) (string, error) {
	return deparse.GetTableDefinition(tableName, table)
}

func GetViewDefinition(viewName string, view *metadatapb.ViewMetadata) (string, error) {
	return deparse.GetViewDefinition(viewName, view)
}
