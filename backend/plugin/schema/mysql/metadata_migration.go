package mysql

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mysql/catalog"

	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

// mysqlCatalog is omni's MySQL catalog as the shared metadata diff sees it. Both
// snapshots load under omnicatalog.Database, so a database diffed against another
// database's changelog compares object by object; the generator's qualifiers for
// that name are stripped from the result.
var mysqlCatalog = omnicatalog.Engine[catalog.Catalog, catalog.LoadMetadataReport]{
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

// omniDiffMetadataMigration generates the migration between two synced snapshots
// with omni's catalog, refusing one whose changed objects the catalog could not
// install as defined (see omnicatalog.DiffMetadata).
func omniDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return omnicatalog.DiffMetadata(mysqlCatalog, oldSchema.GetProto(), newSchema.GetProto())
}
