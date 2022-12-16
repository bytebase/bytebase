package mongodb

import (
	"context"
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/db"
)

// UsersInfo is the subset of the mongodb command result of "usersInfo".
type UsersInfo struct {
	Users []User `bson:"users"`
}

// User is the subset of the `users` field in the `User`.
type User struct {
	ID       string `json:"_id" bson:"_id"`
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
	dbList, err := driver.getNonSystemDatabaseList(ctx)
	if err != nil {
		return nil, err
	}
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
func (driver *Driver) SyncDBSchema(ctx context.Context, databaseName string) (*db.Schema, error) {
	exist, err := driver.isDatabaseExist(ctx, databaseName)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.Errorf("database %s does not exist", databaseName)
	}
	schema := db.Schema{}
	schema.Name = databaseName

	tableList, err := driver.syncAllCollectionSchema(ctx, databaseName)
	if err != nil {
		return nil, err
	}
	schema.TableList = tableList

	// TODO(zp): sync View schema
	return &schema, nil
}

func (driver *Driver) syncAllCollectionSchema(ctx context.Context, databaseName string) ([]db.Table, error) {
	database := driver.client.Database(databaseName)
	collectionFilter := bson.M{"type": collectionType}
	collectionList, err := database.ListCollectionNames(ctx, collectionFilter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list collection names")
	}
	var tableList []db.Table
	for _, collectionName := range collectionList {
		table, err := driver.syncCollectionSchema(ctx, databaseName, collectionName)
		if err != nil {
			return nil, err
		}
		tableList = append(tableList, table)
	}
	return tableList, nil
}

// syncCollectionSchema returns the collection schema.
func (driver *Driver) syncCollectionSchema(ctx context.Context, databaseName string, collectionName string) (db.Table, error) {
	var table db.Table
	table.Name = collectionName
	table.ShortName = collectionName
	table.Type = collectionType

	// Get estimated document count.
	database := driver.client.Database(databaseName)
	collection := database.Collection(collectionName)
	count, err := collection.EstimatedDocumentCount(ctx)
	if err != nil {
		return table, errors.Wrap(err, "failed to get estimated document count")
	}
	table.RowCount = count

	// Get collection data size and total index size in byte.
	dataSize, totalIndexSize, err := driver.getCollectionDataSizeAndIndexSizeInByte(ctx, databaseName, collectionName)
	if err != nil {
		return table, errors.Wrapf(err, "failed to get collection size and index size in byte of collection %s", collectionName)
	}
	table.DataSize = int64(dataSize)
	table.IndexSize = int64(totalIndexSize)

	// Get collection index schema.
	indexList, err := driver.syncAllIndexSchema(ctx, databaseName, collectionName)
	if err != nil {
		return table, errors.Wrapf(err, "failed to get index schema of collection %s", collectionName)
	}
	table.IndexList = indexList

	// TODO(zp): sync Column schema

	return table, nil
}

// syncAllIndexSchema returns all indexes schema of a collection.
func (driver *Driver) syncAllIndexSchema(ctx context.Context, databaseName, collectionName string) ([]db.Index, error) {
	database := driver.client.Database(databaseName)
	collection := database.Collection(collectionName)
	indexCursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list indexes")
	}

	var indexList []db.Index
	for indexCursor.Next(ctx) {
		var indexInfo bson.M
		if err := indexCursor.Decode(&indexInfo); err != nil {
			return nil, errors.Wrap(err, "failed to decode index info")
		}
		index, err := getIndexSchema(ctx, indexInfo)
		if err != nil {
			return nil, err
		}
		indexList = append(indexList, index)
	}

	return indexList, nil
}

// getIndexSchema returns the index schema.
// https://www.mongodb.com/docs/manual/reference/command/listIndexes/#output
func getIndexSchema(ctx context.Context, indexInfo bson.M) (db.Index, error) {
	var index db.Index
	indexName, ok := indexInfo["name"]
	if !ok {
		return index, errors.New("cannot get index name from index info")
	}
	index.Name = indexName.(string)

	key, ok := indexInfo["key"]
	if !ok {
		return index, errors.New("cannot get index key from index info")
	}
	keystr, err := json.Marshal(key)
	if err != nil {
		return index, errors.Wrap(err, "cannot marshal index key to json")
	}
	index.Expression = string(keystr)

	unique, ok := indexInfo["unique"]
	if !ok {
		// If the unique field is not set, the index is not unique.
		unique = false
	}
	index.Unique = unique.(bool)

	return index, nil
}

// getCollectionDataSizeAndIndexSizeInByte returns the collection data size and total index size in bytes.
func (driver *Driver) getCollectionDataSizeAndIndexSizeInByte(ctx context.Context, databaseName string, collectionName string) (int32, int32, error) {
	database := driver.client.Database(databaseName)
	command := bson.D{{
		Key:   "collStats",
		Value: collectionName,
	}}
	var commandResult bson.M
	if err := database.RunCommand(ctx, command).Decode(&commandResult); err != nil {
		return 0, 0, errors.Wrap(err, "cannot run collStats command")
	}
	size, ok := commandResult["size"]
	if !ok {
		return 0, 0, errors.New("cannot get size from collStats command result")
	}

	totalIndexSize, ok := commandResult["totalIndexSize"]
	if !ok {
		return 0, 0, errors.New("cannot get totalIndexSize from collStats command result")
	}
	return size.(int32), totalIndexSize.(int32), nil
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

// isDatabaseExist returns true if the database exists.
func (driver *Driver) isDatabaseExist(ctx context.Context, databaseName string) (bool, error) {
	filter := bson.M{"name": databaseName}
	databaseList, err := driver.client.ListDatabaseNames(ctx, filter)
	if err != nil {
		return false, errors.Wrap(err, "failed to list database names")
	}
	return len(databaseList) == 1, nil
}

// getNonSystemDatabaseList returns the list of non system databases.
func (driver *Driver) getNonSystemDatabaseList(ctx context.Context) ([]string, error) {
	filter := bson.M{
		"name": bson.M{
			"$nin": []string{"admin", "config", "local", "bytebase"},
		},
	}
	return driver.client.ListDatabaseNames(ctx, filter)
}
