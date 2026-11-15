package pg

import (
	"context"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/store/model"
)

// omniDiffMetadataMigration generates the migration between two synced snapshots
// with omni's catalog.
//
// An object omni cannot install as the snapshot defines it (a stand-in, or one
// left out) is safe to leave in place only when it is the same on both sides: the
// two loads then agree on it and the diff skips it. Where it differs, the diff would
// compare stand-ins, so the migration is refused rather than guessed.
func omniDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	from, fromReport, err := loadMetadataCatalog(oldSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load source schema")
	}
	to, toReport, err := loadMetadataCatalog(newSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load target schema")
	}
	if err := unloadableChanges(oldSchema.GetProto(), newSchema.GetProto(), fromReport, toReport); err != nil {
		return "", err
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}
	plan := catalog.GenerateMigration(from, to, diff).Filter(func(op catalog.MigrationOp) bool {
		return op.SchemaName != "bbdataarchive"
	})
	return plan.SQL(), nil
}

// loadMetadataCatalog installs a snapshot under the search path its database is
// configured with, which its view bodies resolve against.
func loadMetadataCatalog(d *model.DatabaseMetadata) (*catalog.Catalog, *catalog.LoadMetadataReport, error) {
	c := catalog.New()
	meta := &metadatapb.DatabaseSchemaMetadata{}
	if d != nil {
		if searchPath := getConfiguredSearchPath(d); len(searchPath) > 0 {
			c.SetSearchPath(searchPath)
		}
		if p := d.GetProto(); p != nil {
			meta = p
		}
	}
	report, err := c.LoadMetadata(context.Background(), meta, catalog.LoadMetadataOptions{Full: true})
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
		for _, m := range []map[string]error{report.Degraded, report.Missing} {
			for key, err := range m {
				if _, seen := reasons[key]; !seen {
					reasons[key] = err
				}
			}
		}
	}
	if len(reasons) == 0 {
		return nil
	}
	oldObjects, newObjects := snapshotObjects(oldMeta), snapshotObjects(newMeta)
	var changed []string
	for key, err := range reasons {
		if !proto.Equal(oldObjects[key], newObjects[key]) {
			changed = append(changed, key+": "+err.Error())
		}
	}
	if len(changed) == 0 {
		return nil
	}
	slices.Sort(changed)
	return errors.Errorf("cannot generate a migration for objects whose definitions could not be loaded: %s", strings.Join(changed, "; "))
}

// snapshotObjects indexes a snapshot's objects by the keys LoadMetadataReport
// uses. A constraint key maps to its table, which holds the constraint.
func snapshotObjects(meta *metadatapb.DatabaseSchemaMetadata) map[string]proto.Message {
	objects := make(map[string]proto.Message)
	for _, ext := range meta.GetExtensions() {
		objects["ext:"+ext.GetName()] = ext
	}
	for _, s := range meta.GetSchemas() {
		schema := s.GetName()
		objects["schema:"+schema] = &metadatapb.SchemaMetadata{Name: schema, Owner: s.GetOwner(), Comment: s.GetComment()}
		qualified := func(name string) string { return schema + "." + name }
		for _, e := range s.GetEnumTypes() {
			objects["type:"+qualified(e.GetName())] = e
		}
		for _, ct := range s.GetCompositeTypes() {
			objects["type:"+qualified(ct.GetName())] = ct
		}
		for _, seq := range s.GetSequences() {
			objects["seq:"+qualified(seq.GetName())] = seq
		}
		for _, fn := range s.GetFunctions() {
			identity := fn.GetSignature()
			if identity == "" {
				identity = fn.GetDefinition()
			}
			objects["func:"+qualified(fn.GetName())+"|"+identity] = fn
		}
		for _, v := range s.GetViews() {
			objects["rel:"+qualified(v.GetName())] = v
		}
		for _, mv := range s.GetMaterializedViews() {
			objects["rel:"+qualified(mv.GetName())] = mv
			for _, idx := range mv.GetIndexes() {
				objects["idx:"+qualified(mv.GetName())+"."+idx.GetName()] = idx
			}
		}
		for _, t := range s.GetTables() {
			def := tableDefinition(t)
			objects["rel:"+qualified(t.GetName())] = def
			addTableParts(objects, qualified(t.GetName()), def)
			forEachPartition(t.GetPartitions(), func(p *metadatapb.TablePartitionMetadata) {
				objects["rel:"+qualified(p.GetName())] = p
				for _, idx := range p.GetIndexes() {
					objects["idx:"+qualified(p.GetName())+"."+idx.GetName()] = idx
				}
			})
		}
	}
	return objects
}

// addTableParts indexes a table's indexes, and maps each of its constraints to the
// table itself.
func addTableParts(objects map[string]proto.Message, table string, t *metadatapb.TableMetadata) {
	for _, idx := range t.GetIndexes() {
		objects["idx:"+table+"."+idx.GetName()] = idx
		objects["con:"+table+"."+idx.GetName()] = t
	}
	for _, c := range t.GetCheckConstraints() {
		objects["con:"+table+"."+c.GetName()] = t
	}
	for _, c := range t.GetExcludeConstraints() {
		objects["con:"+table+"."+c.GetName()] = t
	}
	for _, fk := range t.GetForeignKeys() {
		objects["con:"+table+"."+fk.GetName()] = t
	}
}

func forEachPartition(partitions []*metadatapb.TablePartitionMetadata, f func(*metadatapb.TablePartitionMetadata)) {
	for _, p := range partitions {
		f(p)
		forEachPartition(p.GetSubpartitions(), f)
	}
}

// tableDefinition is t without the statistics sync records, which change with
// the data rather than the definition.
func tableDefinition(t *metadatapb.TableMetadata) *metadatapb.TableMetadata {
	t = proto.CloneOf(t)
	t.RowCount, t.DataSize, t.IndexSize, t.DataFree = 0, 0, 0, 0
	return t
}
