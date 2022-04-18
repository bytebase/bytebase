package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// anomalyRaw is the store model for an Anomaly.
// Fields have exactly the same meanings as Anomaly.
type anomalyRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	InstanceID int
	DatabaseID *int

	// Domain specific fields
	Type api.AnomalyType
	// Calculated field derived from type
	Severity api.AnomalySeverity
	Payload  string
}

// toAnomaly creates an instance of Anomaly based on the anomalyRaw.
// This is intended to be called when we need to compose an Anomaly relationship.
func (raw *anomalyRaw) toAnomaly() *api.Anomaly {
	return &api.Anomaly{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		InstanceID: raw.InstanceID,
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Type: raw.Type,
		// Calculated field derived from type
		Severity: raw.Severity,
		Payload:  raw.Payload,
	}
}

// UpsertActiveAnomaly upserts an instance of anomaly
func (s *Store) UpsertActiveAnomaly(ctx context.Context, upsert *api.AnomalyUpsert) (*api.Anomaly, error) {
	anomalyRaw, err := s.upsertActiveAnomalyRaw(ctx, upsert)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert active anomaly with AnomalyUpsert[%+v], error[%w]", upsert, err)
	}
	anomaly, err := s.composeAnomaly(ctx, anomalyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose anomaly with AnomalyRaw[%+v], error[%w]", anomalyRaw, err)
	}
	return anomaly, nil
}

// FindAnomaly finds a list of Anomaly instances
func (s *Store) FindAnomaly(ctx context.Context, find *api.AnomalyFind) ([]*api.Anomaly, error) {
	anomalyRawList, err := s.findAnomalyRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find anomaly with AnomalyFind[%+v], error[%w]", find, err)
	}
	var anomalyList []*api.Anomaly
	for _, anomalyRaw := range anomalyRawList {
		anomaly, err := s.composeAnomaly(ctx, anomalyRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose anomaly with AnomalyRaw[%+v], error[%w]", anomalyRaw, err)
		}
		anomalyList = append(anomalyList, anomaly)
	}
	return anomalyList, nil
}

//
// private functions
//

func (s *Store) composeAnomaly(ctx context.Context, raw *anomalyRaw) (*api.Anomaly, error) {
	anomaly := raw.toAnomaly()

	creator, err := s.GetPrincipalByID(ctx, anomaly.CreatorID)
	if err != nil {
		return nil, err
	}
	anomaly.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, anomaly.UpdaterID)
	if err != nil {
		return nil, err
	}
	anomaly.Updater = updater

	return anomaly, nil
}

// upsertActiveAnomalyRaw would update the existing active anomaly if both database id and type match, otherwise create a new one.
// Do not use ON CONFLICT (upsert syntax) as it will consume autoincrement id. Functional wise, this is fine, but
// from the UX perspective, it's not great, since user will see large id gaps.
func (s *Store) upsertActiveAnomalyRaw(ctx context.Context, upsert *api.AnomalyUpsert) (*anomalyRaw, error) {
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
	list, err := findAnomalyListImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	var anomalyRaw *anomalyRaw
	if len(list) == 0 {
		anomalyRaw, err = createAnomalyImpl(ctx, tx.PTx, upsert)
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
		anomalyRaw, err = patchAnomalyImpl(ctx, tx.PTx, patch)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d active anomalies with filter %+v, expect 1", len(list), find)}
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return anomalyRaw, nil
}

// findAnomalyRaw retrieves a list of anomalies based on the find condition.
func (s *Store) findAnomalyRaw(ctx context.Context, find *api.AnomalyFind) ([]*anomalyRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findAnomalyListImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// ArchiveAnomaly archives an existing anomaly by ID.
// Returns ENOTFOUND if anomaly does not exist.
func (s *Store) ArchiveAnomaly(ctx context.Context, archive *api.AnomalyArchive) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := archiveAnomalyImpl(ctx, tx.PTx, archive); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createAnomalyImpl creates a new anomaly.
func createAnomalyImpl(ctx context.Context, tx *sql.Tx, upsert *api.AnomalyUpsert) (*anomalyRaw, error) {
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
	var anomalyRaw anomalyRaw
	databaseID := sql.NullInt32{}
	if err := row.Scan(
		&anomalyRaw.ID,
		&anomalyRaw.CreatorID,
		&anomalyRaw.CreatedTs,
		&anomalyRaw.UpdaterID,
		&anomalyRaw.UpdatedTs,
		&anomalyRaw.InstanceID,
		&databaseID,
		&anomalyRaw.Type,
		&anomalyRaw.Payload,
	); err != nil {
		return nil, FormatError(err)
	}
	if databaseID.Valid {
		value := int(databaseID.Int32)
		anomalyRaw.DatabaseID = &value
	}
	anomalyRaw.Severity = api.AnomalySeverityFromType(anomalyRaw.Type)

	return &anomalyRaw, err
}

func findAnomalyListImpl(ctx context.Context, tx *sql.Tx, find *api.AnomalyFind) ([]*anomalyRaw, error) {
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

	// Iterate over result set and deserialize rows into anomalyRawList.
	var anomalyRawList []*anomalyRaw
	for rows.Next() {
		var anomalyRaw anomalyRaw
		databaseID := sql.NullInt32{}
		if err := rows.Scan(
			&anomalyRaw.ID,
			&anomalyRaw.CreatorID,
			&anomalyRaw.CreatedTs,
			&anomalyRaw.UpdaterID,
			&anomalyRaw.UpdatedTs,
			&anomalyRaw.InstanceID,
			&databaseID,
			&anomalyRaw.Type,
			&anomalyRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}
		if databaseID.Valid {
			value := int(databaseID.Int32)
			anomalyRaw.DatabaseID = &value
		}
		anomalyRaw.Severity = api.AnomalySeverityFromType(anomalyRaw.Type)

		anomalyRawList = append(anomalyRawList, &anomalyRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return anomalyRawList, nil
}

type anomalyPatch struct {
	ID int

	// Standard fields
	UpdaterID int

	// Domain specific fields
	Payload string
}

// patchAnomalyImpl patches an anomaly
func patchAnomalyImpl(ctx context.Context, tx *sql.Tx, patch *anomalyPatch) (*anomalyRaw, error) {
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
	var anomalyRaw anomalyRaw
	databaseID := sql.NullInt32{}
	if err := row.Scan(
		&anomalyRaw.ID,
		&anomalyRaw.CreatorID,
		&anomalyRaw.CreatedTs,
		&anomalyRaw.UpdaterID,
		&anomalyRaw.UpdatedTs,
		&anomalyRaw.InstanceID,
		&databaseID,
		&anomalyRaw.Type,
		&anomalyRaw.Payload,
	); err != nil {
		return nil, FormatError(err)
	}
	if databaseID.Valid {
		value := int(databaseID.Int32)
		anomalyRaw.DatabaseID = &value
	}
	anomalyRaw.Severity = api.AnomalySeverityFromType(anomalyRaw.Type)

	return &anomalyRaw, err
}

// archiveAnomalyImpl archives an anomaly by ID.
func archiveAnomalyImpl(ctx context.Context, tx *sql.Tx, archive *api.AnomalyArchive) error {
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
