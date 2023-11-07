package v2

import (
	"context"

	"github.com/pkg/errors"

	pgquery "github.com/pganalyze/pg_query_go/v4"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// querySpanExtractor is the extractor to extract the query span from the given pgquery.RawStmt.
type querySpanExtractor struct {
	ctx         context.Context
	connectedDB string
	// The metaCache serves as a lazy-load cache for the database metadata and should not be accessed directly.
	// Instead, use querySpanExtractor.getDatabaseMetadata to access it.
	metaCache map[string]*model.DatabaseMetadata
	f         base.GetDatabaseMetadataFunc

	// Private fields.
	// cteOuterSchemaInfo is the schema info for the outer query.
	// It should be reset to the previous state after the query is processed.
	cteOuterSchemaInfo []base.TableResource
}

// newQuerySpanExtractor creates a new query span extractor, the databaseMetadata and the ast are in the read guard.
func newQuerySpanExtractor(connectedDB string, getDatabaseMetadata base.GetDatabaseMetadataFunc) *querySpanExtractor {
	return &querySpanExtractor{
		connectedDB: connectedDB,
		metaCache:   make(map[string]*model.DatabaseMetadata),
		f:           getDatabaseMetadata,
	}
}

// nolint: unused
func (q *querySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if meta, ok := q.metaCache[database]; ok {
		return meta, nil
	}
	meta, err := q.f(q.ctx, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", database)
	}
	q.metaCache[database] = meta
	return meta, nil
}

func (q *querySpanExtractor) getQuerySpanResult(ctx context.Context, ast *pgquery.RawStmt) ([]*base.QuerySpanResult, error) {
	q.ctx = ctx

	return q.extractSpanResultFromNode(ctx, ast.Stmt)
}

// extractSpanResultFromNode is the entry for recursively extracting the span sources from the given node.
func (q *querySpanExtractor) extractSpanResultFromNode(ctx context.Context, node *pgquery.Node) ([]*base.QuerySpanResult, error) {
	if node == nil {
		return nil, nil
	}

	switch node := node.Node.(type) {
	case *pgquery.Node_SelectStmt:
		return q.extractSpanResultFromSelect(ctx, node)
	}
	return nil, nil
}

func (q *querySpanExtractor) extractSpanResultFromSelect(ctx context.Context, node *pgquery.Node_SelectStmt) ([]*base.QuerySpanResult, error) {
	// The WITH clause.
	if node.SelectStmt.WithClause != nil {
		previousCteOuterLength := len(q.cteOuterSchemaInfo)
		defer func() {
			q.cteOuterSchemaInfo = q.cteOuterSchemaInfo[:previousCteOuterLength]
		}()

		for _, cte := range node.SelectStmt.WithClause.Ctes {
			cteExpr, ok := cte.Node.(*pgquery.Node_CommonTableExpr)
			if !ok {
				return nil, errors.Errorf("expect CommonTableExpr for CTE, but got %T", cte.Node)
			}
			var cteTableResource base.TableResource
			var err error
			if node.SelectStmt.WithClause.Recursive {
				cteTableResource, err = q.extractTableResourceFromRecursiveCTE(ctx, cteExpr)
			} else {
				cteTableResource, err = q.extractTableResourceFromNonRecursiveCTE(ctx, cteExpr)
			}
			if err != nil {
				return nil, err
			}
			q.cteOuterSchemaInfo = append(q.cteOuterSchemaInfo, cteTableResource)
		}
	}

	// The VALUES case.
	if len(node.SelectStmt.ValuesLists) > 0 {
		var result []*base.QuerySpanResult
		for _, row := range node.SelectStmt.ValuesLists {
			list, ok := row.Node.(*pgquery.Node_List)
			if !ok {
				return nil, errors.Errorf("expect List for VALUES list, but got %T", row.Node)
			}
			for _, value := range list.List.Items {
				spanResult, err := q.extractColumnRefFromExpressionNode(item)
				if err != nil {
					return nil, err
				}
				result = append(result, spanResult)
			}
		}
	}
	return nil, nil
}

func (q *querySpanExtractor) extractTableResourceFromNonRecursiveCTE(ctx context.Context, cteExpr *pgquery.Node_CommonTableExpr) (base.TableResource, error) {
	querySpanResults, err := q.extractSpanResultFromNode(ctx, cteExpr.CommonTableExpr.Ctequery)
	if err != nil {
		return base.TableResource{}, errors.Wrapf(err, "failed to extract span result from CTE query: %+v", cteExpr.CommonTableExpr.Ctequery)
	}

	if len(cteExpr.CommonTableExpr.Aliascolnames) > 0 {
		if len(cteExpr.CommonTableExpr.Aliascolnames) != len(querySpanResults) {
			return base.TableResource{}, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(querySpanResults), len(cteExpr.CommonTableExpr.Aliascolnames))
		}
		for i, name := range cteExpr.CommonTableExpr.Aliascolnames {
			stringNode, ok := name.Node.(*pgquery.Node_String_)
			if !ok {
				return base.TableResource{}, errors.Errorf("expect string node for alias column name, but got %T", name.Node)
			}
			querySpanResults[i].Name = stringNode.String_.Sval
		}
	}

	return base.TableResource{
		Name:    cteExpr.CommonTableExpr.Ctename,
		Columns: querySpanResults,
	}, nil
}

func (q *querySpanExtractor) extractTableResourceFromRecursiveCTE(ctx context.Context, cteExpr *pgquery.Node_CommonTableExpr) (base.TableResource, error) {
	switch selectNode := cteExpr.CommonTableExpr.Ctequery.Node.(type) {
	case *pgquery.Node_SelectStmt:
		if selectNode.SelectStmt.Op != pgquery.SetOperation_SETOP_UNION {
			return q.extractTableResourceFromNonRecursiveCTE(ctx, cteExpr)
		}
		// For PostgreSQL, recursive CTE would be an UNION statement, and the left node is the initial part,
		// the right node is the recursive part.
		// https://www.postgresql.org/docs/15/queries-with.html#QUERIES-WITH-RECURSIVE
		initialTableResource, err := q.extractSpanResultFromSelect(ctx, &pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Larg})
		if err != nil {
			return base.TableResource{}, errors.Wrapf(err, "failed to extract span result from CTE initial query: %+v", selectNode.SelectStmt.Larg)
		}
		if len(cteExpr.CommonTableExpr.Aliascolnames) > 0 {
			if len(cteExpr.CommonTableExpr.Aliascolnames) != len(initialTableResource) {
				return base.TableResource{}, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(initialTableResource), len(cteExpr.CommonTableExpr.Aliascolnames))
			}
			for i, name := range cteExpr.CommonTableExpr.Aliascolnames {
				stringNode, ok := name.Node.(*pgquery.Node_String_)
				if !ok {
					return base.TableResource{}, errors.Errorf("expect string node for alias column name, but got %T", name.Node)
				}
				initialTableResource[i].Name = stringNode.String_.Sval
			}
		}

		cteTableResource := base.TableResource{Name: cteExpr.CommonTableExpr.Ctename, Columns: initialTableResource}

		// Compute dependent closures.
		// There are two ways to compute dependent closures:
		//   1. find the all dependent edges, then use graph theory traversal to find the closure.
		//   2. Iterate to simulate the CTE recursive process, each turn check whether the columns has changed, and stop if not change.
		//
		// Consider the option 2 can easy to implementation, because the simulate process has been written.
		// On the other hand, the number of iterations of the entire algorithm will not exceed the length of fields.
		// In actual use, the length of fields will not be more than 20 generally.
		// So I think it's OK for now.
		// If any performance issues in use, optimize here.
		q.cteOuterSchemaInfo = append(q.cteOuterSchemaInfo, cteTableResource)
		defer func() {
			q.cteOuterSchemaInfo = q.cteOuterSchemaInfo[:len(q.cteOuterSchemaInfo)-1]
		}()

		for {
			spanQueryResults, err := q.extractSpanResultFromSelect(ctx, &pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Rarg})
			if err != nil {
				return base.TableResource{}, errors.Wrapf(err, "failed to extract span result from CTE recursive query: %+v", selectNode.SelectStmt.Rarg)
			}
			if len(spanQueryResults) != len(initialTableResource) {
				return base.TableResource{}, errors.Errorf("cte table expr has %d columns, but recursive query has %d columns", len(initialTableResource), len(spanQueryResults))
			}

			changed := false
			for i, spanQueryResult := range spanQueryResults {
				newResourceColumns, hasDiff := base.MergeSourceColumnSet(initialTableResource[i].SourceColumns, spanQueryResult.SourceColumns)
				if hasDiff {
					changed = true
					initialTableResource[i].SourceColumns = newResourceColumns
				}
			}

			if !changed {
				break
			}
			q.cteOuterSchemaInfo[len(q.cteOuterSchemaInfo)-1].Columns = initialTableResource
		}
		return cteTableResource, nil
	default:
		return q.extractTableResourceFromNonRecursiveCTE(ctx, cteExpr)
	}
}
