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
	ProjectID    string
	InstanceID   string
	DatabaseName string
	// Type is the type of the anomaly.
	Type api.AnomalyType
	// Output only fields.
	//
	// UID is the unique identifier of the anomaly.
	UID int
	// UpdatedAt is the timestamp when the anomaly is updated.
	UpdatedAt time.Time
}

// ListAnomalyMessage is the message to list anomalies.
type ListAnomalyMessage struct {
	ProjectID    string
	InstanceID   *string
	DatabaseName *string
	Types        []api.AnomalyType
}

// DeleteAnomalyMessage is the message to delete an anomaly.
type DeleteAnomalyMessage struct {
	InstanceID   string
	DatabaseName string
	Type         api.AnomalyType
}

// UpsertActiveAnomalyV2 upserts an instance of anomaly.
func (s *Store) UpsertActiveAnomalyV2(ctx context.Context, upsert *AnomalyMessage) (*AnomalyMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	upsert.UpdatedAt = time.Now()
	query := `
	INSERT INTO anomaly (
		updated_at,
		project,
		instance,
		db_name,
		type
	)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (project, instance, db_name, type) DO UPDATE SET
		updated_at = EXCLUDED.updated_at
`
	if _, err := tx.ExecContext(ctx, query,
		upsert.UpdatedAt,
		upsert.ProjectID,
		upsert.InstanceID,
		upsert.DatabaseName,
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
	txn, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer txn.Rollback()

	anomalies, err := s.listAnomalyImplV2(ctx, txn, list)
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
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
		`DELETE FROM anomaly WHERE instance = $1 AND db_name = $2 AND type = $3`,
		d.InstanceID,
		d.DatabaseName,
		d.Type,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (*Store) listAnomalyImplV2(ctx context.Context, txn *sql.Tx, list *ListAnomalyMessage) ([]*AnomalyMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("project = $%d", len(args)+1)), append(args, list.ProjectID)
	if v := list.InstanceID; v != nil {
		where, args = append(where, fmt.Sprintf("instance = $%d", len(args)+1)), append(args, *v)
	}
	if v := list.DatabaseName; v != nil {
		where, args = append(where, fmt.Sprintf("db_name = $%d", len(args)+1)), append(args, *v)
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
			updated_at,
			instance,
			db_name,
			type
		FROM anomaly
		WHERE %s
	`, strings.Join(where, " AND "))

	rows, err := txn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var anomalies []*AnomalyMessage
	for rows.Next() {
		var anomaly AnomalyMessage
		if err := rows.Scan(
			&anomaly.UID,
			&anomaly.UpdatedAt,
			&anomaly.InstanceID,
			&anomaly.DatabaseName,
			&anomaly.Type,
		); err != nil {
			return nil, err
		}
		anomalies = append(anomalies, &anomaly)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return anomalies, nil
}
