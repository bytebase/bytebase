package store

import (
	"context"

	"github.com/bytebase/bytebase/metric"
)

// CountMemberGroupByRoleAndStatus counts the number of member and group by role and status.
// Used by the metric collector.
func (s *Store) CountMemberGroupByRoleAndStatus(ctx context.Context) ([]*metric.MemberCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT role, status, row_status, COUNT(*)
		FROM member
		GROUP BY role, status, row_status`,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var res []*metric.MemberCountMetric
	for rows.Next() {
		var metric metric.MemberCountMetric
		if err := rows.Scan(&metric.Role, &metric.Status, &metric.RowStatus, &metric.Count); err != nil {
			return nil, FormatError(err)
		}
		res = append(res, &metric)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}
	return res, nil
}
