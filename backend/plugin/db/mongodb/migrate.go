package mongodb

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

const (
	// migrationHistoryDefaultDatabase is the default database name for migration history.
	migrationHistoryDefaultDatabase = "bytebase"
	// migrationHistoryDefaultCollection is the default collection name for migration history.
	migrationHistoryDefaultCollection = "migration_history"
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
	Description         string              `bson:"description"`
	Statement           string              `bson:"statement"`
	ExecutionDurationNs int64               `bson:"execution_duration_ns"`
	IssueID             string              `bson:"issue_id"`
	Payload             string              `bson:"payload"`
}

// ExecuteMigration executes a migration.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (string, string, error) {
	_, err := driver.Execute(ctx, statement, m.CreateDatabase)
	return "", "", err
}

// FindMigrationHistoryList finds the migration history list.
func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	database := driver.client.Database(migrationHistoryDefaultDatabase)
	collection := database.Collection(migrationHistoryDefaultCollection)
	filter := bson.M{}
	if v := find.ID; v != nil {
		longMigrationHistoryID, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse inserted ID %s to int64", *v)
		}
		filter["id"] = longMigrationHistoryID
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
		migrationHistory, err := convertMigrationHistory(history)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert migration history")
		}
		migrationHistoryList = append(migrationHistoryList, &migrationHistory)
	}

	return migrationHistoryList, nil
}

func convertMigrationHistory(history MigrationHistory) (db.MigrationHistory, error) {
	useSemanticVersion, version, semanticVersionSuffix, err := util.FromStoredVersion(history.Version)
	if err != nil {
		return db.MigrationHistory{}, err
	}
	return db.MigrationHistory{
		ID:                    fmt.Sprintf("%d", history.ID),
		Creator:               history.CreatedBy,
		CreatedTs:             int64(history.CreatedTs.T),
		Updater:               history.UpdatedBy,
		UpdatedTs:             int64(history.UpdatedTs.T),
		ReleaseVersion:        history.ReleaseVersion,
		Namespace:             history.Namespace,
		Sequence:              int(history.Sequence),
		Source:                db.MigrationSource(history.Source),
		Type:                  db.MigrationType(history.Type),
		Status:                db.MigrationStatus(history.Status),
		Version:               version,
		Description:           history.Description,
		Statement:             history.Statement,
		ExecutionDurationNs:   history.ExecutionDurationNs,
		IssueID:               history.IssueID,
		Payload:               history.Payload,
		UseSemanticVersion:    useSemanticVersion,
		SemanticVersionSuffix: semanticVersionSuffix,
	}, nil
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
