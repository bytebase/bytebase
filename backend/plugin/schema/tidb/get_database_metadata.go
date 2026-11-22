package tidb

import (
	"fmt"
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

// catalogDBName is the scratch database the schema DDL is simulated in when it
// selects none of its own. The name is deliberately unlikely to appear as a
// real table, alias or literal, so stripping its view-definition qualifier
// (tidbViewDefinition) is unambiguous.
const catalogDBName = "_bytebase_schema_scratch"

// GetDatabaseMetadata parses TiDB schema DDL text and returns database metadata
// using the omni TiDB catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return omnicatalog.ExtractDatabaseMetadata(
		schemaText,
		catalogDBName,
		catalog.New,
		execCatalog,
		func(c *catalog.Catalog) string { return c.CurrentDatabase() },
		func(c *catalog.Catalog, dbName string) *metadatapb.DatabaseSchemaMetadata {
			proto := c.DatabaseMetadata(dbName)
			if dbName == catalogDBName {
				// When the DDL selected no database of its own, the objects sit in the
				// scratch database; return schema-relative metadata (empty database
				// name and unqualified same-database view bodies).
				proto.Name = ""
				for _, s := range proto.GetSchemas() {
					for _, v := range s.GetViews() {
						v.Definition = tidbViewDefinition(v.Definition, dbName)
					}
				}
			}
			return proto
		},
	)
}

// tidbViewDefinition drops the qualifier omni adds to a view body's
// same-database references ONLY when that database is the throwaway scratch one,
// so the definition does not mention it. When the DDL selected a real database,
// its qualifier is left intact — live TiDB qualifies same-database references in
// information_schema.VIEWS the same way, so the stored definition matches sync.
// The scratch name cannot occur as a real alias or literal, so this replacement
// is unambiguous.
func tidbViewDefinition(definition, dbName string) string {
	if dbName != catalogDBName {
		return definition
	}
	return strings.ReplaceAll(definition, fmt.Sprintf("`%s`.", dbName), "")
}
