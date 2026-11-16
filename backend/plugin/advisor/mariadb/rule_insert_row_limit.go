package mariadb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqldriver "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

var (
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, &InsertRowLimitAdvisor{})
}

// InsertRowLimitAdvisor is the advisor checking for insert row limit. omni
// (mariadb/v1review.StatementInsertRowLimitProbes) picks the INSERTs whose rows
// the database must estimate; this adapter EXPLAINs and estimates them, and
// omni (StatementInsertRowLimit) judges every INSERT.
type InsertRowLimitAdvisor struct {
}

// Check checks for insert row limit.
func (*InsertRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	maxRow := int(numberPayload.Number)
	driver := checkCtx.Driver
	title := checkCtx.Rule.Type.String()
	var explains advisor.ExplainBudget

	nodes, stmts := mariadbNodes(checkCtx.ParsedStatements)
	probes := make(map[int]v1review.Probe)
	if driver != nil {
		for _, i := range mariadbv1review.StatementInsertRowLimitProbes(nodes) {
			if !explains.Spend(statementStartPosition(stmts[i])) {
				continue
			}
			text := dryRunText(stmts[i].Text)
			plan, err := mysqldriver.ExplainJSON(ctx, driver, text)
			if err != nil {
				probes[i] = v1review.Probe{Error: err.Error()}
				continue
			}
			rowCount, err := mysqldriver.EstimateAffectedRowsMariaDB(ctx, driver, nodes[i], text, plan)
			if err != nil {
				probes[i] = v1review.Probe{CountError: err.Error()}
				continue
			}
			probes[i] = v1review.Probe{Rows: rowCount}
		}
	}

	var advice []*storepb.Advice
	for _, f := range mariadbv1review.StatementInsertRowLimit(nodes, probes, maxRow) {
		stmt := stmts[f.Statement]
		text := dryRunText(stmt.Text)
		adviceCode, content := code.InsertTooManyRows, ""
		switch f.Code {
		case v1review.DryRunFailed:
			content = fmt.Sprintf("\"%s\" dry runs failed: %s", text, f.Value)
		case v1review.RowCountFailed:
			adviceCode, content = code.Internal, fmt.Sprintf("failed to get row count for \"%s\": %s", text, f.Value)
		case v1review.RowLimitExceeded:
			content = fmt.Sprintf("\"%s\" inserts %s rows. The count exceeds %d.", text, f.Value, maxRow)
		default:
			continue
		}
		advice = append(advice, &storepb.Advice{
			Status:        level,
			Code:          adviceCode.Int32(),
			Title:         title,
			Content:       content,
			StartPosition: statementStartPosition(stmt),
		})
	}
	return explains.AppendSkippedAdvice(advice, title, code.InsertTooManyRows), nil
}

// mariadbNodes returns the statements that parsed to an omni node, with their
// nodes, index for index.
func mariadbNodes(parsed []base.ParsedStatement) ([]ast.Node, []base.ParsedStatement) {
	var nodes []ast.Node
	var stmts []base.ParsedStatement
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := mariadbparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}
	return nodes, stmts
}

// statementStartPosition is the position of the statement's first
// non-blank character.
func statementStartPosition(stmt base.ParsedStatement) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(mariadbparser.ByteOffsetToRunePosition(stmt.Text, contentStartIndex(stmt.Text)).Line))
}

func contentStartIndex(text string) int {
	for i, c := range text {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return i
		}
	}
	return 0
}
