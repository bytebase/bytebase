package mongodb

import (
	"context"
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
