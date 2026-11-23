package tidb

import (
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/tidb/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_TIDB, GetDatabaseMetadata)
}

// catalogDBName is the throwaway scratch database used when schemaText selects
// none of its own; its name is chosen not to collide with user identifiers.
const catalogDBName = "_bytebase_schema_scratch"

// GetDatabaseMetadata parses TiDB schema DDL text and returns database metadata
// using the omni TiDB catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return omnicatalog.ExtractDatabaseMetadata(
		schemaText,
		catalogDBName,
		catalog.New,
		execCatalog,
		(*catalog.Catalog).CurrentDatabase,
		func(c *catalog.Catalog, dbName string) *metadatapb.DatabaseSchemaMetadata {
			proto := c.DatabaseMetadata(dbName)
			if dbName == catalogDBName {
				// Return schema-relative metadata when the DDL selected no database of its own.
				// Real-database qualifiers stay intact to match information_schema.VIEWS sync.
				proto.Name = ""
				prefix := "`" + catalogDBName + "`."
				for _, s := range proto.GetSchemas() {
					for _, v := range s.GetViews() {
						v.Definition = strings.ReplaceAll(v.Definition, prefix, "")
					}
				}
			}
			return proto
		},
	)
}
