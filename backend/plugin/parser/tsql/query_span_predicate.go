package tsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func (q *querySpanExtractor) extractPredicateColumnFromSearchCondition(
	ctx parser.ISearch_conditionContext,
) error {
	if ctx == nil {
		return nil
	}

	if ctx.Predicate() != nil {
		if err := q.extractPredicateColumnFromPredicate(ctx.Predicate()); err != nil {
			return err
		}
	}

	for _, s := range ctx.AllSearch_condition() {
		if err := q.extractPredicateColumnFromSearchCondition(s); err != nil {
			return err
		}
	}

	return nil
}

func (q *querySpanExtractor) extractPredicateColumnFromPredicate(
	ctx parser.IPredicateContext,
) error {
	if ctx == nil {
		return nil
	}

	if ctx.Subquery() != nil {
		if err := q.extractPredicateColumnFromSubquery(ctx.Subquery()); err != nil {
			return err
		}
	}

	if ctx.Freetext_predicate() != nil {
		if err := q.extractPredicateColumnFromFreetextPredicate(ctx.Freetext_predicate()); err != nil {
			return err
		}
	}

	for _, p := range ctx.AllExpression() {
		if err := q.extractPredicateColumnFromExpression(p); err != nil {
			return err
		}
	}

	if ctx.Expression_list_() != nil {
		for _, e := range ctx.Expression_list_().AllExpression() {
			if err := q.extractPredicateColumnFromExpression(e); err != nil {
				return err
			}
		}
	}

	return nil
}

func (q *querySpanExtractor) extractPredicateColumnFromFreetextPredicate(
	ctx parser.IFreetext_predicateContext,
) error {
	if ctx == nil {
		return nil
	}

	for _, c := range ctx.AllFull_column_name() {
		r, err := q.tsqlIsFullColumnNameSensitive(c)
		if err != nil {
			return err
		}
		q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, r.SourceColumns)
	}

	for _, e := range ctx.AllExpression() {
		if err := q.extractPredicateColumnFromExpression(e); err != nil {
			return err
		}
	}

	return nil
}

func (q *querySpanExtractor) extractPredicateColumnFromExpression(
	ctx antlr.ParserRuleContext,
) error {
	if ctx == nil {
		return nil
	}

	switch ctx := ctx.(type) {
	case parser.IFull_column_nameContext:
		r, err := q.tsqlIsFullColumnNameSensitive(ctx)
		if err != nil {
			return err
		}
		q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, r.SourceColumns)
	case parser.ISubqueryContext:
		if err := q.extractPredicateColumnFromSubquery(ctx); err != nil {
			return err
		}
	default:
	}

	var list []antlr.ParserRuleContext
	for _, child := range ctx.GetChildren() {
		if child, ok := child.(antlr.ParserRuleContext); ok {
			list = append(list, child)
		}
	}

	for _, c := range list {
		if err := q.extractPredicateColumnFromExpression(c); err != nil {
			return err
		}
	}

	return nil
}

func (q *querySpanExtractor) extractPredicateColumnFromSubquery(
	ctx parser.ISubqueryContext,
) error {
	if ctx == nil {
		return nil
	}

	// Clone the extractor so the subquery sees the enclosing FROM clause as
	// outer table sources. Without this, correlated subqueries like
	// `WHERE EXISTS (SELECT 1 FROM t e WHERE e.col = outer.col)` cannot
	// resolve the `outer` alias and fail with "no matching column".
	cloneExtractor := &querySpanExtractor{
		ctx:                 q.ctx,
		defaultDatabase:     q.defaultDatabase,
		defaultSchema:       q.defaultSchema,
		ignoreCaseSensitive: q.ignoreCaseSensitive,
		gCtx:                q.gCtx,
		ctes:                q.ctes,
		outerTableSources:   append(q.outerTableSources, q.tableSourcesFrom...),
		predicateColumns:    make(base.SourceColumnSet),
	}
	tableSource, err := cloneExtractor.extractTSqlSensitiveFieldsFromSubquery(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get query span for subquery")
	}

	q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, cloneExtractor.predicateColumns)
	for _, r := range tableSource.GetQuerySpanResult() {
		q.predicateColumns, _ = base.MergeSourceColumnSet(q.predicateColumns, r.SourceColumns)
	}

	return nil
}
