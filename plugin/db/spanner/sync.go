package spanner

import (
	"context"

	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"google.golang.org/api/iterator"

	"github.com/pkg/errors"

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

		match := dsnRegExp.FindStringSubmatch(database.Name)
		matches := make(map[string]string)
		for i, name := range dsnRegExp.SubexpNames() {
			if i != 0 && name != "" {
				matches[name] = match[i]
			}
		}
		databaseName := matches["DATABASEGROUP"]
		if databaseName == "" {
			return nil, errors.Errorf("failed to parse database name from %s", database.Name)
		}
		if excludedDatabaseList[databaseName] {
			continue
		}

		databaseList = append(databaseList, db.DatabaseMeta{Name: databaseName})
	}

	return &db.InstanceMeta{
		DatabaseList: databaseList,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context, _ string) (*db.Schema, map[string][]*storepb.ForeignKeyMetadata, error) {
	return nil, nil, errors.New("not implemented")
}
