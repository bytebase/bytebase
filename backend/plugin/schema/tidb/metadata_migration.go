package tidb

import (
	"github.com/bytebase/omni/tidb/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

func init() {
	omnicatalog.RegisterMetadataMigration(storepb.Engine_TIDB, omnicatalog.MySQLFamilyEngine(
		catalog.New,
		func(report *catalog.LoadMetadataReport) []map[string]error {
			return []map[string]error{report.Degraded, report.Missing, report.Incomplete}
		},
		catalog.Diff,
		catalog.GenerateMigration,
		nil,
	))
}
