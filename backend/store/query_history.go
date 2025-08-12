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

type QueryHistoryType string

const (
	// QueryHistoryTypeQuery is the type for query.
	QueryHistoryTypeQuery QueryHistoryType = "QUERY"
	// QueryHistoryTypeExport is the type for export.
	QueryHistoryTypeExport QueryHistoryType = "EXPORT"
)

// QueryHistoryMessage is the API message for query history.
type QueryHistoryMessage struct {
	// Output only fields
	UID       int
	CreatedAt time.Time

	// Related fields
	CreatorUID int
	// ProjectID is the project resource id.
	ProjectID string

	// Domain specific fields
	Type      QueryHistoryType
	Statement string
	Payload   *storepb.QueryHistoryPayload
	// Database is the database resource name, like instances/{instance}/databases/{database}
	Database string
}

// FindQueryHistoryMessage is the API message for finding query histories.
type FindQueryHistoryMessage struct {
	CreatorUID *int
	ProjectID  *string
	// Instance is the instance resource name like instances/{instance}.
	Instance *string
	// Database is database resource name like instances/{instance}/databases/{database}.
	Database *string
	Type     *QueryHistoryType

	Limit  *int
	Offset *int
	Filter *ListResourceFilter
}

// CreateQueryHistory creates the query history.
func (s *Store) CreateQueryHistory(ctx context.Context, create *QueryHistoryMessage) (*QueryHistoryMessage, error) {
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, `
		INSERT INTO query_history (
			creator_id,
			project_id,
			database,
			statement,
			type,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id,
			created_at
	`,
		create.CreatorUID,
		create.ProjectID,
		create.Database,
		create.Statement,
		create.Type,
		payload,
	).Scan(
		&create.UID,
		&create.CreatedAt,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return create, nil
}

// ListQueryHistories lists the query history.
func (s *Store) ListQueryHistories(ctx context.Context, find *FindQueryHistoryMessage) ([]*QueryHistoryMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if filter := find.Filter; filter != nil {
		where = append(where, filter.Where)
		args = append(args, filter.Args...)
	}

	if v := find.CreatorUID; v != nil {
		where, args = append(where, fmt.Sprintf("query_history.creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("query_history.project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Instance; v != nil {
		where, args = append(where, fmt.Sprintf("query_history.database LIKE $%d", len(args)+1)), append(args, *v)
	} else if v := find.Database; v != nil {
		where, args = append(where, fmt.Sprintf("query_history.database = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("query_history.type = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		SELECT
			query_history.id,
			query_history.creator_id,
			query_history.created_at,
			query_history.project_id,
			query_history.database,
			query_history.statement,
			query_history.type,
			query_history.payload
		FROM query_history
		WHERE %s
		ORDER BY id DESC
	`, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	var queryHistories []*QueryHistoryMessage
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		queryHistory := &QueryHistoryMessage{}
		var payloadStr string
		if err := rows.Scan(
			&queryHistory.UID,
			&queryHistory.CreatorUID,
			&queryHistory.CreatedAt,
			&queryHistory.ProjectID,
			&queryHistory.Database,
			&queryHistory.Statement,
			&queryHistory.Type,
			&payloadStr,
		); err != nil {
			return nil, err
		}

		var payload storepb.QueryHistoryPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return nil, err
		}
		queryHistory.Payload = &payload

		queryHistories = append(queryHistories, queryHistory)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return queryHistories, nil
}
