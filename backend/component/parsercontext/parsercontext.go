// Package parsercontext builds the GetDatabaseMetadataFunc /
// ListDatabaseNamesFunc / GetLinkedDatabaseMetadataFunc closures the SQL
// parsers need to resolve cross-database / linked-database references.
//
// Lives outside the api/v1 package so both the api layer (e.g. sql_service,
// release_service_check) and the runner layer (approval rule evaluation,
// taskrun export executor) can use these without creating an import cycle
// — `api/v1` imports the review workflow, so the workflow can't import
// from `api/v1` in return.
package parsercontext

import (
	"context"
	"log/slog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"

	"github.com/bytebase/bytebase/backend/common"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// BuildGetDatabaseMetadataFunc returns a closure that loads a database's
// metadata (schema graph) by (instance, database) name. Used by the SQL
// parser to resolve unqualified table references to their owning database.
func BuildGetDatabaseMetadataFunc(storeInstance *store.Store) parserbase.GetDatabaseMetadataFunc {
	return func(ctx context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
		databaseMetadata, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			Workspace:    common.GetWorkspaceIDFromContext(ctx),
			InstanceID:   instanceID,
			DatabaseName: databaseName,
		})
		if err != nil {
			return "", nil, err
		}
		if databaseMetadata == nil {
			return "", nil, nil
		}
		return databaseName, databaseMetadata, nil
	}
}

// BuildListDatabaseNamesFunc returns a closure that lists all database
// names on a given instance. Used by the parser when a statement references
// a database by name without prior context.
func BuildListDatabaseNamesFunc(storeInstance *store.Store) parserbase.ListDatabaseNamesFunc {
	return func(ctx context.Context, instanceID string) ([]string, error) {
		databases, err := storeInstance.ListDatabases(ctx, &store.FindDatabaseMessage{
			Workspace:  common.GetWorkspaceIDFromContext(ctx),
			InstanceID: &instanceID,
		})
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(databases))
		for _, database := range databases {
			names = append(names, database.DatabaseName)
		}
		return names, nil
	}
}

// BuildGetLinkedDatabaseMetadataFunc returns a closure that resolves an Oracle
// database link (`schema.table@link`) to the Bytebase database it reaches:
// (instance resource ID, database name, metadata), or nils when Bytebase
// cannot identify that database. Returns nil for non-Oracle engines; the
// parser treats that as "linked references not supported here".
func BuildGetLinkedDatabaseMetadataFunc(storeInstance *store.Store, engine storepb.Engine) parserbase.GetLinkedDatabaseMetadataFunc {
	if engine != storepb.Engine_ORACLE {
		return nil
	}
	return func(ctx context.Context, instanceID string, linkName string, schemaName string) (string, string, *model.DatabaseMetadata, error) {
		resolution, err := resolveOracleLink(ctx, storeInstance, common.GetWorkspaceIDFromContext(ctx), engine, instanceID, linkName, schemaName)
		if err != nil {
			return "", "", nil, err
		}
		if resolution.Meta == nil {
			slog.Debug("database link not resolved", slog.String("instance", instanceID), slog.String("link", linkName), slog.String("reason", resolution.Reason))
			return "", "", nil, nil
		}
		return resolution.InstanceID, resolution.DatabaseName, resolution.Meta, nil
	}
}
