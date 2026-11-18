package oceanbase

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/oceanbase/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterDiffMetadataMigration(storepb.Engine_OCEANBASE, oceanBaseDiffMetadataMigration)
}

// oceanBaseCatalog is omni's OceanBase catalog as the shared metadata diff sees
// it. The diff runs on OceanBase's own fork rather than on omni's MySQL catalog:
// the fork is where OceanBase's index options, TTL and column groups load and
// render as OceanBase does.
var oceanBaseCatalog = omnicatalog.Engine[catalog.Catalog, catalog.LoadMetadataReport]{
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

// oceanBaseDiffMetadataMigration generates the migration between two synced
// snapshots, refusing one whose changed objects the catalog could not install as
// defined (see omnicatalog.DiffMetadata).
func oceanBaseDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return omnicatalog.DiffMetadata(oceanBaseCatalog, oldSchema.GetProto(), newSchema.GetProto())
}
