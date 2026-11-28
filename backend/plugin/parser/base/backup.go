package base

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type TransformContext struct {
	Version                 any
	InstanceID              string
	GetDatabaseMetadataFunc GetDatabaseMetadataFunc
	ListDatabaseNamesFunc   ListDatabaseNamesFunc
	// ListReferencingDatabasesFunc lists the databases that may hold a foreign
	// key into another database. MySQL-family prior backup reads only these
	// for keys that cascade into its tables; nil makes it read every database.
	ListReferencingDatabasesFunc ListDatabaseNamesFunc
	IsCaseSensitive              bool
	DatabaseName                 string
}

// ReferencingDatabases is ListReferencingDatabasesFunc as omni's
// BackupRequest.ReferencingDatabases takes it, listed once for every table a
// backup checks. The list does not depend on the database asked about.
func (c TransformContext) ReferencingDatabases(ctx context.Context) func(string) ([]string, error) {
	if c.ListReferencingDatabasesFunc == nil {
		return nil
	}
	var names []string
	var err error
	listed := false
	return func(string) ([]string, error) {
		if !listed {
			names, err = c.ListReferencingDatabasesFunc(ctx, c.InstanceID)
			listed = true
		}
		return names, err
	}
}

type BackupStatement struct {
	Statement       string
	SourceSchema    string
	SourceTableName string
	TargetTableName string

	StartPosition *storepb.Position
	EndPosition   *storepb.Position
}
