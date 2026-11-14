package mysql

import (
	"context"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mysql/catalog"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/store/model"
)

// omniDiffMetadataMigration generates the migration between two synced snapshots
// with omni's catalog. Both load under sdlDatabaseContext, so a database diffed
// against another database's changelog compares object by object.
//
// An object omni cannot install as the snapshot defines it (a stand-in, a missing
// routine, or a table without a clause that did not parse) is safe to leave in
// place only when it is the same on both sides: the two loads then agree on it and
// the diff skips it. Where it differs, the diff would compare stand-ins, so the
// migration is refused rather than guessed.
func omniDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	oldMeta, newMeta := oldSchema.GetProto(), newSchema.GetProto()
	from, fromReport, err := loadMetadataCatalog(oldMeta)
	if err != nil {
		return "", errors.Wrap(err, "failed to load source schema")
	}
	to, toReport, err := loadMetadataCatalog(newMeta)
	if err != nil {
		return "", errors.Wrap(err, "failed to load target schema")
	}
	if err := unloadableChanges(oldMeta, newMeta, fromReport, toReport); err != nil {
		return "", err
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}
	return stripDatabaseContext(catalog.GenerateMigration(from, to, diff).SQL()), nil
}

func loadMetadataCatalog(meta *metadatapb.DatabaseSchemaMetadata) (*catalog.Catalog, *catalog.LoadMetadataReport, error) {
	if meta == nil {
		meta = &metadatapb.DatabaseSchemaMetadata{}
	}
	c := catalog.New()
	report, err := c.LoadMetadataInto(context.Background(), meta, sdlDatabaseContext)
	if err != nil {
		return nil, nil, err
	}
	return c, report, nil
}

// unloadableChanges returns an error naming each object that either load could
// not install as defined and that differs between the two snapshots.
func unloadableChanges(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, reports ...*catalog.LoadMetadataReport) error {
	reasons := make(map[string]error)
	for _, report := range reports {
		for _, m := range []map[string]error{report.Degraded, report.Missing, report.Incomplete} {
			for key, err := range m {
				if _, seen := reasons[key]; !seen {
					reasons[key] = err
				}
			}
		}
	}
	var changed []string
	for key, err := range reasons {
		if !proto.Equal(snapshotObject(oldMeta, key), snapshotObject(newMeta, key)) {
			changed = append(changed, key+": "+err.Error())
		}
	}
	if len(changed) == 0 {
		return nil
	}
	slices.Sort(changed)
	return errors.Errorf("cannot generate a migration for objects whose definitions could not be loaded: %s", strings.Join(changed, "; "))
}

// snapshotObject finds the object a LoadMetadataReport key names, or nil.
func snapshotObject(meta *metadatapb.DatabaseSchemaMetadata, key string) proto.Message {
	kind, name, _ := strings.Cut(key, ":")
	for _, s := range meta.GetSchemas() {
		switch kind {
		case "table":
			for _, t := range s.GetTables() {
				if t.GetName() == name {
					return tableDefinition(t)
				}
			}
		case "view":
			for _, v := range s.GetViews() {
				if v.GetName() == name {
					return v
				}
			}
		case "function":
			for _, f := range s.GetFunctions() {
				if f.GetName() == name {
					return f
				}
			}
		case "procedure":
			for _, p := range s.GetProcedures() {
				if p.GetName() == name {
					return p
				}
			}
		case "trigger":
			table, trigger, _ := strings.Cut(name, ".")
			for _, t := range s.GetTables() {
				if t.GetName() != table {
					continue
				}
				for _, trg := range t.GetTriggers() {
					if trg.GetName() == trigger {
						return trg
					}
				}
			}
		case "event":
			for _, e := range s.GetEvents() {
				if e.GetName() == name {
					return e
				}
			}
		default:
		}
	}
	return nil
}

// tableDefinition is t without the statistics sync records, which change with
// the data rather than the definition.
func tableDefinition(t *metadatapb.TableMetadata) *metadatapb.TableMetadata {
	t = proto.CloneOf(t)
	t.RowCount, t.DataSize, t.IndexSize, t.DataFree = 0, 0, 0, 0
	return t
}
