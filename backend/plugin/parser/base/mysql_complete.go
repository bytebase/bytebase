package base

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"
)

// MySQLFamilyDatabaseLoaders adapts CompletionContext's Metadata and
// ListDatabaseNames callbacks into the plain callbacks expected by omni's
// MySQL-family session completion APIs.
func MySQLFamilyDatabaseLoaders(cCtx CompletionContext) (
	loadDatabase func(ctx context.Context, name string) (resolvedName string, meta *metadatapb.DatabaseSchemaMetadata, caseSensitive bool, ok bool),
	listDatabaseNames func(ctx context.Context) ([]string, error),
) {
	if cCtx.Metadata != nil {
		loadDatabase = func(ctx context.Context, name string) (string, *metadatapb.DatabaseSchemaMetadata, bool, bool) {
			resolvedName, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, name)
			if err != nil || dbMeta == nil {
				return "", nil, false, false
			}
			return resolvedName, dbMeta.GetProto(), dbMeta.GetIsObjectCaseSensitive(), true
		}
	}
	if cCtx.ListDatabaseNames != nil {
		listDatabaseNames = func(ctx context.Context) ([]string, error) {
			return cCtx.ListDatabaseNames(ctx, cCtx.InstanceID)
		}
	}
	return loadDatabase, listDatabaseNames
}
