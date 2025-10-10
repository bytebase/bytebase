package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
	// RiskSourceDatabaseDataExport is for database data export.
	RiskSourceDatabaseDataExport RiskSource = "bb.risk.database.data.export"
	// RiskSourceDatabaseCreate is for creating databases.
	RiskSourceDatabaseCreate RiskSource = "bb.risk.database.create"
	// RiskRequestRole is for requesting role.
	RiskRequestRole RiskSource = "bb.risk.request.role"
)

// RiskMessage is the message for risks.
type RiskMessage struct {
	Source     RiskSource
	Level      storepb.RiskLevel
	Name       string
	Active     bool
	Expression *expr.Expr // *v1alpha1.ParsedExpr

	// Output only
	ID int64
}

// UpdateRiskMessage is the message for updating a risk.
type UpdateRiskMessage struct {
	Name       *string
	Active     *bool
	Level      *storepb.RiskLevel
	Expression *expr.Expr
	Source     *RiskSource
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
			expression
		FROM risk
		WHERE id = $1`

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var risk RiskMessage
	var expressionBytes []byte
	var levelStr string
	if err := tx.QueryRowContext(ctx, query, id).Scan(
		&risk.ID,
		&risk.Source,
		&levelStr,
		&risk.Name,
		&risk.Active,
		&expressionBytes,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	var expression expr.Expr // v1alpha1.ParsedExpr
	if err := common.ProtojsonUnmarshaler.Unmarshal(expressionBytes, &expression); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal")
	}
	risk.Expression = &expression
	// Convert database string to enum
	if value, ok := storepb.RiskLevel_value[levelStr]; ok {
		risk.Level = storepb.RiskLevel(value)
	} else {
		risk.Level = storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	return &risk, nil
}

// ListRisks lists risks.
// returned risks are sorted by source, level DESC, id.
func (s *Store) ListRisks(ctx context.Context) ([]*RiskMessage, error) {
	if v, ok := s.risksCache.Get(0); ok && s.enableCache {
		return v, nil
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
		ORDER BY source, level DESC, id
	`

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
		var levelStr string
		if err := rows.Scan(
			&risk.ID,
			&risk.Source,
			&levelStr,
			&risk.Name,
			&risk.Active,
			&expressionBytes,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		var expression expr.Expr
		if err := common.ProtojsonUnmarshaler.Unmarshal(expressionBytes, &expression); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal")
		}
		risk.Expression = &expression
		// Convert database string to enum
		if value, ok := storepb.RiskLevel_value[levelStr]; ok {
			risk.Level = storepb.RiskLevel(value)
		} else {
			risk.Level = storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED
		}

		risks = append(risks, &risk)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows.Err() is not nil")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	s.risksCache.Add(0, risks)
	return risks, nil
}

// CreateRisk creates a risk.
func (s *Store) CreateRisk(ctx context.Context, risk *RiskMessage) (*RiskMessage, error) {
	query := `
		INSERT INTO risk (
			source,
			level,
			name,
			active,
			expression
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	expressionBytes, err := protojson.Marshal(risk.Expression)
	if err != nil {
		return nil, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var id int64
	// Convert enum to database string using .String() method
	if err := tx.QueryRowContext(ctx, query, risk.Source, risk.Level.String(), risk.Name, risk.Active, string(expressionBytes)).Scan(&id); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	s.risksCache.Remove(0)
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
func (s *Store) UpdateRisk(ctx context.Context, patch *UpdateRiskMessage, id int64) (*RiskMessage, error) {
	set, args := []string{}, []any{}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Active; v != nil {
		set, args = append(set, fmt.Sprintf("active = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Level; v != nil {
		// Convert enum to database string using .String() method
		set, args = append(set, fmt.Sprintf("level = $%d", len(args)+1)), append(args, v.String())
	}
	if v := patch.Source; v != nil {
		set, args = append(set, fmt.Sprintf("source = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Expression; v != nil {
		expressionBytes, err := protojson.Marshal(patch.Expression)
		if err != nil {
			return nil, err
		}
		set, args = append(set, fmt.Sprintf("expression = $%d", len(args)+1)), append(args, string(expressionBytes))
	}
	args = append(args, id)

	tx, err := s.GetDB().BeginTx(ctx, nil)
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

	s.risksCache.Remove(0)
	return s.GetRisk(ctx, id)
}

func (s *Store) DeleteRisk(ctx context.Context, id int64) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM risk WHERE id = $1`, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	s.risksCache.Remove(0)
	return nil
}
