// Package trino implements the bytebase schema.GetDatabaseDefinition contract
// for the Trino engine, backed by the omni Trino deparser
// (github.com/bytebase/omni/trino/deparse).
package trino

import (
	"github.com/bytebase/omni/trino/deparse"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_TRINO, GetDatabaseDefinition)
}

// GetDatabaseDefinition renders the CREATE TABLE SDL for a Trino database
// schema. It converts the storepb metadata into the omni deparser's metadata
// model and delegates to deparse.GetDatabaseDefinition, which emits one CREATE
// TABLE statement per table in schema-then-table snapshot order.
func GetDatabaseDefinition(_ schema.GetDefinitionContext, metadata *storepb.DatabaseSchemaMetadata) (string, error) {
	return deparse.GetDatabaseDefinition(convertDatabaseMetadata(metadata))
}

// convertDatabaseMetadata maps storepb.DatabaseSchemaMetadata onto the omni
// deparse metadata model one-to-one, copying only the fields the Trino SDL dump
// consumes (schemas, tables, and column Name/Type/Nullable).
func convertDatabaseMetadata(metadata *storepb.DatabaseSchemaMetadata) *deparse.DatabaseSchemaMetadata {
	if metadata == nil {
		return nil
	}
	out := &deparse.DatabaseSchemaMetadata{
		Name:    metadata.Name,
		Schemas: make([]*deparse.SchemaMetadata, 0, len(metadata.Schemas)),
	}
	for _, s := range metadata.Schemas {
		if s == nil {
			continue
		}
		schemaMeta := &deparse.SchemaMetadata{
			Name:   s.Name,
			Tables: make([]*deparse.TableMetadata, 0, len(s.Tables)),
		}
		for _, t := range s.Tables {
			if t == nil {
				continue
			}
			tableMeta := &deparse.TableMetadata{
				Name:    t.Name,
				Columns: make([]*deparse.ColumnMetadata, 0, len(t.Columns)),
			}
			for _, c := range t.Columns {
				if c == nil {
					continue
				}
				tableMeta.Columns = append(tableMeta.Columns, &deparse.ColumnMetadata{
					Name:     c.Name,
					Type:     c.Type,
					Nullable: c.Nullable,
				})
			}
			schemaMeta.Tables = append(schemaMeta.Tables, tableMeta)
		}
		out.Schemas = append(out.Schemas, schemaMeta)
	}
	return out
}
