package mongodb

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	// embed will embeds the migration schema.
	_ "embed"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
)

const (
	// migrationHistoryDefaultDatabase is the default database name for migration history.
	migrationHistoryDefaultDatabase = "bytebase"
	// migrationHistoryDefaultCollection is the default collection name for migration history.
	migrationHistoryDefaultCollection = "migration_history"
)

var (
	//go:embed collmod_migrationHistory_collection_command.json
	collmodMigrationHistoryCollectionCommand string
	//go:embed create_index_on_migrationHistory_collection_command.json
	createIndexOnMigrationHistoryCollectionCommand string

	_ util.MigrationExecutor = (*Driver)(nil)
)

// MigrationHistory is the migration history record in MongoDB.
type MigrationHistory struct {
	ID                  int64               `bson:"id"`
	CreatedBy           string              `bson:"created_by"`
	CreatedTs           primitive.Timestamp `bson:"created_ts"`
	UpdatedBy           string              `bson:"updated_by"`
	UpdatedTs           primitive.Timestamp `bson:"updated_ts"`
	ReleaseVersion      string              `bson:"release_version"`
	Namespace           string              `bson:"namespace"`
	Sequence            int64               `bson:"sequence"`
	Source              string              `bson:"source"`
	Type                string              `bson:"type"`
	Status              string              `bson:"status"`
	Version             string              `bson:"version"`
	Description         string              `bson:"hash"`
	Statement           string              `bson:"statement"`
	ExecutionDurationNs int64               `bson:"execution_duration_ns"`
	IssueID             string              `bson:"issue_id"`
	Payload             string              `bson:"payload"`
}

// NeedsSetupMigration returns whether the driver needs to setup migration.
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	database := driver.client.Database(migrationHistoryDefaultDatabase)
	collectionNames, err := database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to list collection names for database %q", migrationHistoryDefaultDatabase)
	}
	for _, collectionName := range collectionNames {
		if collectionName == migrationHistoryDefaultCollection {
			return false, nil
		}
	}
	return true, nil
}

// SetupMigrationIfNeeded sets up migration if needed.
func (driver *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return err
	}
	if !setup {
		return nil
	}
	log.Info("Bytebase migration schema not found, creating schema for mongodb...",
		zap.String("environment", driver.connectionCtx.EnvironmentName),
		zap.String("instance", driver.connectionCtx.InstanceName),
	)
	database := driver.client.Database(migrationHistoryDefaultDatabase)
	if err := database.CreateCollection(ctx, migrationHistoryDefaultCollection); err != nil {
		return errors.Wrapf(err, "failed to create collection %q in mongodb database %q for instance named %q", migrationHistoryDefaultCollection, migrationHistoryDefaultDatabase, driver.connectionCtx.InstanceName)
	}

	var b interface{}
	if err := bson.UnmarshalExtJSON([]byte(collmodMigrationHistoryCollectionCommand), true, &b); err != nil {
		return errors.Wrap(err, "failed to unmarshal collmod command")
	}
	var result interface{}
	if err := database.RunCommand(context.Background(), b).Decode(&result); err != nil {
		return errors.Wrap(err, "failed to run collmod command")
	}

	if err := bson.UnmarshalExtJSON([]byte(createIndexOnMigrationHistoryCollectionCommand), true, &b); err != nil {
		return errors.Wrap(err, "failed to unmarshal create index command")
	}
	if err := database.RunCommand(context.Background(), b).Decode(&result); err != nil {
		return errors.Wrap(err, "failed to run create index command")
	}
	log.Info("Successfully created migration schema for mongodb.",
		zap.String("environment", driver.connectionCtx.EnvironmentName),
		zap.String("instance", driver.connectionCtx.InstanceName),
	)
	return nil
}

// ExecuteMigration executes a migration.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	return util.ExecuteMigration(ctx, driver, m, statement, migrationHistoryDefaultDatabase)
}

// FindMigrationHistoryList finds the migration history list.
func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	database := driver.client.Database(migrationHistoryDefaultDatabase)
	collection := database.Collection(migrationHistoryDefaultCollection)
	filter := bson.M{}
	if v := find.ID; v != nil {
		id, err := primitive.ObjectIDFromHex(strconv.Itoa(*v))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert id %q to object id", strconv.Itoa(*v))
		}
		filter["id"] = id
	}
	if v := find.Database; v != nil {
		filter["namespace"] = *v
	}
	if v := find.Version; v != nil {
		storedVersion, err := util.ToStoredVersion(false, *v, "")
		if err != nil {
			return nil, err
		}
		filter["version"] = storedVersion
	}
	if v := find.Source; v != nil {
		filter["source"] = *v
	}
	findOptions := []*options.FindOptions{
		// Sort by id in descending order.
		options.Find().SetSort(bson.M{"id": -1}),
	}
	if v := find.Limit; v != nil {
		findOptions = append(findOptions, options.Find().SetLimit(int64(*v)))
	}

	cursor, err := collection.Find(ctx, filter, findOptions...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find migration history list")
	}
	defer cursor.Close(ctx)

	var migrationHistoryList []*db.MigrationHistory
	for cursor.Next(ctx) {
		var history MigrationHistory
		if err := cursor.Decode(&history); err != nil {
			return nil, errors.Wrapf(err, "failed to decode migration history")
		}
		migrationHistory := convertMigrationHistory(history)
		migrationHistoryList = append(migrationHistoryList, &migrationHistory)
	}

	return migrationHistoryList, nil
}

func convertMigrationHistory(history MigrationHistory) db.MigrationHistory {
	return db.MigrationHistory{
		ID:                  int(history.ID),
		Creator:             history.CreatedBy,
		CreatedTs:           int64(history.CreatedTs.T),
		Updater:             history.UpdatedBy,
		UpdatedTs:           int64(history.UpdatedTs.T),
		ReleaseVersion:      history.ReleaseVersion,
		Namespace:           history.Namespace,
		Sequence:            int(history.Sequence),
		Source:              db.MigrationSource(history.Source),
		Type:                db.MigrationType(history.Type),
		Status:              db.MigrationStatus(history.Status),
		Version:             history.Version,
		Description:         history.Description,
		Statement:           history.Statement,
		ExecutionDurationNs: history.ExecutionDurationNs,
		IssueID:             history.IssueID,
		Payload:             history.Payload,
	}
}

// FindLargestSequence finds the largest sequence, return 0 if not found.
func (driver *Driver) FindLargestSequence(ctx context.Context, _ *sql.Tx, namespace string, baseline bool) (int, error) {
	collection := driver.client.Database(migrationHistoryDefaultDatabase).Collection(migrationHistoryDefaultCollection)
	filter := bson.M{
		"namespace": namespace,
	}
	if baseline {
		filter["type"] = bson.M{"$in": []string{string(db.Baseline), string(db.Branch)}}
	}

	// Set up MAX(sequence) aggregation.
	aggregation := []bson.M{
		{
			"$match": filter,
		},
		{
			"$group": bson.M{
				"_id":      nil,
				"sequence": bson.M{"$max": "$sequence"},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, aggregation)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to aggregate migration history list to find largest sequence")
	}
	defer cursor.Close(ctx)

	var result bson.M
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, errors.Wrapf(err, "failed to decode find largest sequence aggregation result")
		}
		if sequence, ok := result["sequence"]; ok {
			if sequenceInt, ok := sequence.(int64); ok {
				return int(sequenceInt), nil
			}
		}
		return 0, errors.Errorf("failed to get sequence from find largest sequence aggregation result: %+v", result)
	}
	// If cursor contains no data, we check the err, if err is nil, it means that the query is successful, but there is no data.
	if err := cursor.Err(); err != nil {
		return 0, errors.Wrapf(err, "failed to find largest sequence")
	}
	return 0, nil
}

// FindLargestVersionSinceBaseline will find the largest version since last baseline or branch.
func (driver *Driver) FindLargestVersionSinceBaseline(ctx context.Context, tx *sql.Tx, namespace string) (*string, error) {
	collection := driver.client.Database(migrationHistoryDefaultDatabase).Collection(migrationHistoryDefaultCollection)
	largestBaselineSeuqence, err := driver.FindLargestSequence(ctx, tx, namespace, true /* baseline */)
	if err != nil {
		return nil, err
	}
	filter := bson.M{
		"namespace": namespace,
		"sequence":  bson.M{"$gte": largestBaselineSeuqence},
	}
	// Set up MAX(version) aggregation.
	aggregation := []bson.M{
		{
			"$match": filter,
		},
		{
			"$group": bson.M{
				"_id":     nil,
				"version": bson.M{"$max": "$version"},
			},
		},
	}
	cursor, err := collection.Aggregate(ctx, aggregation)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to aggregate migration history list to find largest version since baseline")
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return nil, errors.Wrapf(err, "failed to decode find largest version since baseline aggregation result")
		}
		if version, ok := result["version"]; ok {
			if versionStr, ok := version.(string); ok {
				return &versionStr, nil
			}
		}
		return nil, errors.Errorf("failed to get version from find largest version since baseline aggregation result: %v", result)
	}
	// If cursor contains no data, we check the err, if err is nil, it means that the query is successful, but there is no data.
	if err := cursor.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to find largest version since baseline")
	}
	return nil, nil
}

// InsertPendingHistory will insert the migration record with pending status and return the inserted ID.
func (driver *Driver) InsertPendingHistory(ctx context.Context, _ *sql.Tx, sequence int, _ string, m *db.MigrationInfo, storedVersion, statement string) (insertedID int64, err error) {
	collection := driver.client.Database(migrationHistoryDefaultDatabase).Collection(migrationHistoryDefaultCollection)

	retryTimes := 3
	nextID := int64(0)
	for i := 0; i < retryTimes; i++ {
		currentTimestamp := getMongoTimestamp()
		nextID, err = driver.getMigrationHistoryNextID(ctx)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to get next migration history ID")
		}

		document := bson.M{
			"id":                    nextID,
			"created_by":            m.Creator,
			"created_ts":            currentTimestamp,
			"updated_by":            m.Creator,
			"updated_ts":            currentTimestamp,
			"release_version":       m.ReleaseVersion,
			"namespace":             m.Namespace,
			"sequence":              int64(sequence),
			"source":                m.Source,
			"type":                  m.Type,
			"status":                db.Pending,
			"version":               storedVersion,
			"description":           m.Description,
			"statement":             statement,
			"execution_duration_ns": int64(0),
			"issue_id":              m.IssueID,
			"payload":               m.Payload,
		}
		if _, err = collection.InsertOne(ctx, document); err != nil {
			// Detect if it's a duplicate key error.
			if driverErr, ok := err.(mongo.WriteException); ok {
				for _, writeErr := range driverErr.WriteErrors {
					if writeErr.Code == 121 {
						log.Info("Duplicate key error with migration_history id: %d, retrying...", zap.Int64("id", nextID))
						continue
					}
				}
			}
			return 0, errors.Wrapf(err, "failed to insert a pending migration history record")
		}
		return nextID, nil
	}
	return nextID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (driver *Driver) UpdateHistoryAsDone(ctx context.Context, _ *sql.Tx, migrationDurationNs int64, _ string, insertedID int64) error {
	collection := driver.client.Database(migrationHistoryDefaultDatabase).Collection(migrationHistoryDefaultCollection)
	filter := bson.M{
		"id": insertedID,
	}
	update := bson.M{
		"$set": bson.M{
			"status":                db.Done,
			"execution_duration_ns": migrationDurationNs,
		},
	}

	updateResult, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.Wrapf(err, "failed to update a migration history as done")
	}
	if updateResult.MatchedCount == 0 {
		return errors.Errorf("failed to find migration history with ID %d", insertedID)
	}
	return nil
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (driver *Driver) UpdateHistoryAsFailed(ctx context.Context, _ *sql.Tx, migrationDurationNs int64, insertedID int64) error {
	collection := driver.client.Database(migrationHistoryDefaultDatabase).Collection(migrationHistoryDefaultCollection)
	filter := bson.M{
		"id": insertedID,
	}
	update := bson.M{
		"$set": bson.M{
			"status":                db.Failed,
			"execution_duration_ns": migrationDurationNs,
		},
	}
	var result MigrationHistory
	if err := collection.FindOneAndUpdate(ctx, filter, update).Decode(&result); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.Errorf("failed to find migration history with ID %d", insertedID)
		}
		return errors.Wrapf(err, "failed to update migration history with ID %d", insertedID)
	}
	return nil
}

func getMongoTimestamp() primitive.Timestamp {
	return primitive.Timestamp{T: uint32(time.Now().Unix()), I: 1}
}

// getMigrationHistoryNextID will get the next ID for the migration history.
// In RDBMS, it is usually auto-increment, but in MongoDB, we need to maintain the ID by ourselves.
// Multiple goroutines may call this function at the same time and the ID may be duplicated.
// It's caller's responsibility to handle the duplicated ID, likes retry.
func (driver *Driver) getMigrationHistoryNextID(ctx context.Context) (int64, error) {
	collection := driver.client.Database(migrationHistoryDefaultDatabase).Collection(migrationHistoryDefaultCollection)
	// Set up MAX(id) aggregation.
	aggregation := []bson.M{
		{
			"$group": bson.M{
				"_id": nil,
				"id":  bson.M{"$max": "$id"},
			},
		},
	}
	cursor, err := collection.Aggregate(ctx, aggregation)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to aggregate migration history list to find largest ID")
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return 0, errors.Wrapf(err, "failed to decode find largest ID aggregation result")
		}
		if resultID, ok := result["id"]; ok {
			if id, ok := resultID.(int64); ok {
				return id + 1, nil
			}
		}
		return 0, errors.Errorf("failed to get ID from find largest ID aggregation result: %v", result)
	}
	// If cursor contains no data, we check the err, if err is nil, it means that the query is successful, but there is no data.
	if err := cursor.Err(); err != nil {
		return 0, errors.Wrapf(err, "failed to find largest ID")
	}
	// Return 1 if there is no data.
	return 1, nil
}
