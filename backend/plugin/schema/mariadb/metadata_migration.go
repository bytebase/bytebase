package mariadb

import (
	"context"

	"github.com/bytebase/omni/mariadb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterDiffMetadataMigration(storepb.Engine_MARIADB, mariaDBDiffMetadataMigration)
}

// mariaDBCatalog is omni's MariaDB catalog as the shared metadata diff sees it.
// The diff must run on MariaDB's own fork, not on omni's MySQL catalog: the MySQL
// loader drops WITH SYSTEM VERSIONING and installs the ROW START/ROW END columns
// as plain NOT NULL ones, so a diff there would silently recreate the table
// without history.
var mariaDBCatalog = omnicatalog.Engine[catalog.Catalog, catalog.LoadMetadataReport]{
	New: catalog.New,
	Load: func(c *catalog.Catalog, meta *metadatapb.DatabaseSchemaMetadata) (*catalog.LoadMetadataReport, error) {
		return c.LoadMetadataInto(context.Background(), meta, omnicatalog.Database)
	},
	Problems: func(report *catalog.LoadMetadataReport) []map[string]error {
		return []map[string]error{report.Degraded, report.Missing, report.Incomplete}
	},
	Diff: func(from, to *catalog.Catalog) (string, bool) {
		diff := catalog.Diff(from, to)
		if diff.IsEmpty() {
			return "", true
		}
		return catalog.GenerateMigration(from, to, diff).SQL(), false
	},
}

// mariaDBDiffMetadataMigration generates the migration between two synced
// snapshots, refusing one whose changed objects the catalog could not install as
// defined (see omnicatalog.DiffMetadata).
func mariaDBDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return omnicatalog.DiffMetadata(mariaDBCatalog, oldSchema.GetProto(), newSchema.GetProto())
}
