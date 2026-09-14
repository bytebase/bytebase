package advisor

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ExplainBudget caps the statements a row limit rule EXPLAINs at common.MaximumLintExplainSize and
// counts the statements it leaves unchecked.
type ExplainBudget struct {
	used         int
	skipped      int
	firstSkipped *storepb.Position
}

// Spend reports whether the statement at position may be explained, recording it as skipped
// otherwise.
func (b *ExplainBudget) Spend(position *storepb.Position) bool {
	if b.used < common.MaximumLintExplainSize {
		b.used++
		return true
	}
	if b.skipped == 0 {
		b.firstSkipped = position
	}
	b.skipped++
	return false
}

// AppendSkippedAdvice appends one warning covering the skipped statements, if any. It is a warning
// whatever the rule's level.
func (b *ExplainBudget) AppendSkippedAdvice(adviceList []*storepb.Advice, title string, adviceCode code.Code) []*storepb.Advice {
	if b.skipped == 0 {
		return adviceList
	}
	return append(adviceList, &storepb.Advice{
		Status:        storepb.Advice_WARNING,
		Code:          adviceCode.Int32(),
		Title:         title,
		Content:       fmt.Sprintf("Only the first %d statements were estimated; %d more were not checked against the row limit.", common.MaximumLintExplainSize, b.skipped),
		StartPosition: b.firstSkipped,
	})
}

type QueryContext struct {
	TenantMode    bool
	PreExecutions []string
}

// Query runs the EXPLAIN or SELECT statements for advisors.
func Query(ctx context.Context, qCtx QueryContext, connection *sql.DB, engine storepb.Engine, statement string) ([]any, error) {
	tx, err := connection.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if engine == storepb.Engine_POSTGRES && qCtx.TenantMode {
		const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
		var owner string
		if err := tx.QueryRowContext(ctx, query).Scan(&owner); err != nil {
			return nil, err
		}
		// Use SET SESSION ROLE to match the execution logic in backend/plugin/db/pg/pg.go
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET SESSION ROLE '%s';", owner)); err != nil { // NOSONAR(go:S2077) owner is from pg_roles system catalog, not user input
			return nil, err
		}
	}

	for _, preExec := range qCtx.PreExecutions {
		if preExec != "" {
			if _, err := tx.ExecContext(ctx, preExec); err != nil {
				return nil, errors.Wrapf(err, "failed to execute pre-execution: %s", preExec)
			}
		}
	}

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	colCount := len(columnTypes)

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	data := []any{}
	for rows.Next() {
		scanArgs := make([]any, colCount)
		for i, v := range columnTypeNames {
			// TODO(steven need help): Consult a common list of data types from database driver documentation. e.g. MySQL,PostgreSQL.
			switch v {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
			case "INT", "INTEGER":
				scanArgs[i] = new(sql.NullInt64)
			case "FLOAT":
				scanArgs[i] = new(sql.NullFloat64)
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		rowData := []any{}
		for i := range columnTypes {
			if v, ok := (scanArgs[i]).(*sql.NullBool); ok && v.Valid {
				rowData = append(rowData, v.Bool)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullString); ok && v.Valid {
				rowData = append(rowData, v.String)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt64); ok && v.Valid {
				rowData = append(rowData, v.Int64)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt32); ok && v.Valid {
				rowData = append(rowData, v.Int32)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullFloat64); ok && v.Valid {
				rowData = append(rowData, v.Float64)
				continue
			}
			// If none of them match, set nil to its value.
			rowData = append(rowData, nil)
		}

		data = append(data, rowData)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return []any{columnNames, columnTypeNames, data}, nil
}

// ContainsDDL checks if any of the parsed statements is a DDL statement.
// When DDLs and DMLs are mixed, DML statements often reference objects created
// by DDL statements, causing false positives in dry run checks.
// BYT-8855
func ContainsDDL(engine storepb.Engine, parsedStatements []base.ParsedStatement) bool {
	asts := base.ExtractASTs(parsedStatements)
	types, err := base.GetStatementTypes(engine, asts)
	if err != nil {
		return false
	}
	for _, t := range types {
		switch t {
		case storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
			storepb.StatementType_INSERT,
			storepb.StatementType_UPDATE,
			storepb.StatementType_DELETE,
			storepb.StatementType_MERGE:
		default:
			return true
		}
	}
	return false
}

func DatabaseExists(ctx context.Context, checkCtx Context, database string) bool {
	if checkCtx.ListDatabaseNamesFunc == nil {
		return false
	}

	names, err := checkCtx.ListDatabaseNamesFunc(ctx, checkCtx.InstanceID)
	if err != nil {
		slog.Debug("failed to list databases", slog.String("instance", checkCtx.InstanceID), log.BBError(err))
		return false
	}

	for _, name := range names {
		if name == database {
			return true
		}
	}

	return false
}
