package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// AnomalyMessage is the message of the anomaly.
type AnomalyMessage struct {
	ProjectID string
	// InstanceUID is the instance uid.
	InstanceUID int
	// DatabaseUID is the unique identifier of the database, it will be nil if the anomaly is instance level.
	DatabaseUID int
	// Type is the type of the anomaly.
	Type api.AnomalyType
	// Output only fields.
	//
	// UID is the unique identifier of the anomaly.
	UID int
	// UpdatedTs is the timestamp when the anomaly is updated.
	UpdatedTs time.Time
}

// ListAnomalyMessage is the message to list anomalies.
type ListAnomalyMessage struct {
	ProjectID   string
	InstanceID  *string
	DatabaseUID *int
	Types       []api.AnomalyType
}

// DeleteAnomalyMessage is the message to delete an anomaly.
type DeleteAnomalyMessage struct {
	InstanceID  string
	DatabaseUID int
	Type        api.AnomalyType
}

// UpsertActiveAnomalyV2 upserts an instance of anomaly.
func (s *Store) UpsertActiveAnomalyV2(ctx context.Context, upsert *AnomalyMessage) (*AnomalyMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	upsert.UpdatedTs = time.Now()
	query := `
	INSERT INTO anomaly (
		updated_at,
		project,
		instance_id,
		database_id,
		type
	)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (project, database_id, type) DO UPDATE SET
		updated_at = EXCLUDED.updated_at
`
	if _, err := tx.ExecContext(ctx, query,
		upsert.UpdatedTs,
		upsert.ProjectID,
		upsert.InstanceUID,
		upsert.DatabaseUID,
		upsert.Type,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return upsert, nil
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

// DeleteAnomalyV2 deletes an anomaly.
func (s *Store) DeleteAnomalyV2(ctx context.Context, d *DeleteAnomalyMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM anomaly WHERE database_id = $1 AND type = $2`,
		d.DatabaseUID,
		d.Type,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (*Store) listAnomalyImplV2(ctx context.Context, tx *Tx, list *ListAnomalyMessage) ([]*AnomalyMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("anomaly.project = $%d", len(args)+1)), append(args, list.ProjectID)
	if v := list.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := list.DatabaseUID; v != nil {
		where, args = append(where, fmt.Sprintf("anomaly.database_id = $%d", len(args)+1)), append(args, *v)
	}
	if len(list.Types) > 0 {
		var sub []string
		for _, v := range list.Types {
			sub, args = append(sub, fmt.Sprintf("$%d", len(args)+1)), append(args, v)
		}
		where = append(where, fmt.Sprintf("anomaly.type IN (%s)", strings.Join(sub, `,`)))
	}

	query := fmt.Sprintf(`
		SELECT
			anomaly.id,
			anomaly.updated_at,
			anomaly.instance_id,
			anomaly.database_id,
			anomaly.type
		FROM anomaly
		LEFT JOIN instance ON anomaly.instance_id = instance.id
		WHERE (%s
			AND EXISTS (
				SELECT 1
				FROM instance
				WHERE instance.id = anomaly.instance_id
			)
		)
	`, strings.Join(where, " AND "))

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var anomalies []*AnomalyMessage
	for rows.Next() {
		var anomaly AnomalyMessage
		// DatabaseID field can be NULL in the PostgreSQL database, so we use sql.NullInt32 to represent it.
		var databaseID sql.NullInt32
		if err := rows.Scan(
			&anomaly.UID,
			&anomaly.UpdatedTs,
			&anomaly.InstanceUID,
			&databaseID,
			&anomaly.Type,
		); err != nil {
			return nil, err
		}
		if databaseID.Valid {
			value := int(databaseID.Int32)
			anomaly.DatabaseUID = value
		}
		anomalies = append(anomalies, &anomaly)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return anomalies, nil
}
