// Package pg implements Bytebase's schema contracts for PostgreSQL and
// CockroachDB on top of omni's pg catalog. The catalog owns the database,
// multi-file and per-object definitions, the SDL diff and the metadata diff.
// Bytebase keeps the registry wiring, the backup-schema filter, the legacy
// index-expression repair in legacy_normalize.go, and the DDL walk-through.
//
// CockroachDB shares the catalog: its synced metadata is PostgreSQL-shaped and
// omni has no CockroachDB catalog. testdata/cutover_parity.yaml records the
// migrations the deleted Bytebase generator produced, so every text change the
// catalog introduces stays named.
package pg

import (
	"strings"

	"google.golang.org/protobuf/proto"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB} {
		schema.RegisterGetDatabaseDefinition(engine, GetDatabaseDefinition)
		schema.RegisterGetSchemaDefinition(engine, catalog.SchemaDefinition)
		schema.RegisterGetTableDefinition(engine, catalog.TableDefinition)
		schema.RegisterGetViewDefinition(engine, catalog.ViewDefinition)
		schema.RegisterGetMaterializedViewDefinition(engine, catalog.MaterializedViewDefinition)
		schema.RegisterGetFunctionDefinition(engine, catalog.FunctionDefinition)
		schema.RegisterGetSequenceDefinition(engine, catalog.SequenceDefinition)
		schema.RegisterGetMultiFileDatabaseDefinition(engine, GetMultiFileDatabaseDefinition)
	}
}

// GetDatabaseDefinition renders metadata through omni's catalog.DatabaseDefinition
// after dropping the backup schema when asked and repairing legacy metadata.
func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	return catalog.DatabaseDefinition(prepareDefinitionMetadata(ctx, metadata), catalog.DefinitionOptions{
		PrintHeader: ctx.PrintHeader,
		SDLFormat:   ctx.SDLFormat,
	})
}

// GetMultiFileDatabaseDefinition renders metadata through omni's
// catalog.MultiFileDefinition, prepared as GetDatabaseDefinition prepares it.
func GetMultiFileDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (*schema.MultiFileSchemaResult, error) {
	files, err := catalog.MultiFileDefinition(prepareDefinitionMetadata(ctx, metadata))
	if err != nil {
		return nil, err
	}
	result := &schema.MultiFileSchemaResult{}
	if files != nil {
		result.Files = make([]schema.File, 0, len(files))
		for _, file := range files {
			result.Files = append(result.Files, schema.File{Name: file.Name, Content: file.Content})
		}
	}
	return result, nil
}

func prepareDefinitionMetadata(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	metadata = filterBackupSchemaIfNecessary(ctx, metadata)

	// Clone before mutating: the caller's *DatabaseSchemaMetadata is often a
	// shared pointer returned from store.dbSchemaCache (see
	// backend/store/model/database.go:GetProto), so concurrent callers could
	// race on in-place normalization writes below.
	metadata = proto.CloneOf(metadata)

	// Repair historical non-canonical metadata shapes (e.g. index key
	// expressions stored without outer parens) so emission produces valid SQL.
	// See legacy_normalize.go for removal criteria.
	normalizeLegacyMetadata(metadata)
	return metadata
}

func filterBackupSchemaIfNecessary(ctx schema.GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) *metadatapb.DatabaseSchemaMetadata {
	if !ctx.SkipBackupSchema {
		return metadata
	}

	filtered := &metadatapb.DatabaseSchemaMetadata{
		Extensions:    metadata.Extensions,
		EventTriggers: metadata.EventTriggers,
	}
	for _, schema := range metadata.Schemas {
		if schema.Name == common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES) {
			continue
		}
		filtered.Schemas = append(filtered.Schemas, schema)
	}
	return filtered
}

// parseQualifiedTypeIdent splits a type reference produced by the sync layer
// (format_type output, schema-qualified for user-defined types) into
// (schema, name), ignoring any typmod or array suffix. It returns ok=false
// for unqualified references — built-in types are never schema-qualified.
func parseQualifiedTypeIdent(s string) (string, string, bool) {
	var parts []string
	for i := 0; i < len(s); {
		if s[i] == '"' {
			var b strings.Builder
			i++
			for i < len(s) {
				if s[i] == '"' {
					if i+1 < len(s) && s[i+1] == '"' {
						b.WriteByte('"')
						i += 2
						continue
					}
					i++
					break
				}
				b.WriteByte(s[i])
				i++
			}
			parts = append(parts, b.String())
		} else {
			start := i
			for i < len(s) && s[i] != '.' && s[i] != '(' && s[i] != '[' {
				i++
			}
			parts = append(parts, strings.TrimSpace(s[start:i]))
		}
		if i < len(s) && s[i] == '.' {
			i++
			continue
		}
		break
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], true
	}
	return "", "", false
}
