package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// RiskSource is the source of the risk.
type RiskSource string

const (
	// RiskSourceUnknown is for unknown source.
	RiskSourceUnknown RiskSource = ""
	// RiskSourceDatabaseSchemaUpdate is for DDL.
	RiskSourceDatabaseSchemaUpdate RiskSource = "bb.risk.database.schema.update"
	// RiskSourceDatabaseDataUpdate is for DML.
	RiskSourceDatabaseDataUpdate RiskSource = "bb.risk.database.data.update"
	// RiskSourceDatabaseCreate is for creating databases.
	RiskSourceDatabaseCreate RiskSource = "bb.risk.database.create"
	// RiskRequestQuery is for requesting query grant.
	RiskRequestQuery RiskSource = "bb.risk.request.query"
	// RiskRequestExport is for requesting export grant.
	RiskRequestExport RiskSource = "bb.risk.request.export"
)

// RiskMessage is the message for risks.
type RiskMessage struct {
	Source     RiskSource
	Level      int64
	Name       string
	Active     bool
	Expression *expr.Expr // *v1alpha1.ParsedExpr

	// Output only
	ID      int64
	Deleted bool
}

// UpdateRiskMessage is the message for updating a risk.
type UpdateRiskMessage struct {
	Name       *string
	Active     *bool
	Level      *int64
	Expression *expr.Expr
	RowStatus  *api.RowStatus
}

// GetRisk gets a risk.
func (s *Store) GetRisk(ctx context.Context, id int64) (*RiskMessage, error) {
	query := `
		SELECT
			id,
			source,
			level,
			name,
			active,
			expression,
			row_status
		FROM risk
		WHERE id = $1`

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var risk RiskMessage
	var expressionBytes []byte
	var rowStatus api.RowStatus
	if err := tx.QueryRowContext(ctx, query, id).Scan(
		&risk.ID,
		&risk.Source,
		&risk.Level,
		&risk.Name,
		&risk.Active,
		&expressionBytes,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	risk.Deleted = convertRowStatusToDeleted(string(rowStatus))

	var expression expr.Expr // v1alpha1.ParsedExpr
	if err := protojson.Unmarshal(expressionBytes, &expression); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal")
	}
	risk.Expression = &expression

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	return &risk, nil
}

// ListRisks lists risks.
// returned risks are sorted by source, level DESC, id.
func (s *Store) ListRisks(ctx context.Context) ([]*RiskMessage, error) {
	if risks, ok := s.risksCache.Load(0); ok {
		return risks.([]*RiskMessage), nil
	}

	query := `
		SELECT
			id,
			source,
			level,
			name,
			active,
			expression
		FROM risk
		WHERE row_status = 'NORMAL'
		ORDER BY source, level DESC, id
	`

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin tx")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query %s", query)
	}
	defer rows.Close()

	var risks []*RiskMessage

	for rows.Next() {
		var risk RiskMessage
		var expressionBytes []byte
		if err := rows.Scan(
			&risk.ID,
			&risk.Source,
			&risk.Level,
			&risk.Name,
			&risk.Active,
			&expressionBytes,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		var expression expr.Expr
		if err := protojson.Unmarshal(expressionBytes, &expression); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal")
		}
		risk.Expression = &expression

		risks = append(risks, &risk)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows.Err() is not nil")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	s.risksCache.Store(0, risks)
	return risks, nil
}

// CreateRisk creates a risk.
func (s *Store) CreateRisk(ctx context.Context, risk *RiskMessage, creatorID int) (*RiskMessage, error) {
	query := `
		INSERT INTO risk (
			creator_id,
			updater_id,
			source,
			level,
			name,
			active,
			expression
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	expressionBytes, err := protojson.Marshal(risk.Expression)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var id int64
	if err := tx.QueryRowContext(ctx, query, creatorID, creatorID, risk.Source, risk.Level, risk.Name, risk.Active, string(expressionBytes)).Scan(&id); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	s.risksCache.Delete(0)
	return &RiskMessage{
		ID:         id,
		Source:     risk.Source,
		Level:      risk.Level,
		Name:       risk.Name,
		Active:     risk.Active,
		Expression: risk.Expression,
	}, nil
}

// UpdateRisk updates a risk.
func (s *Store) UpdateRisk(ctx context.Context, patch *UpdateRiskMessage, id int64, updaterID int) (*RiskMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{updaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Active; v != nil {
		set, args = append(set, fmt.Sprintf("active = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Level; v != nil {
		set, args = append(set, fmt.Sprintf("level = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Expression; v != nil {
		expressionBytes, err := protojson.Marshal(patch.Expression)
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("expression = $%d", len(args)+1)), append(args, string(expressionBytes))
	}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, id)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE risk
		SET `+strings.Join(set, ", ")+`
		WHERE id = `+fmt.Sprintf("$%d", len(args)),
		args...,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	s.risksCache.Delete(0)
	return s.GetRisk(ctx, id)
}
