package store

import (
	"context"
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

// UpsertAnomaly would update the existing user if name matches.
func (s *AnomalyService) UpsertAnomaly(ctx context.Context, upsert *api.AnomalyUpsert) (*api.Anomaly, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	anomaly, err := upsertAnomaly(ctx, tx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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
	defer tx.Rollback()

	list, err := findAnomalyList(ctx, tx, find)
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
	defer tx.Rollback()

	err = archiveAnomaly(ctx, tx, archive)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// upsertAnomaly upserts a new anomaly.
func upsertAnomaly(ctx context.Context, tx *Tx, upsert *api.AnomalyUpsert) (*api.Anomaly, error) {
	// Upsert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO anomaly (
			creator_id,
			updater_id,
			instance_id,
			database_id,
			`+"`type`,"+`
			payload
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (database_id, `+"`type`"+`) DO UPDATE SET
			updater_id = excluded.updater_id,
			payload = excluded.payload
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, `+"`type`"+`, payload
	`,
		upsert.CreatorId,
		upsert.CreatorId,
		upsert.InstanceId,
		upsert.DatabaseId,
		upsert.Type,
		upsert.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var anomaly api.Anomaly
	if err := row.Scan(
		&anomaly.ID,
		&anomaly.CreatorId,
		&anomaly.CreatedTs,
		&anomaly.UpdaterId,
		&anomaly.UpdatedTs,
		&anomaly.InstanceId,
		&anomaly.DatabaseId,
		&anomaly.Type,
		&anomaly.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return nil, err
}

func findAnomalyList(ctx context.Context, tx *Tx, find *api.AnomalyFind) (_ []*api.Anomaly, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	where, args = append(where, "database_id = ?"), append(args, find.DatabaseId)
	if v := find.RowStatus; v != nil {
		where, args = append(where, "row_status = ?"), append(args, *v)
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
			`+"`type`,"+`
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
		if err := rows.Scan(
			&anomaly.ID,
			&anomaly.CreatorId,
			&anomaly.CreatedTs,
			&anomaly.UpdaterId,
			&anomaly.UpdatedTs,
			&anomaly.InstanceId,
			&anomaly.DatabaseId,
			&anomaly.Type,
			&anomaly.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &anomaly)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// archiveAnomaly archives an anomaly by ID.
func archiveAnomaly(ctx context.Context, tx *Tx, archive *api.AnomalyArchive) error {
	// Remove row from database.
	result, err := tx.ExecContext(ctx,
		`UPDATE anomaly SET row_status = ? WHERE database_id = ? AND type = ?`,
		api.Archived,
		archive.DatabaseId,
		archive.Type,
	)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &common.Error{Code: common.NotFound, Err: fmt.Errorf("anomaly not found database: %d type: %s", archive.DatabaseId, archive.Type)}
	}

	return nil
}
