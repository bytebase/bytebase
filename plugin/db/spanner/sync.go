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

		// database.Name is of the form `projects/<project>/instances/<instance>/databases/<database>`
		// We use regular expression to extract <database> from it.
		databaseName, err := getDatabaseFromDSN(database.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database name from %s", database.Name)
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
func (*Driver) SyncDBSchema(_ context.Context, _ string) (*storepb.DatabaseMetadata, error) {
	return nil, errors.New("not implemented")
}
