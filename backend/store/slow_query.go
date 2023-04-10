package store

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// UpsertSlowLogMessage is the message to upsert slow query logs.
type UpsertSlowLogMessage struct {
	// We need EnvironmentID, InstanceID, and DatabaseName to find the database UID.
	EnvironmentID *string
	InstanceID    *string
	DatabaseName  string

	InstanceUID int
	LogDate     time.Time
	SlowLog     *storepb.SlowQueryStatistics

	UpdaterID int
}

// UpsertSlowLog upserts slow query logs.
func (s *Store) UpsertSlowLog(ctx context.Context, upsert *UpsertSlowLogMessage) error {
	var databaseUID sql.NullInt32
	if upsert.DatabaseName != "" {
		database, err := s.GetDatabaseV2(ctx, &FindDatabaseMessage{
			EnvironmentID: upsert.EnvironmentID,
			InstanceID:    upsert.InstanceID,
			DatabaseName:  &upsert.DatabaseName,
		})
		if err != nil {
			return err
		}
		if database != nil {
			databaseUID.Int32 = int32(database.UID)
			databaseUID.Valid = true
		}
	}

	logDate, err := strconv.Atoi(upsert.LogDate.Format("20060102"))
	if err != nil {
		return err
	}

	logBytes, err := protojson.Marshal(upsert.SlowLog)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO slow_query (
			creator_id,
			updater_id,
			instance_id,
			database_id,
			log_date_ts,
			slow_query_statistics
		) VALUES ( $1, $2, $3, $4, $5, $6 )
		ON CONFLICT (database_id, log_date_ts) DO UPDATE SET
			updater_id = EXCLUDED.updater_id,
			slow_query_statistics = EXCLUDED.slow_query_statistics
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.InstanceUID,
		databaseUID,
		logDate,
		logBytes,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteOutdatedSlowLog deletes outdated slow query logs.
func (s *Store) DeleteOutdatedSlowLog(ctx context.Context, instanceUID int, earliestDate time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.deleteSlowLogImpl(ctx, tx, instanceUID, earliestDate); err != nil {
		return err
	}

	return tx.Commit()
}

func (*Store) deleteSlowLogImpl(ctx context.Context, tx *Tx, instanceUID int, earliestDate time.Time) error {
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM
			slow_query
		WHERE
			instance_id = $1
			AND log_date_ts < $2`,
		instanceUID,
		earliestDate.Format("20060102"),
	); err != nil {
		return err
	}
	return nil
}

// GetLatestSlowLogDate returns the latest slow query log date.
func (s *Store) GetLatestSlowLogDate(ctx context.Context, instanceUID int) (*time.Time, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	result, err := s.getLatestSlowLogDateImpl(ctx, tx, instanceUID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (*Store) getLatestSlowLogDateImpl(ctx context.Context, tx *Tx, instanceUID int) (*time.Time, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			MAX(log_date_ts)
		FROM
			slow_query
		WHERE
			instance_id = $1`,
		instanceUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result *time.Time
	for rows.Next() {
		var logDate sql.NullInt32
		if err := rows.Scan(&logDate); err != nil {
			return nil, err
		}
		if logDate.Valid {
			t, err := time.Parse("20060102", strconv.Itoa(int(logDate.Int32)))
			if err != nil {
				return nil, err
			}
			result = &t
		} else {
			result = nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
