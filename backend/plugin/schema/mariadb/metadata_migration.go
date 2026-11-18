package mariadb

import (
	"context"
	"strings"

	"github.com/bytebase/omni/mariadb/catalog"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"

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
			default:
			}
		}
	}
}

// mariaDBDiffMetadataMigration generates the migration between two synced
// snapshots, refusing one whose changed objects the catalog could not install as
// defined (see omnicatalog.DiffMetadata).
func mariaDBDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	if err := rejectCaseSensitiveIdentities(oldSchema, newSchema); err != nil {
		return "", err
	}
	return omnicatalog.DiffMetadata(mariaDBCatalog, oldSchema.GetProto(), newSchema.GetProto())
}

// rejectCaseSensitiveIdentities refuses a snapshot whose table or view names
// differ only by case. omni's MariaDB catalog keys objects by lower-cased name,
// so a case-only rename or a pair of such objects would be conflated. Snapshots
// with no such ambiguity diff correctly even on a case-sensitive instance.
func rejectCaseSensitiveIdentities(oldSchema, newSchema *model.DatabaseMetadata) error {
	if !oldSchema.GetIsObjectCaseSensitive() && !newSchema.GetIsObjectCaseSensitive() {
		return nil
	}
	for _, schema := range []*model.DatabaseMetadata{oldSchema, newSchema} {
		for _, s := range schema.GetProto().GetSchemas() {
			seen := make(map[string]string)
			for _, name := range relationNames(s) {
				lower := strings.ToLower(name)
				if previous, ok := seen[lower]; ok && previous != name {
					return errors.Errorf("case-sensitive schema has %q and %q, which the MariaDB catalog cannot distinguish", previous, name)
				}
				seen[lower] = name
			}
		}
	}
	oldNames := relationNameMap(oldSchema.GetProto())
	for _, schema := range newSchema.GetProto().GetSchemas() {
		for _, name := range relationNames(schema) {
			if previous, ok := oldNames[strings.ToLower(name)]; ok && previous != name {
				return errors.Errorf("case-only rename %q to %q is not supported by the MariaDB catalog diff", previous, name)
			}
		}
	}
	return nil
}

func relationNames(schema *metadatapb.SchemaMetadata) []string {
	names := make([]string, 0, len(schema.GetTables())+len(schema.GetViews()))
	for _, table := range schema.GetTables() {
		names = append(names, table.GetName())
	}
	for _, view := range schema.GetViews() {
		names = append(names, view.GetName())
	}
	return names
}

func relationNameMap(meta *metadatapb.DatabaseSchemaMetadata) map[string]string {
	names := make(map[string]string)
	for _, schema := range meta.GetSchemas() {
		for _, name := range relationNames(schema) {
			names[strings.ToLower(name)] = name
		}
	}
	return names
}
