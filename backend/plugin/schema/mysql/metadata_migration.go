package mysql

import (
	"github.com/bytebase/omni/mysql/catalog"

	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

var mysqlCatalog = omnicatalog.MySQLFamilyEngine(
	catalog.New,
	func(report *catalog.LoadMetadataReport) []map[string]error {
		return []map[string]error{report.Degraded, report.Missing, report.Incomplete}
	},
	catalog.Diff,
	catalog.GenerateMigration,
	nil,
)

func omniDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return omnicatalog.DiffMetadataMigration(mysqlCatalog, oldSchema, newSchema)
}
