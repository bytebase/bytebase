package mysql

import (
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	// MySQL-only for now: OceanBase is intentionally NOT registered pending validation
	// against a live OceanBase oracle.
	schema.RegisterGetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, GetMultiFileDatabaseDefinition)
}

// GetMultiFileDatabaseDefinition emits the database schema as one SDL file per object.
// Each declarative rollout re-concatenates the files (action/command/file.go) into the
// statement set the single-file SDL dump carries.
func GetMultiFileDatabaseDefinition(_ schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (*schema.MultiFileSchemaResult, error) {
	files, err := catalog.MultiFileDefinition(metadata)
	if err != nil {
		return nil, err
	}
	var result []schema.File
	if files != nil {
		result = make([]schema.File, 0, len(files))
	}
	for _, file := range files {
		result = append(result, schema.File{Name: file.Name, Content: file.Content})
	}
	return &schema.MultiFileSchemaResult{Files: result}, nil
}
