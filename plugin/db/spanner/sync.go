package spanner

import (
	"context"

	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"google.golang.org/api/iterator"

	"github.com/bytebase/bytebase/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMeta, error) {
	var databaseList []db.DatabaseMeta
	iter := d.dbClient.ListDatabases(ctx, &databasepb.ListDatabasesRequest{
		Parent: d.config.Host,
	})
	for {
		database, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		databaseList = append(databaseList, db.DatabaseMeta{Name: database.Name})
	}

	return &db.InstanceMeta{
		DatabaseList: databaseList,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context, _ string) (*db.Schema, map[string][]*storepb.ForeignKeyMetadata, error) {
	panic("not implemented")
}
