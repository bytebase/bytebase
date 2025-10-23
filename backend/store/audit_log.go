package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type AuditLogMethod string

// The methods other than v1 api.
const AuditLogMethodProjectRepositoryPush AuditLogMethod = "bb.project.repository.push"

func (m AuditLogMethod) String() string {
	return string(m)
}

type AuditLog struct {
	ID        int
	CreatedAt time.Time
	Payload   *storepb.AuditLog
}

type AuditLogFind struct {
	Project     *string
	Filter      *ListResourceFilter
	Limit       *int
	Offset      *int
	OrderByKeys []OrderByKey
}

func (s *Store) CreateAuditLog(ctx context.Context, payload *storepb.AuditLog) error {
	p, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal payload")
	}

	q := qb.Q().Space("INSERT INTO audit_log (payload) VALUES (?)", p)
	sql, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to create audit log")
	}
	return nil
}

func (s *Store) SearchAuditLogs(ctx context.Context, find *AuditLogFind) ([]*AuditLog, error) {
	q := qb.Q().Space(`
		SELECT
			id,
			created_at,
			payload
		FROM audit_log
		WHERE TRUE
	`)

	if v := find.Filter; v != nil {
		// Convert $1, $2, etc. to ? for qb
		q.And(ConvertDollarPlaceholders(v.Where), v.Args...)
	}
	if v := find.Project; v != nil {
		q.And("payload->>'parent' = ?", *v)
	}

	if len(find.OrderByKeys) > 0 {
		orderBy := []string{}
		for _, v := range find.OrderByKeys {
			orderBy = append(orderBy, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
		}
		q.Space(fmt.Sprintf("ORDER BY %s", strings.Join(orderBy, ", ")))
	} else {
		q.Space("ORDER BY id DESC")
	}
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, sql, args...)
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
			&l.CreatedAt,
			&payload,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, l.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal payload")
		}

		logs = append(logs, l)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}

	return logs, nil
}
