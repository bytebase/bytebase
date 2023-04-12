package store

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ListSlowQueryMessage is the message to list slow query logs.
type ListSlowQueryMessage struct {
	InstanceUID *int
	DatabaseUID *int
	// List slow query logs in [StartLogDate, EndLogDate).
	StartLogDate *time.Time
	EndLogDate   *time.Time
}

// ListSlowQuery lists slow query logs.
func (s *Store) ListSlowQuery(ctx context.Context, list *ListSlowQueryMessage) ([]*v1pb.SlowQueryLog, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	slowQueryLog, err := s.listSlowQueryImpl(ctx, tx, list)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return slowQueryLog, nil
}

func (*Store) listSlowQueryImpl(ctx context.Context, tx *Tx, list *ListSlowQueryMessage) ([]*v1pb.SlowQueryLog, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := list.InstanceUID; v != nil {
		where, args = append(where, fmt.Sprintf("instance_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := list.DatabaseUID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := list.StartLogDate; v != nil {
		where, args = append(where, fmt.Sprintf("log_date_ts >= $%d", len(args)+1)), append(args, v.Format("20060102"))
	}
	if v := list.EndLogDate; v != nil {
		where, args = append(where, fmt.Sprintf("log_date_ts < $%d", len(args)+1)), append(args, v.Format("20060102"))
	}

	query := fmt.Sprintf(`
		SELECT
			instance_id,
			database_id,
			log_date_ts,
			slow_query_statistics
		FROM slow_query
		WHERE (%s)
	`, strings.Join(where, " AND "))

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logMap := make(map[string]*v1pb.SlowQueryLog)
	for rows.Next() {
		var instanceUID int
		var databaseUID sql.NullInt32
		var logDate int
		var logBytes []byte
		if err := rows.Scan(
			&instanceUID,
			&databaseUID,
			&logDate,
			&logBytes,
		); err != nil {
			return nil, err
		}

		var slowLog storepb.SlowQueryStatistics
		if err := protojson.Unmarshal(logBytes, &slowLog); err != nil {
			return nil, err
		}

		for _, item := range slowLog.Items {
			if log, exists := logMap[item.SqlFingerprint]; exists {
				log.Statistics.Count += item.Count
				if log.Statistics.LatestLogTime.AsTime().Before(item.LatestLogTime.AsTime()) {
					log.Statistics.LatestLogTime = item.LatestLogTime
				}
				for _, sample := range item.Samples {
					details := &v1pb.SlowQueryDetails{
						StartTime:    sample.StartTime,
						QueryTime:    sample.QueryTime,
						LockTime:     sample.LockTime,
						RowsSent:     sample.RowsSent,
						RowsExamined: sample.RowsExamined,
						SqlText:      sample.SqlText,
					}

					if len(log.Statistics.Samples) < db.SlowQueryMaxSamplePerFingerprint {
						log.Statistics.Samples = append(log.Statistics.Samples, details)
					} else {
						// Use Reservoir Sampling to sample slow logs.
						pos := rand.Intn(len(log.Statistics.Samples))
						log.Statistics.Samples[pos] = details
					}
				}
			} else {
				logMap[item.SqlFingerprint] = &v1pb.SlowQueryLog{
					Statistics: &v1pb.SlowQueryStatistics{
						SqlFingerprint: item.SqlFingerprint,
						Count:          item.Count,
						LatestLogTime:  item.LatestLogTime,
						Samples: func() []*v1pb.SlowQueryDetails {
							var details []*v1pb.SlowQueryDetails
							for _, sample := range item.Samples {
								details = append(details, &v1pb.SlowQueryDetails{
									StartTime:    sample.StartTime,
									QueryTime:    sample.QueryTime,
									LockTime:     sample.LockTime,
									RowsSent:     sample.RowsSent,
									RowsExamined: sample.RowsExamined,
									SqlText:      sample.SqlText,
								})
							}
							return details
						}(),
					},
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []*v1pb.SlowQueryLog
	for key := range logMap {
		result = append(result, calculateStatistics(logMap[key]))
	}

	return result, nil
}

func calculateStatistics(log *v1pb.SlowQueryLog) *v1pb.SlowQueryLog {
	var totalQueryTime time.Duration
	var totalSent, totalExamined int64

	for _, sample := range log.Statistics.Samples {
		totalQueryTime += sample.QueryTime.AsDuration()
		totalSent += int64(sample.RowsSent)
		totalExamined += int64(sample.RowsExamined)
	}

	log.Statistics.AverageQueryTime = durationpb.New(totalQueryTime / time.Duration(len(log.Statistics.Samples)))
	log.Statistics.AverageRowsSent = int32(totalSent / int64(len(log.Statistics.Samples)))
	log.Statistics.AverageRowsExamined = int32(totalExamined / int64(len(log.Statistics.Samples)))

	nightyFifthPos := int(float64(len(log.Statistics.Samples)) * 0.95)

	sort.Slice(log.Statistics.Samples, func(i, j int) bool {
		return log.Statistics.Samples[i].RowsSent < log.Statistics.Samples[j].RowsSent
	})
	log.Statistics.NightyFifthPercentileRowsSent = log.Statistics.Samples[nightyFifthPos].RowsSent

	sort.Slice(log.Statistics.Samples, func(i, j int) bool {
		return log.Statistics.Samples[i].RowsExamined < log.Statistics.Samples[j].RowsExamined
	})
	log.Statistics.NightyFifthPercentileRowsExamined = log.Statistics.Samples[nightyFifthPos].RowsExamined

	sort.Slice(log.Statistics.Samples, func(i, j int) bool {
		return log.Statistics.Samples[i].QueryTime.AsDuration() < log.Statistics.Samples[j].QueryTime.AsDuration()
	})
	log.Statistics.NightyFifthPercentileQueryTime = log.Statistics.Samples[nightyFifthPos].QueryTime

	return log
}

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
