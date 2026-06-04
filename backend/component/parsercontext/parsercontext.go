// Package parsercontext builds the GetDatabaseMetadataFunc /
// ListDatabaseNamesFunc / GetLinkedDatabaseMetadataFunc closures the SQL
// parsers need to resolve cross-database / linked-database references.
//
// Lives outside the api/v1 package so both the api layer (e.g. sql_service,
// release_service_check) and the runner layer (approval rule evaluation,
// taskrun export executor) can use these without creating an import cycle
// — `api/v1` imports `runner/approval`, so `runner/approval` can't import
// from `api/v1` in return.
package parsercontext

import (
	"context"
	"strings"

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

// BuildGetLinkedDatabaseMetadataFunc returns a closure that resolves Oracle
// linked-database references (DBLINK syntax) to the actual remote database
// metadata. Returns nil for non-Oracle engines — the parser interprets that
// as "linked references not supported here".
func BuildGetLinkedDatabaseMetadataFunc(storeInstance *store.Store, engine storepb.Engine) parserbase.GetLinkedDatabaseMetadataFunc {
	if engine != storepb.Engine_ORACLE {
		return nil
	}
	return func(ctx context.Context, instanceID string, linkedDatabaseName string, schemaName string) (string, string, *model.DatabaseMetadata, error) {
		// Find the linked database metadata.
		databases, err := storeInstance.ListDatabases(ctx, &store.FindDatabaseMessage{
			Workspace:  common.GetWorkspaceIDFromContext(ctx),
			InstanceID: &instanceID,
		})
		if err != nil {
			return "", "", nil, err
		}
		var linkedMeta *storepb.LinkedDatabaseMetadata
		for _, database := range databases {
			meta, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
				Workspace:    common.GetWorkspaceIDFromContext(ctx),
				InstanceID:   database.InstanceID,
				DatabaseName: database.DatabaseName,
			})
			if err != nil {
				return "", "", nil, err
			}
			if linkedMeta = meta.GetLinkedDatabase(linkedDatabaseName); linkedMeta != nil {
				break
			}
		}
		if linkedMeta == nil {
			return "", "", nil, nil
		}
		// Find the linked database in Bytebase.
		var linkedDatabase *store.DatabaseMessage
		databaseName := linkedMeta.GetUsername()
		if schemaName != "" {
			databaseName = schemaName
		}
		databaseList, err := storeInstance.ListDatabases(ctx, &store.FindDatabaseMessage{
			Workspace:    common.GetWorkspaceIDFromContext(ctx),
			DatabaseName: &databaseName,
			Engine:       &engine,
		})
		if err != nil {
			return "", "", nil, err
		}
		for _, database := range databaseList {
			instance, err := storeInstance.GetInstance(ctx, &store.FindInstanceMessage{Workspace: common.GetWorkspaceIDFromContext(ctx), ResourceID: &database.InstanceID})
			if err != nil {
				return "", "", nil, err
			}
			if instance != nil {
				for _, dataSource := range instance.Metadata.DataSources {
					if strings.Contains(linkedMeta.GetHost(), dataSource.GetHost()) {
						linkedDatabase = database
						break
					}
				}
				if linkedDatabase != nil {
					break
				}
			}
		}
		if linkedDatabase == nil {
			return "", "", nil, nil
		}
		// Get the linked database metadata.
		linkedDatabaseMetadata, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			Workspace:    common.GetWorkspaceIDFromContext(ctx),
			InstanceID:   linkedDatabase.InstanceID,
			DatabaseName: linkedDatabase.DatabaseName,
		})
		if err != nil {
			return "", "", nil, err
		}
		if linkedDatabaseMetadata == nil {
			return "", "", nil, nil
		}
		return linkedDatabase.InstanceID, linkedDatabaseName, linkedDatabaseMetadata, nil
	}
}
