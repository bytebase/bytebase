package mongodb

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// bytebaseDefaultDatabase is the default database name for bytebase.
	bytebaseDefaultDatabase = "bytebase"
)

// CreateRole creates the role.
func (*Driver) CreateRole(_ context.Context, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("create role for MongoDB is not implemented yet")
}

// UpdateRole updates the role.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("update role for MongoDB is not implemented yet")
}

// FindRole finds the role by name.
func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("find role for MongoDB is not implemented yet")
}

// ListRole lists the role.
func (*Driver) ListRole(_ context.Context) ([]*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("list role for MongoDB is not implemented yet")
}

// DeleteRole deletes the role by name.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	return errors.Errorf("delete role for MongoDB is not implemented yet")
}

// getUserList returns the list of users.
func (driver *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRoleMetadata, error) {
	database := driver.client.Database(bytebaseDefaultDatabase)
	command := bson.D{{
		Key: "usersInfo",
		Value: bson.D{{
			Key:   "forAllDBs",
			Value: true,
		}},
	}}
	var commandResult UsersInfo
	if err := database.RunCommand(ctx, command).Decode(&commandResult); err != nil {
		if isAtlasUnauthorizedError(err) {
			log.Info("Skip getting user list because the user is not authorized to run the command 'usersInfo' in atlas cluster M0/M2/M5")
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot run usersInfo command")
	}
	var instanceRoles []*storepb.InstanceRoleMetadata
	for _, user := range commandResult.Users {
		bs, err := json.Marshal(user)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal user")
		}

		instanceRoles = append(instanceRoles, &storepb.InstanceRoleMetadata{
			Name:  user.ID,
			Grant: string(bs),
		})
	}
	return instanceRoles, nil
}
