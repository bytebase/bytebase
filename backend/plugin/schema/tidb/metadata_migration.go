package tidb

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/tidb/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterDiffMetadataMigration(storepb.Engine_TIDB, tidbDiffMetadataMigration)
}

// tidbCatalog is omni's TiDB catalog as the shared metadata diff sees it. Both
// snapshots load under omnicatalog.Database so a database diffed against another
// database's changelog compares object by object.
var tidbCatalog = omnicatalog.Engine[catalog.Catalog, catalog.LoadMetadataReport]{
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

// tidbDiffMetadataMigration generates the migration between two synced snapshots,
// refusing one whose changed objects the catalog could not install as defined (see
// omnicatalog.DiffMetadata).
func tidbDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	if err := omnicatalog.RejectCaseOnlyIdentities(oldSchema.GetProto(), newSchema.GetProto(), oldSchema.GetIsObjectCaseSensitive(), newSchema.GetIsObjectCaseSensitive()); err != nil {
		return "", err
	}
	return omnicatalog.DiffMetadata(tidbCatalog, oldSchema.GetProto(), newSchema.GetProto())
}
