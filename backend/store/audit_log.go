package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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
	query := `
		INSERT INTO audit_log (payload) VALUES ($1)
	`

	p, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal payload")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, p); err != nil {
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
	if v := find.Project; v != nil {
		where = append(where, fmt.Sprintf("payload->>'parent' = $%d", len(args)+1))
		args = append(args, *v)
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			created_at,
			payload
		FROM audit_log
		WHERE %s`, strings.Join(where, " AND "))

	if len(find.OrderByKeys) > 0 {
		orderBy := []string{}
		for _, v := range find.OrderByKeys {
			orderBy = append(orderBy, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
		}
		query += fmt.Sprintf(" ORDER BY %s", strings.Join(orderBy, ", "))
	} else {
		query += " ORDER BY id DESC"
	}
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
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
