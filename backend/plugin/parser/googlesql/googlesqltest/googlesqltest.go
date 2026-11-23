// Package googlesqltest holds the mock metadata and assertion helpers shared by
// the bigquery and spanner query-span seam tests.
package googlesqltest

import (
	"context"
	"slices"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// BuildMockDatabaseMetadataGetter builds the metadata getter/lister pair the
// query-span tests use, keyed by database name.
func BuildMockDatabaseMetadataGetter(engine storepb.Engine, databaseMetadata []*metadatapb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadata {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, engine, true /* isObjectCaseSensitive */)
			}

			if databaseMetadata, ok := m[databaseName]; ok {
				return "", databaseMetadata, nil
			}

			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(_ context.Context, _ string) ([]string, error) {
			var names []string
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}

// DefaultSchemaTables wraps tables into a single default ("") schema.
func DefaultSchemaTables(tables ...*metadatapb.TableMetadata) []*metadatapb.SchemaMetadata {
	return []*metadatapb.SchemaMetadata{{Name: "", Tables: tables}}
}

// SourcesOf renders a result's source columns as sorted "[schema.]table.column"
// strings for compact assertions.
func SourcesOf(r base.QuerySpanResult) []string {
	out := make([]string, 0, len(r.SourceColumns))
	for c := range r.SourceColumns {
		name := c.Table + "." + c.Column
		if c.Schema != "" {
			name = c.Schema + "." + name
		}
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}
