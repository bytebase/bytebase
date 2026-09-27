package oracle

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/utils"
)

// dynamicSamplingHint makes the optimizer sample the tables while it plans the statement, so its
// estimates account for skewed columns without histograms and for correlated predicates. Unlike a
// DYNAMIC_SAMPLING hint, which covers only its own query block, OPT_PARAM covers every block.
const dynamicSamplingHint = "OPT_PARAM('optimizer_dynamic_sampling' 11)"

// CountAffectedRows returns the optimizer's estimate of the rows an UPDATE, DELETE, INSERT, or MERGE
// changes, planned with dynamic sampling.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	// Older Oracle releases reject a trailing semicolon with ORA-00911.
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon)
	randNum, err := rand.Int(rand.Reader, big.NewInt(999))
	if err != nil {
		return 0, errors.Wrapf(err, "failed to generate random statement ID")
	}
	statementID := fmt.Sprintf("%d%d", time.Now().UnixMilli(), randNum.Int64())

	// The default PLAN_TABLE is a session-private temporary table, so explaining,
	// reading, and deleting the plan must share one connection.
	conn, err := d.db.Conn(ctx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get connection")
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, fmt.Sprintf("EXPLAIN PLAN SET STATEMENT_ID = '%s' FOR %s", statementID, withDynamicSamplingHint(statement))); err != nil {
		return 0, errors.Wrapf(err, "failed to explain statement")
	}
	defer func() {
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("DELETE FROM PLAN_TABLE WHERE STATEMENT_ID = '%s'", statementID)); err != nil {
			slog.Warn("failed to delete explain plan rows", log.BBError(err))
		}
	}()

	rows, err := conn.QueryContext(ctx, fmt.Sprintf("SELECT ID, PARENT_ID, OPERATION, CARDINALITY FROM PLAN_TABLE WHERE STATEMENT_ID = '%s' ORDER BY ID", statementID))
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get plan cardinality")
	}
	defer rows.Close()
	var plan []planRow
	for rows.Next() {
		var row planRow
		if err := rows.Scan(&row.id, &row.parentID, &row.operation, &row.cardinality); err != nil {
			return 0, errors.Wrapf(err, "failed to get plan cardinality")
		}
		plan = append(plan, row)
	}
	if err := rows.Err(); err != nil {
		return 0, errors.Wrapf(err, "failed to get plan cardinality")
	}
	return getAffectedRowsFromPlan(plan, isInsertFirst(statement))
}

type planRow struct {
	id          int64
	parentID    sql.NullInt64
	operation   string
	cardinality sql.NullInt64
}

// getAffectedRowsFromPlan returns the cardinality of the statement row (ID 0). For a MERGE it returns
// the cardinality of the rows the MERGE operation reads instead, because with dynamic sampling the
// statement row of a MERGE plan no longer estimates them. A multi-table INSERT ALL can write each row
// it reads to every INTO operation, so its estimate is the statement row times those operations;
// insertFirst marks an INSERT FIRST, which writes each row at most once.
func getAffectedRowsFromPlan(plan []planRow, insertFirst bool) (int64, error) {
	for _, row := range plan {
		if row.operation != "MERGE" {
			continue
		}
		// The first child chain below MERGE passes through views that have no cardinality.
		for parentID := row.id; ; {
			i := slices.IndexFunc(plan, func(r planRow) bool { return r.parentID.Valid && r.parentID.Int64 == parentID })
			if i < 0 || plan[i].id <= parentID {
				return 0, errors.New("the MERGE operation in the plan has no cardinality estimate")
			}
			if plan[i].cardinality.Valid {
				return plan[i].cardinality.Int64, nil
			}
			parentID = plan[i].id
		}
	}
	i := slices.IndexFunc(plan, func(r planRow) bool { return r.id == 0 })
	if i < 0 {
		return 0, errors.New("plan has no statement row")
	}
	if !plan[i].cardinality.Valid {
		return 0, errors.New("plan has no cardinality estimate")
	}
	rows := plan[i].cardinality.Int64
	targets := 0
	for _, row := range plan {
		if row.operation == "INTO" {
			targets++
		}
	}
	if targets > 1 && !insertFirst {
		return common.RoundRows(float64(rows) * float64(targets)), nil
	}
	return rows, nil
}

// isInsertFirst reports whether statement is an INSERT FIRST.
func isInsertFirst(statement string) bool {
	start := skipSpaceAndComments(statement)
	end := start
	for end < len(statement) && isLetter(statement[end]) {
		end++
	}
	if !strings.EqualFold(statement[start:end], "INSERT") {
		return false
	}
	start = end + skipSpaceAndComments(statement[end:])
	end = start
	for end < len(statement) && isLetter(statement[end]) {
		end++
	}
	return strings.EqualFold(statement[start:end], "FIRST")
}

// withDynamicSamplingHint adds dynamicSamplingHint to the hint of an UPDATE, DELETE, INSERT, or MERGE
// statement and returns other statements unchanged.
func withDynamicSamplingHint(statement string) string {
	start := skipSpaceAndComments(statement)
	end := start
	for end < len(statement) && isLetter(statement[end]) {
		end++
	}
	switch strings.ToUpper(statement[start:end]) {
	case "UPDATE", "DELETE", "INSERT", "MERGE":
	default:
		return statement
	}
	// Oracle reads only the first hint after the keyword, so the hint joins an existing one.
	hint := end
	for hint < len(statement) && isSpace(statement[hint]) {
		hint++
	}
	if strings.HasPrefix(statement[hint:], "/*+") || strings.HasPrefix(statement[hint:], "--+") {
		return statement[:hint+3] + " " + dynamicSamplingHint + " " + statement[hint+3:]
	}
	return statement[:end] + " /*+ " + dynamicSamplingHint + " */" + statement[end:]
}

// skipSpaceAndComments returns the offset of the first byte of the statement that is not whitespace
// or part of a comment.
func skipSpaceAndComments(statement string) int {
	i := 0
	for i < len(statement) {
		switch {
		case isSpace(statement[i]):
			i++
		case strings.HasPrefix(statement[i:], "--"):
			end := strings.IndexByte(statement[i:], '\n')
			if end < 0 {
				return len(statement)
			}
			i += end + 1
		case strings.HasPrefix(statement[i:], "/*"):
			end := strings.Index(statement[i+2:], "*/")
			if end < 0 {
				return len(statement)
			}
			i += end + 4
		default:
			return i
		}
	}
	return i
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

func isLetter(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}
