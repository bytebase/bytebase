package mariadb

import (
	"github.com/bytebase/omni/mariadb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_MARIADB, GetDatabaseMetadata)
}

// GetDatabaseMetadata parses MariaDB schema DDL text and returns database
// metadata using the omni MariaDB catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	return omnicatalog.ExtractDatabaseMetadata(
		schemaText,
		"tmp",
		catalog.New,
		execCatalog,
		nil,
		func(c *catalog.Catalog, dbName string) *metadatapb.DatabaseSchemaMetadata {
			return c.DatabaseMetadata(dbName)
		},
	)
}
