package oceanbase

import (
	"github.com/bytebase/omni/oceanbase/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

// Runs on OceanBase's own catalog fork so OceanBase-specific index options,
// TTL, and column groups load and render accurately.
func init() {
	omnicatalog.RegisterMetadataMigration(storepb.Engine_OCEANBASE, omnicatalog.MySQLFamilyEngine(
		catalog.New,
		func(report *catalog.LoadMetadataReport) []map[string]error {
			return []map[string]error{report.Degraded, report.Missing, report.Incomplete}
		},
		catalog.Diff,
		catalog.GenerateMigration,
		nil,
	))
}
