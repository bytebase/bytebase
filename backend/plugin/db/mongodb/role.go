package mongodb

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// bytebaseDefaultDatabase is the default database name for bytebase.
	bytebaseDefaultDatabase = "bytebase"
)

// getUserList returns the list of users.
func (d *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRole, error) {
	database := d.client.Database(bytebaseDefaultDatabase)
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
			slog.Info("Skip getting user list because the user is not authorized to run the command 'usersInfo' in atlas cluster M0/M2/M5")
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot run usersInfo command")
	}
	var instanceRoles []*storepb.InstanceRole
	for _, user := range commandResult.Users {
		instanceRoles = append(instanceRoles, &storepb.InstanceRole{
			Name: user.ID,
		})
	}
	return instanceRoles, nil
}
