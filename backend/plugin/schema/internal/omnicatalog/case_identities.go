package omnicatalog

import (
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
)

// rejectCaseOnlyIdentities refuses a snapshot whose table or view names differ
// only by case. The MySQL-family catalogs key relations by lower-cased name, so
// a case-only rename, or two relations that fold to one key, would be conflated.
// A case-sensitive instance without that ambiguity diffs normally.
func rejectCaseOnlyIdentities(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, oldCaseSensitive, newCaseSensitive bool) error {
	if !oldCaseSensitive && !newCaseSensitive {
		return nil
	}
	for _, meta := range []*metadatapb.DatabaseSchemaMetadata{oldMeta, newMeta} {
		for _, schema := range meta.GetSchemas() {
			seen := make(map[string]string)
			for _, name := range relationNames(schema) {
				lower := strings.ToLower(name)
				if previous, ok := seen[lower]; ok && previous != name {
					return errors.Errorf("case-sensitive schema has %q and %q, which the catalog cannot distinguish", previous, name)
				}
				seen[lower] = name
			}
		}
	}
	oldNames := relationNameMap(oldMeta)
	for _, schema := range newMeta.GetSchemas() {
		for _, name := range relationNames(schema) {
			if previous, ok := oldNames[strings.ToLower(name)]; ok && previous != name {
				return errors.Errorf("case-only rename %q to %q is not supported by the catalog diff", previous, name)
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
