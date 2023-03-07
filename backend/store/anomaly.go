package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// AnomalyMessage is the message of the anomaly.
type AnomalyMessage struct {
	// InstanceUID is the unique identifier of the instance.
	InstanceUID int
	// DatabaseUID is the unique identifier of the database, it will be nil if the anomaly is instance level.
	DatabaseUID *int
	// Type is the type of the anomaly.
	Type api.AnomalyType
	// Payload is the payload of the anomaly.
	Payload string
	// Output only fields.
	//
	// ID is the unique identifier of the anomaly.
	ID int
	// CreatedTs is the timestamp when the anomaly is created.
	CreatedTs int64
	// UpdatedTs is the timestamp when the anomaly is updated.
	UpdatedTs int64
}

// ToAPIAnomaly converts the anomaly message to the api anomaly.
func (a *AnomalyMessage) ToAPIAnomaly() *api.Anomaly {
	return &api.Anomaly{
		ID:         a.ID,
		UpdatedTs:  a.UpdatedTs,
		CreatedTs:  a.CreatedTs,
		InstanceID: a.InstanceUID,
		DatabaseID: a.DatabaseUID,
		Type:       a.Type,
		Payload:    a.Payload,
	}
}

// ListAnomalyMessage is the message to list anomalies.
type ListAnomalyMessage struct {
	RowStatus   *api.RowStatus
	InstanceUID *int
	DatabaseUID *int
	Types       []api.AnomalyType
}

// ArchiveAnomalyMessage is the message to archive an anomaly.
type ArchiveAnomalyMessage struct {
	InstanceUID *int
	DatabaseUID *int
	Type        api.AnomalyType
}

// updateAnomalyMessage is the message to update an anomaly.
type updateAnomalyMessage struct {
	ID      int
	Payload string
}

// UpsertActiveAnomalyV2 upserts an instance of anomaly.
func (s *Store) UpsertActiveAnomalyV2(ctx context.Context, principalUID int, upsert *AnomalyMessage) (*AnomalyMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	status := api.Normal
	find := &ListAnomalyMessage{
		RowStatus:   &status,
		InstanceUID: &upsert.InstanceUID,
		DatabaseUID: upsert.DatabaseUID,
		Types:       []api.AnomalyType{upsert.Type},
	}
	list, err := s.listAnomalyImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	var anomaly *AnomalyMessage
	if len(list) == 0 {
		anomaly, err = s.createAnomalyImplV2(ctx, tx, principalUID, &AnomalyMessage{
			InstanceUID: upsert.InstanceUID,
			DatabaseUID: upsert.DatabaseUID,
			Type:        upsert.Type,
			Payload:     upsert.Payload,
		})
		if err != nil {
			return nil, err
		}
	} else if len(list) == 1 {
		// Even if field value does not change, we still patch to update the updated_ts.
		anomaly, err = updateAnomalyV2(ctx, tx, principalUID, &updateAnomalyMessage{
			ID:      list[0].ID,
			Payload: upsert.Payload,
		})
		if err != nil {
			return nil, err
		}
	} else {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d active anomalies with filter %+v, expect 1", len(list), find)}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return anomaly, nil
}

// ListAnomalyV2 lists anomalies, only return the normal ones.
func (s *Store) ListAnomalyV2(ctx context.Context, list *ListAnomalyMessage) ([]*AnomalyMessage, error) {
	// Build where clause
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	anomalies, err := s.listAnomalyImplV2(ctx, tx, list)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return anomalies, nil
}

// ArchiveAnomalyV2 archives an anomaly.
func (s *Store) ArchiveAnomalyV2(ctx context.Context, archive *ArchiveAnomalyMessage) error {
	if archive.InstanceUID == nil && archive.DatabaseUID == nil {
		return &common.Error{Code: common.Internal, Err: errors.Errorf("failed to close anomaly, should specify either instanceID or databaseID")}
	}
	if archive.InstanceUID != nil && archive.DatabaseUID != nil {
		return &common.Error{Code: common.Internal, Err: errors.Errorf("failed to close anomaly, should specify either instanceID or databaseID, but not both")}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Remove row from database.
	if archive.InstanceUID != nil {
		result, err := tx.ExecContext(ctx,
			`UPDATE anomaly SET row_status = $1 WHERE instance_id = $2 AND database_id IS NULL AND type = $3`,
			api.Archived,
			*archive.InstanceUID,
			archive.Type,
		)
		if err != nil {
			return FormatError(err)
		}

		if rows, _ := result.RowsAffected(); rows == 0 {
			return &common.Error{Code: common.NotFound, Err: errors.Errorf("anomaly not found instance: %d type: %s", *archive.InstanceUID, archive.Type)}
		}
	} else if archive.DatabaseUID != nil {
		result, err := tx.ExecContext(ctx,
			`UPDATE anomaly SET row_status = $1 WHERE database_id = $2 AND type = $3`,
			api.Archived,
			*archive.DatabaseUID,
			archive.Type,
		)
		if err != nil {
			return FormatError(err)
		}

		if rows, _ := result.RowsAffected(); rows == 0 {
			return &common.Error{Code: common.NotFound, Err: errors.Errorf("anomaly not found database: %d type: %s", *archive.DatabaseUID, archive.Type)}
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}
	return nil
}

func (*Store) createAnomalyImplV2(ctx context.Context, tx *Tx, principalUID int, create *AnomalyMessage) (*AnomalyMessage, error) {
	// Inserts row into database.
	if create.Payload == "" {
		create.Payload = "{}"
	}
	query := `
		INSERT INTO anomaly (
			creator_id,
			updater_id,
			instance_id,
			database_id,
			type,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, instance_id, database_id, type, payload
	`
	var anomaly AnomalyMessage
	var databaseUID sql.NullInt32
	if err := tx.QueryRowContext(ctx, query,
		principalUID,
		principalUID,
		create.InstanceUID,
		create.DatabaseUID,
		create.Type,
		create.Payload,
	).Scan(
		&anomaly.ID,
		&anomaly.InstanceUID,
		databaseUID,
		&anomaly.Type,
		&anomaly.Payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	if databaseUID.Valid {
		value := int(databaseUID.Int32)
		anomaly.DatabaseUID = &value
	}

	return &anomaly, nil
}

func (*Store) listAnomalyImplV2(ctx context.Context, tx *Tx, list *ListAnomalyMessage) ([]*AnomalyMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if v := list.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := list.InstanceUID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := list.DatabaseUID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if len(list.Types) > 0 {
		var sub []string
		for _, v := range list.Types {
			sub, args = append(sub, fmt.Sprintf("$%d", len(args)+1)), append(args, v)
		}
		where = append(where, fmt.Sprintf("type IN (%s)", strings.Join(sub, `,`)))
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			created_ts,
			updated_ts,
			instance_id,
			database_id,
			type,
			payload
		FROM anomaly WHERE (%s
		AND EXISTS (
			SELECT 1
			FROM instance
			WHERE instance.id = anomaly.instance_id AND instance.row_status != 'ARCHIVED'
		))
	`, strings.Join(where, " AND "))

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var anomalies []*AnomalyMessage
	for rows.Next() {
		var anomaly AnomalyMessage
		// DatabaseID field can be NULL in the PostgreSQL database, so we use sql.NullInt32 to represent it.
		var databaseID sql.NullInt32
		if err := rows.Scan(
			&anomaly.ID,
			&anomaly.CreatedTs,
			&anomaly.UpdatedTs,
			&anomaly.InstanceUID,
			&databaseID,
			&anomaly.Type,
			&anomaly.Payload,
		); err != nil {
			return nil, FormatError(err)
		}
		if databaseID.Valid {
			value := int(databaseID.Int32)
			anomaly.DatabaseUID = &value
		}
		anomalies = append(anomalies, &anomaly)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return anomalies, nil
}

// updateAnomalyV2 patches an anomaly.
func updateAnomalyV2(ctx context.Context, tx *Tx, principalUID int, update *updateAnomalyMessage) (*AnomalyMessage, error) {
	// Build UPDATE clause.
	if update.Payload == "" {
		update.Payload = "{}"
	}
	set, args := []string{"updater_id = $1"}, []interface{}{principalUID}
	set, args = append(set, "payload = $2"), append(args, update.Payload)
	args = append(args, update.ID)

	// Execute update query with RETURNING.
	query := `
		UPDATE anomaly
		SET ` + strings.Join(set, ", ") + `
		WHERE id = $3
		RETURNING id, created_ts, updated_ts, instance_id, database_id, type, payload
	`
	var anomaly AnomalyMessage
	var databaseID sql.NullInt32
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&anomaly.ID,
		&anomaly.CreatedTs,
		&anomaly.UpdatedTs,
		&anomaly.InstanceUID,
		&databaseID,
		&anomaly.Type,
		&anomaly.Payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	if databaseID.Valid {
		value := int(databaseID.Int32)
		anomaly.DatabaseUID = &value
	}

	return &anomaly, nil
}
