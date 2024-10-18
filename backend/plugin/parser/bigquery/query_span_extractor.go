package bigquery

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/google-sql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type querySpanExtractor struct {
	ctx             context.Context
	defaultDatabase string

	gCtx base.GetQuerySpanContext
}

func newQuerySpanExtractor(defaultDatabase string, gCtx base.GetQuerySpanContext, _ bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase: defaultDatabase,
		gCtx:            gCtx,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	parseResults, err := ParseBigQuerySQL(stmt)
	if err != nil {
		return nil, err
	}
	tree := parseResults.Tree
	q.ctx = ctx
	accessTables, err := getAccessTables(q.defaultDatabase, tree)
	if err != nil {
		return nil, err
	}
	allSystems, mixed := isMixedQuery(accessTables)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}
	if allSystems {
		return &base.QuerySpan{
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	return &base.QuerySpan{
		SourceColumns: accessTables,
	}, nil
}

func getQuerySpanResult(tree *parser.StmtContext) ([]base.QuerySpanResult, error) {
	return nil, nil
}

func getAccessTables(defaultDatabase string, tree antlr.Tree) (base.SourceColumnSet, error) {
	l := newAccessTableListener(defaultDatabase)
	result := make(base.SourceColumnSet)
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if l.err != nil {
		return nil, l.err
	}
	result, _ = base.MergeSourceColumnSet(result, l.sourceColumnSet)
	return result, nil
}

type accessTableListener struct {
	*parser.BaseGoogleSQLParserListener

	currentDatabase string
	sourceColumnSet base.SourceColumnSet
	err             error
}

func newAccessTableListener(currentDatabase string) *accessTableListener {
	return &accessTableListener{
		currentDatabase: currentDatabase,
		sourceColumnSet: make(base.SourceColumnSet),
	}
}

func (l *accessTableListener) EnterTable_path_expression(ctx *parser.Table_path_expressionContext) {
	if l.err != nil {
		return
	}
	// TODO(zp): Handle other unusual table path expression.
	exprBase := ctx.Table_path_expression_base()
	slashedOrDashedPathExpr := exprBase.Maybe_slashed_or_dashed_path_expression()
	if slashedOrDashedPathExpr == nil {
		l.err = errors.Errorf("unsupported table path expression: %s", ctx.GetText())
		return
	}

	dashedPathExpr := slashedOrDashedPathExpr.Maybe_dashed_path_expression()
	if dashedPathExpr == nil {
		l.err = errors.Errorf("unsupported slashed table path expression: %s", ctx.GetText())
		return
	}

	pathExpr := dashedPathExpr.Path_expression()
	if pathExpr == nil {
		l.err = errors.Errorf("unsupported dashed table path expression: %s", ctx.GetText())
	}

	// Table name syntax: https://cloud.google.com/bigquery/docs/reference/standard-sql/lexical
	// Most of the time, the syntax can be [project_id.][dataset_id.]table_id.
	// One difference is that the user access INFORMATION_SCHEMA in dataset,
	// the syntax would be [project_id.]([region_id.]|[dataset_id.])INFORMATION_SCHEMA.VIEW_NAME.
	// In this case, we treat the INFORMATION_SCHEMA as schema name.
	allIdentifiers := pathExpr.AllIdentifier()
	if len(allIdentifiers) == 0 {
		return
	}

	columnSource := base.ColumnResource{
		Database: l.currentDatabase,
	}
	lastIdentifier := allIdentifiers[len(allIdentifiers)-1]
	tableName := unquoteIdentifier(lastIdentifier)
	columnSource.Table = tableName

	if len(allIdentifiers) >= 2 {
		identifier := unquoteIdentifier(allIdentifiers[len(allIdentifiers)-2])
		if strings.EqualFold(identifier, "INFORMATION_SCHEMA") {
			columnSource.Schema = identifier
		} else {
			columnSource.Database = identifier
		}
	}

	l.sourceColumnSet[columnSource] = true
}

func unquoteIdentifier(identifier parser.IIdentifierContext) string {
	if len(identifier.GetText()) >= 3 && strings.HasPrefix(identifier.GetText(), "`") && strings.HasSuffix(identifier.GetText(), "`") {
		return identifier.GetText()[1 : len(identifier.GetText())-1]
	}
	return identifier.GetText()
}

func isMixedQuery(m base.SourceColumnSet) (bool, bool) {
	hasSystem, hasUser := false, false
	for table := range m {
		if isSystemResource(table) {
			hasSystem = true
		} else {
			hasUser = true
		}
	}

	if hasSystem && hasUser {
		return false, true
	}

	return !hasUser && hasSystem, false
}

func isSystemResource(resource base.ColumnResource) bool {
	if strings.EqualFold(resource.Schema, "INFORMATION_SCHEMA") {
		return true
	}
	return false
}
