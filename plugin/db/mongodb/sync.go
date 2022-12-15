package mongodb

import (
	"context"
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/db"
)

// UserInfo is the subset of the mongodb command result of "usersInfo".
type UsersInfo struct {
	Users []User `bson:"users"`
}

// User is the subset of the `users` field in the `User`..
type User struct {
	Id       string `json:"_id" bson:"_id"`
	UserName string `json:"user" bson:"user"`
	DB       string `json:"db" bson:"db"`
	Roles    []Role `json:"roles" bson:"roles"`
}

// Role is the subset of the `roles` field in the `User`.
type Role struct {
	RoleName string `json:"role" bson:"role"`
	DB       string `json:"db" bson:"db"`
}

// SyncInstance syncs the instance meta.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMeta, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}
	userList, err := driver.getUserMetaList(ctx)
	if err != nil {
		return nil, err
	}
	var databaseMetaList []db.DatabaseMeta
	dbList, err := driver.getDatabaseList(ctx)
	for _, databaseName := range dbList {
		databaseMetaList = append(databaseMetaList, db.DatabaseMeta{
			Name: databaseName,
		})
	}

	return &db.InstanceMeta{
		Version:      version,
		UserList:     userList,
		DatabaseList: databaseMetaList,
	}, nil
}

// SyncDBSchema syncs the database schema.
func (*Driver) SyncDBSchema(_ context.Context, _ string) (*db.Schema, error) {
	panic("not implemented")
}

// getVersion returns the version of mongod or mongos instance.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	database := driver.client.Database(migrationHistoryDefaultDatabase)
	var commandResult bson.M
	command := bson.D{{Key: "buildInfo", Value: 1}}
	if err := database.RunCommand(ctx, command).Decode(&commandResult); err != nil {
		return "", errors.Wrap(err, "cannot run buildInfo command")
	}
	version, ok := commandResult["version"]
	if !ok {
		return "", errors.New("cannot get version from buildInfo command result")
	}
	return version.(string), nil
}

// getUserList returns the list of users.
func (driver *Driver) getUserMetaList(ctx context.Context) ([]db.User, error) {
	database := driver.client.Database(migrationHistoryDefaultDatabase)
	command := bson.D{{
		Key: "usersInfo",
		Value: bson.D{{
			Key:   "forAllDBs",
			Value: true,
		}},
	}}
	var commandResult UsersInfo
	if err := database.RunCommand(ctx, command).Decode(&commandResult); err != nil {
		return nil, errors.Wrap(err, "cannot run usersInfo command")
	}
	var dbUserList []db.User
	for _, user := range commandResult.Users {
		var dbUser db.User
		dbUser.Name = user.UserName
		bs, err := json.Marshal(user)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal user")
		}
		dbUser.Grant = string(bs)
		dbUserList = append(dbUserList, dbUser)
	}
	return dbUserList, nil
}

// getDatabaseList returns the list of databases.
func (driver *Driver) getDatabaseList(ctx context.Context) ([]string, error) {
	return driver.client.ListDatabaseNames(ctx, bson.M{})
}
