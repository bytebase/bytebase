package store

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type AuditLog struct {
	ID        int
	CreatedTs int64
	Payload   *storepb.AuditLog
}

type AuditLogFind struct {
	Filter *AuditLogFilter
	Limit  *int
	Offset *int
}

type AuditLogFilter struct {
	Args  []any
	Where string
}

func (s *Store) CreateAuditLog(ctx context.Context, payload *storepb.AuditLog) error {
	if !s.profile.DevelopmentAudit {
		return nil
	}
	query := `
		INSERT INTO audit_log (payload) VALUES ($1)
	`

	p, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal payload")
	}

	if _, err := s.db.db.ExecContext(ctx, query, p); err != nil {
		return errors.Wrapf(err, "failed to create audit log")
	}
	return nil
}

func (s *Store) SearchAuditLogs(ctx context.Context, find *AuditLogFind) ([]*AuditLog, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.Filter; v != nil {
		where = append(where, v.Where)
		args = append(args, v.Args...)
	}

	limitOffsetClause := ""
	if v := find.Limit; v != nil {
		limitOffsetClause += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		limitOffsetClause += fmt.Sprintf(" OFFSET %d", *v)
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			created_ts,
			payload
		FROM audit_log
		WHERE %s
		%s
	`, strings.Join(where, " AND "), limitOffsetClause)

	rows, err := s.db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query context")
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		l := &AuditLog{
			Payload: &storepb.AuditLog{},
		}
		var payload []byte

		if err := rows.Scan(
			&l.ID,
			&l.CreatedTs,
			&payload,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		if err := protojson.Unmarshal(payload, l.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal payload")
		}

		logs = append(logs, l)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}

	return logs, nil
}
