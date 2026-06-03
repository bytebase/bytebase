package oracle

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_ORACLE, GetDatabaseMetadata)
}

// GetDatabaseMetadata parses the Oracle schema text and returns the database metadata.
func GetDatabaseMetadata(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	return GetDatabaseMetadataOmni(schemaText)
}
