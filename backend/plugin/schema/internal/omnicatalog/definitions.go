package omnicatalog

import (
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// MySQLFamilyDefinitions holds the per-object definition functions exposed by a
// MySQL-family omni catalog (mysql, oceanbase, tidb, mariadb).
type MySQLFamilyDefinitions struct {
	Database  func(schema.GetDefinitionContext, *metadatapb.DatabaseSchemaMetadata) (string, error)
	Table     func(*metadatapb.TableMetadata) (string, error)
	View      func(*metadatapb.ViewMetadata) (string, error)
	Function  func(*metadatapb.FunctionMetadata) (string, error)
	Procedure func(*metadatapb.ProcedureMetadata) (string, error)
	Sequence  func(*metadatapb.SequenceMetadata) (string, error)
}

// RegisterMySQLFamilyDefinitions registers a MySQL-family engine's definition
// renderers, adapting single-schema omni signatures to schema's registry.
func RegisterMySQLFamilyDefinitions(engine storepb.Engine, defs MySQLFamilyDefinitions) {
	schema.Register(schema.Engine{
		Definitions: schema.Definitions{
			Database:  defs.Database,
			Table:     ignoreTableScope(defs.Table),
			View:      ignoreObjectScope(defs.View),
			Function:  ignoreObjectScope(defs.Function),
			Procedure: ignoreObjectScope(defs.Procedure),
			Sequence:  ignoreObjectScope(defs.Sequence),
		},
	}, engine)
}

// ignoreTableScope adapts a table definition renderer that takes only *TableMetadata
// to schema's (schemaName, table, sequences) signature.
func ignoreTableScope(fn func(*metadatapb.TableMetadata) (string, error)) func(string, *metadatapb.TableMetadata, []*metadatapb.SequenceMetadata) (string, error) {
	if fn == nil {
		return nil
	}
	return func(_ string, table *metadatapb.TableMetadata, _ []*metadatapb.SequenceMetadata) (string, error) {
		return fn(table)
	}
}

// ignoreObjectScope adapts a per-object definition renderer that ignores schemaName.
func ignoreObjectScope[T any](fn func(T) (string, error)) func(string, T) (string, error) {
	if fn == nil {
		return nil
	}
	return func(_ string, obj T) (string, error) {
		return fn(obj)
	}
}
