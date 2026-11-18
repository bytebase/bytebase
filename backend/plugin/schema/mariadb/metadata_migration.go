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
		report, err := c.LoadMetadataInto(context.Background(), meta, omnicatalog.Database)
		if err != nil {
			return report, err
		}
		clearImplicitTableCharsets(c, meta)
		return report, nil
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

// clearImplicitTableCharsets restores the legacy generator's omission of a
// charset or collation the snapshot does not specify. The catalog fills an
// implicit table with its own release defaults, which for 11.8 includes
// utf8mb4_uca1400_ai_ci, a collation MariaDB 10.7 (still supported) rejects.
// Leaving the option unset lets each target server apply its own default.
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
			}
		}
	}
}

// mariaDBDiffMetadataMigration generates the migration between two synced
// snapshots, refusing one whose changed objects the catalog could not install as
// defined (see omnicatalog.DiffMetadata).
func mariaDBDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return omnicatalog.DiffMetadata(mariaDBCatalog, oldSchema.GetProto(), newSchema.GetProto())
}
