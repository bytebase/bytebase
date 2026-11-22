package mariadb

import (
	"github.com/bytebase/omni/mariadb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
)

// Runs on MariaDB's own catalog fork rather than MySQL's: the MySQL loader
// drops WITH SYSTEM VERSIONING and installs ROW START/END as plain NOT NULL columns.
func init() {
	omnicatalog.RegisterMetadataMigration(storepb.Engine_MARIADB, omnicatalog.MySQLFamilyEngine(
		catalog.New,
		func(report *catalog.LoadMetadataReport) []map[string]error {
			return []map[string]error{report.Degraded, report.Missing, report.Incomplete}
		},
		catalog.Diff,
		catalog.GenerateMigration,
		clearImplicitTableCharsets,
	))
}

// clearImplicitTableCharsets restores the legacy generator's omission of a
// charset or collation the snapshot does not specify. The catalog fills an
// implicit table with its own release defaults, which for 11.8 includes
// utf8mb4_uca1400_ai_ci, a collation MariaDB 10.7 (still supported) rejects.
func clearImplicitTableCharsets(c *catalog.Catalog, meta *metadatapb.DatabaseSchemaMetadata) {
	db := c.GetDatabase(omnicatalog.Database)
	if db == nil {
		return
	}
	for _, schema := range meta.GetSchemas() {
		for _, table := range schema.GetTables() {
			loaded := db.GetTable(table.GetName())
			if loaded == nil {
				continue
			}
			switch {
			case table.GetCharset() == "" && table.GetCollation() == "":
				loaded.Charset, loaded.Collation = "", ""
			case table.GetCollation() == "":
				loaded.Collation = ""
			default:
			}
		}
	}
}
