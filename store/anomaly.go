package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.AnomalyService = (*AnomalyService)(nil)
)

// AnomalyService represents a service for managing anomaly.
type AnomalyService struct {
	l  *zap.Logger
	db *DB
}

// NewAnomalyService returns a new instance of AnomalyService.
func NewAnomalyService(logger *zap.Logger, db *DB) *AnomalyService {
	return &AnomalyService{l: logger, db: db}
}

// UpsertActiveAnomaly would update the existing active anomaly if both database id and type match, otherwise create a new one.
// Do not use ON CONFLICT (upsert syntax) as it will consume autoincrement id. Functional wise, this is fine, but
// from the UX perspective, it's not great, since user will see large id gaps.
func (s *AnomalyService) UpsertActiveAnomaly(ctx context.Context, upsert *api.AnomalyUpsert) (*api.Anomaly, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	status := api.Normal
	find := &api.AnomalyFind{
		RowStatus:  &status,
		InstanceID: &upsert.InstanceID,
		DatabaseID: upsert.DatabaseID,
		Type:       &upsert.Type,
	}
	list, err := findAnomalyList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	var anomaly *api.Anomaly
	if len(list) == 0 {
		anomaly, err = createAnomaly(ctx, tx.PTx, upsert)
		if err != nil {
			return nil, err
		}
	} else if len(list) == 1 {
		// Even if field value does not change, we still patch to update the updated_ts
		patch := &anomalyPatch{
			ID:        list[0].ID,
			UpdaterID: upsert.CreatorID,
			Payload:   upsert.Payload,
		}
		anomaly, err = patchAnomaly(ctx, tx.PTx, patch)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d active anomalies with filter %+v, expect 1", len(list), find)}
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return anomaly, nil
}

// FindAnomalyList retrieves a list of anomalys based on find.
func (s *AnomalyService) FindAnomalyList(ctx context.Context, find *api.AnomalyFind) ([]*api.Anomaly, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findAnomalyList(ctx, tx.PTx, find)
	if err != nil {
		return []*api.Anomaly{}, err
	}

	return list, nil
}

// ArchiveAnomaly archives an existing anomaly by ID.
// Returns ENOTFOUND if anomaly does not exist.
func (s *AnomalyService) ArchiveAnomaly(ctx context.Context, archive *api.AnomalyArchive) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := archiveAnomaly(ctx, tx.PTx, archive); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createAnomaly creates a new anomaly.
func createAnomaly(ctx context.Context, tx *sql.Tx, upsert *api.AnomalyUpsert) (*api.Anomaly, error) {
	// Inserts row into database.
	if upsert.Payload == "" {
		upsert.Payload = "{}"
	}
	row, err := tx.QueryContext(ctx, `
		INSERT INTO anomaly (
			creator_id,
			updater_id,
			instance_id,
			database_id,
			type,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, type, payload
	`,
		upsert.CreatorID,
		upsert.CreatorID,
		upsert.InstanceID,
		upsert.DatabaseID,
		upsert.Type,
		upsert.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var anomaly api.Anomaly
	databaseID := sql.NullInt32{}
	if err := row.Scan(
		&anomaly.ID,
		&anomaly.CreatorID,
		&anomaly.CreatedTs,
		&anomaly.UpdaterID,
		&anomaly.UpdatedTs,
		&anomaly.InstanceID,
		&databaseID,
		&anomaly.Type,
		&anomaly.Payload,
	); err != nil {
		return nil, FormatError(err)
	}
	if databaseID.Valid {
		value := int(databaseID.Int32)
		anomaly.DatabaseID = &value
	}
	anomaly.Severity = api.AnomalySeverityFromType(anomaly.Type)

	return nil, err
}

func findAnomalyList(ctx context.Context, tx *sql.Tx, find *api.AnomalyFind) ([]*api.Anomaly, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_id = $%d", len(args)+1)), append(args, *v)
		if find.InstanceOnly {
			where = append(where, "database_id is NULL")
		}
	}
	if find.InstanceID == nil || !find.InstanceOnly {
		if v := find.DatabaseID; v != nil {
			where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
		}
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			instance_id,
			database_id,
			type,
			payload
		FROM anomaly
		WHERE `+strings.Join(where, " AND ")+`
		`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Anomaly, 0)
	for rows.Next() {
		var anomaly api.Anomaly
		databaseID := sql.NullInt32{}
		if err := rows.Scan(
			&anomaly.ID,
			&anomaly.CreatorID,
			&anomaly.CreatedTs,
			&anomaly.UpdaterID,
			&anomaly.UpdatedTs,
			&anomaly.InstanceID,
			&databaseID,
			&anomaly.Type,
			&anomaly.Payload,
		); err != nil {
			return nil, FormatError(err)
		}
		if databaseID.Valid {
			value := int(databaseID.Int32)
			anomaly.DatabaseID = &value
		}
		anomaly.Severity = api.AnomalySeverityFromType(anomaly.Type)

		list = append(list, &anomaly)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

type anomalyPatch struct {
	ID int

	// Standard fields
	UpdaterID int

	// Domain specific fields
	Payload string
}

// patchAnomaly patches an anomaly
func patchAnomaly(ctx context.Context, tx *sql.Tx, patch *anomalyPatch) (*api.Anomaly, error) {
	// Build UPDATE clause.
	if patch.Payload == "" {
		patch.Payload = "{}"
	}
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	set, args = append(set, "payload = $2"), append(args, patch.Payload)
	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE anomaly
		SET `+strings.Join(set, ", ")+`
		WHERE id = $3
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, type, payload
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var anomaly api.Anomaly
	databaseID := sql.NullInt32{}
	if err := row.Scan(
		&anomaly.ID,
		&anomaly.CreatorID,
		&anomaly.CreatedTs,
		&anomaly.UpdaterID,
		&anomaly.UpdatedTs,
		&anomaly.InstanceID,
		&anomaly.DatabaseID,
		&anomaly.Type,
		&anomaly.Payload,
	); err != nil {
		return nil, FormatError(err)
	}
	if databaseID.Valid {
		value := int(databaseID.Int32)
		anomaly.DatabaseID = &value
	}
	anomaly.Severity = api.AnomalySeverityFromType(anomaly.Type)

	return &anomaly, err
}

// archiveAnomaly archives an anomaly by ID.
func archiveAnomaly(ctx context.Context, tx *sql.Tx, archive *api.AnomalyArchive) error {
	if archive.InstanceID == nil && archive.DatabaseID == nil {
		return &common.Error{Code: common.Internal, Err: fmt.Errorf("failed to close anomaly, should specify either instanceID or databaseID")}
	}
	if archive.InstanceID != nil && archive.DatabaseID != nil {
		return &common.Error{Code: common.Internal, Err: fmt.Errorf("failed to close anomaly, should specify either instanceID or databaseID, but not both")}
	}
	// Remove row from database.
	if archive.InstanceID != nil {
		result, err := tx.ExecContext(ctx,
			`UPDATE anomaly SET row_status = $1 WHERE instance_id = $2 AND database_id IS NULL AND type = $3`,
			api.Archived,
			*archive.InstanceID,
			archive.Type,
		)
		if err != nil {
			return FormatError(err)
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			return &common.Error{Code: common.NotFound, Err: fmt.Errorf("anomaly not found instance: %d type: %s", *archive.InstanceID, archive.Type)}
		}
	} else if archive.DatabaseID != nil {
		result, err := tx.ExecContext(ctx,
			`UPDATE anomaly SET row_status = $1 WHERE database_id = $2 AND type = $3`,
			api.Archived,
			*archive.DatabaseID,
			archive.Type,
		)
		if err != nil {
			return FormatError(err)
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			return &common.Error{Code: common.NotFound, Err: fmt.Errorf("anomaly not found database: %d type: %s", *archive.DatabaseID, archive.Type)}
		}
	}

	return nil
}
