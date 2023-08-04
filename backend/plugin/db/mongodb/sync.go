package mongodb

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var systemCollection = map[string]bool{
	"system.namespaces": true,
	"system.indexes":    true,
	"system.profile":    true,
	"system.js":         true,
	"system.views":      true,
}

var systemDatabase = map[string]bool{
	"admin":    true,
	"config":   true,
	"local":    true,
	"bytebase": true,
}

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
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}
	users, err := driver.getInstanceRoles(ctx)
	if err != nil {
		return nil, err
	}
	var databases []*storepb.DatabaseSchemaMetadata
	databaseNames, err := driver.getNonSystemDatabaseList(ctx)
	if err != nil {
		return nil, err
	}
	for _, databaseName := range databaseNames {
		databases = append(databases, &storepb.DatabaseSchemaMetadata{
			Name: databaseName,
		})
	}

	return &db.InstanceMetadata{
		Version:       version,
		InstanceRoles: users,
		Databases:     databases,
	}, nil
}

// SyncDBSchema syncs the database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{
		Name: "",
	}

	exist, err := driver.isDatabaseExist(ctx, driver.databaseName)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.Errorf("database %s does not exist", driver.databaseName)
	}

	database := driver.client.Database(driver.databaseName)
	collectionList, err := database.ListCollectionNames(ctx, bson.M{"type": "collection"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list collection names")
	}
	sort.Strings(collectionList)

	for _, collectionName := range collectionList {
		if systemCollection[collectionName] {
			continue
		}

		collection := database.Collection(collectionName)
		count, err := collection.EstimatedDocumentCount(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get estimated document count")
		}
		// Get collection data size and total index size in byte.
		var commandResult bson.M
		if err := database.RunCommand(ctx, bson.D{{
			Key:   "collStats",
			Value: collectionName,
		}}).Decode(&commandResult); err != nil {
			return nil, errors.Wrap(err, "cannot run collStats command")
		}
		dataSize, ok := commandResult["size"]
		if !ok {
			return nil, errors.New("cannot get size from collStats command result")
		}
		dataSize64, err := convertEmptyInterfaceToInt64(dataSize)
		if err != nil {
			log.Debug("Failed to convert dataSize to int64", zap.Any("dataSize", dataSize))
		}

		totalIndexSize, ok := commandResult["totalIndexSize"]
		if !ok {
			return nil, errors.New("cannot get totalIndexSize from collStats command result")
		}
		totalIndexSize64, err := convertEmptyInterfaceToInt64(totalIndexSize)
		if err != nil {
			log.Debug("Failed to convert totalIndexSize to int64", zap.Any("totalIndexSize", totalIndexSize))
		}

		// Get collection indexes.
		indexes, err := getIndexes(ctx, collection)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get index schema of collection %s", collectionName)
		}
		schemaMetadata.Tables = append(schemaMetadata.Tables, &storepb.TableMetadata{
			Name:      collectionName,
			RowCount:  count,
			DataSize:  dataSize64,
			IndexSize: totalIndexSize64,
			Indexes:   indexes,
		})
	}

	viewList, err := database.ListCollectionNames(ctx, bson.M{"type": "view"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list view names")
	}
	for _, viewName := range viewList {
		schemaMetadata.Views = append(schemaMetadata.Views, &storepb.ViewMetadata{Name: viewName})
	}

	return &storepb.DatabaseSchemaMetadata{
		Name:    driver.databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}, nil
}

// getIndexes returns all indexes schema of a collection.
// https://www.mongodb.com/docs/manual/reference/command/listIndexes/#output
func getIndexes(ctx context.Context, collection *mongo.Collection) ([]*storepb.IndexMetadata, error) {
	indexCursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list indexes")
	}
	indexMap := make(map[string]*storepb.IndexMetadata)
	defer indexCursor.Close(ctx)
	for indexCursor.Next(ctx) {
		var indexInfo bson.M
		if err := indexCursor.Decode(&indexInfo); err != nil {
			return nil, errors.Wrap(err, "failed to decode index info")
		}
		name, ok := indexInfo["name"]
		if !ok {
			return nil, errors.New("cannot get index name from index info")
		}
		indexName := name.(string)
		key, ok := indexInfo["key"]
		if !ok {
			return nil, errors.New("cannot get index key from index info")
		}
		expression, err := json.Marshal(key)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal index key to json")
		}
		unique := false
		if u, ok := indexInfo["unique"]; ok {
			unique = u.(bool)
		}

		if _, ok := indexMap[indexName]; !ok {
			indexMap[indexName] = &storepb.IndexMetadata{
				Name:   indexName,
				Unique: unique,
			}
		}
		indexMap[indexName].Expressions = append(indexMap[indexName].Expressions, string(expression))
	}

	var indexes []*storepb.IndexMetadata
	var indexNames []string
	for name := range indexMap {
		indexNames = append(indexNames, name)
	}
	sort.Strings(indexNames)
	for _, name := range indexNames {
		indexes = append(indexes, indexMap[name])
	}
	return indexes, nil
}

// getVersion returns the version of mongod or mongos instance.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	database := driver.client.Database(bytebaseDefaultDatabase)
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

// isDatabaseExist returns true if the database exists.
func (driver *Driver) isDatabaseExist(ctx context.Context, databaseName string) (bool, error) {
	// We do the filter by hand instead of using the filter option of ListDatabaseNames because we may encounter the following error:
	// Unallowed argument in listDatabases command: filter
	databaseList, err := driver.client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list database names")
	}
	for _, database := range databaseList {
		if database == databaseName {
			return true, nil
		}
	}
	return false, nil
}

// getNonSystemDatabaseList returns the list of non system databases.
func (driver *Driver) getNonSystemDatabaseList(ctx context.Context) ([]string, error) {
	databaseNames, err := driver.client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list database names")
	}
	var nonSystemDatabaseList []string
	for _, databaseName := range databaseNames {
		if _, ok := systemDatabase[databaseName]; !ok {
			nonSystemDatabaseList = append(nonSystemDatabaseList, databaseName)
		}
	}
	return nonSystemDatabaseList, nil
}

// isAtlasUnauthorizedError returns true if the error is an Atlas unauthorized error.
func isAtlasUnauthorizedError(err error) bool {
	commandError, ok := err.(mongo.CommandError)
	if ok {
		return commandError.Name == "AtlasError" && commandError.Code == 8000 && strings.Contains(commandError.Message, "Unauthorized")
	}
	return strings.Contains(err.Error(), "AtlasError: Unauthorized")
}

func convertEmptyInterfaceToInt64(value any) (int64, error) {
	// NOTE: convert uint64 to int64 may cause overflow. But we don't care about it.
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	default:
		return 0, errors.Errorf("cannot convert %v to int64", value)
	}
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
