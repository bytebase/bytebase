package mysql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	parser "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// querySpanExtractor is the extractor to extract the query span from a single statement.
type querySpanExtractor struct {
	ctx         context.Context
	connectedDB string

	f base.GetDatabaseMetadataFunc

	// ctes is the common table expressions, which is used to record the cte schema.
	// It should be reset to original state while quit the nested cte. For example:
	// WITH cte_1 AS (WITH cte_2 AS (SELECT * FROM t) SELECT * FROM cte_2) SELECT * FROM cte_2;
	// MySQL will throw a runtime error: (1146, "Table 'junk.cte_2' doesn't exist"), the `junk` is the database name.
	ctes []*base.PseudoTable
}

func (q *querySpanExtractor) shrinkCtes(originalLength int) {
	q.ctes = q.ctes[:originalLength]
}

func newQuerySpanExtractor(connectedDB string, f base.GetDatabaseMetadataFunc) *querySpanExtractor {
	return &querySpanExtractor{
		connectedDB: connectedDB,
		f:           f,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	q.ctx = ctx

	list, err := ParseMySQL(stmt)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	}
	if len(list) == 1 {
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", len(list))
	}

	// We assumes the caller had handled the statement type case,
	// so we only need to handle the determined statement type here.
	// In order to decrease the maintenance cost, we use listener to handle
	// the select statement precisely instead of using type switch.
	listener := newSelectOnlyListener(q)
	antlr.ParseTreeWalkerDefault.Walk(listener, list[0].Tree)

	return listener.querySpan, listener.err
}

// extractSelectStatement extracts the table source from the select statement.
// It regards the result of the select statement as the pseudo table.
func (q *querySpanExtractor) extractSelectStatement(ctx mysql.ISelectStatementContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return base.NewPseudoTable("", []base.QuerySpanResult{}), nil
	}

	switch {
	case ctx.QueryExpression() != nil:
	case ctx.QueryExpressionParens() != nil:

	}
}

func (q *querySpanExtractor) extractQueryExpression(ctx mysql.IQueryExpressionContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return base.NewPseudoTable("", []base.QuerySpanResult{}), nil
	}

	if ctx.WithClause() != nil {
		originalCtesLength := len(q.ctes)
		defer func() {
			q.shrinkCtes(originalCtesLength)
		}()
		recursive := ctx.WithClause().RECURSIVE_SYMBOL() != nil
		for 
	}
}

// selectOnlyListener is the listener to listen the top level select statement only.
type selectOnlyListener struct {
	*parser.BaseMySQLParserListener

	// The only misson of the listener is the find the precise select statement.
	// All the eval work will be handled by the querySpanExtractor.
	extractor *querySpanExtractor
	querySpan *base.QuerySpan
	err       error
}

func newSelectOnlyListener(extractor *querySpanExtractor) *selectOnlyListener {
	return &selectOnlyListener{
		extractor: extractor,
		querySpan: &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: make(base.SourceColumnSet),
		},
	}
}

func (s *selectOnlyListener) EnterSelectStatement(ctx *parser.SelectStatementContext) {
	parent := ctx.GetParent()
	if parent == nil {
		return
	}

	if _, ok := parent.(*mysql.SimpleStatementContext); !ok {
		return
	}

	return
}
