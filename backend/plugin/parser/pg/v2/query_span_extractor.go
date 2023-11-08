package v2

import (
	"context"
	"fmt"

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
	cteOuterSchemaInfo []base.PseudoTable
	fromFieldList      []base.TableSource
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

	return nil, nil
}

// extractTableSourceFromNode is the entry for recursively extracting the span sources from the given node.
// It returns the table source for the given node, which can be a physical table or a down cast temporary table losing the original schema info.
func (q *querySpanExtractor) extractTableSourceFromNode(ctx context.Context, node *pgquery.Node) (base.TableSource, error) {
	if node == nil {
		return nil, nil
	}

	switch node := node.Node.(type) {
	case *pgquery.Node_SelectStmt:
		return q.extractTableSourceFromSelect(ctx, node)
	}
	return nil, nil
}

func (q *querySpanExtractor) extractTableSourceFromSelect(ctx context.Context, node *pgquery.Node_SelectStmt) (base.TableSource, error) {
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
			var cteTableResource base.PseudoTable
			var err error
			if node.SelectStmt.WithClause.Recursive {
				cteTableResource, err = q.extractTemporaryTableResourceFromRecursiveCTE(ctx, cteExpr)
			} else {
				cteTableResource, err = q.extractTemporaryTableResourceFromNonRecursiveCTE(ctx, cteExpr)
			}
			if err != nil {
				return nil, err
			}
			q.cteOuterSchemaInfo = append(q.cteOuterSchemaInfo, cteTableResource)
		}
	}

	// The VALUES case.
	// https://www.postgresql.org/docs/current/queries-values.html
	if len(node.SelectStmt.ValuesLists) > 0 {
		var columnSourceSets []base.SourceColumnSet
		for _, row := range node.SelectStmt.ValuesLists {
			list, ok := row.Node.(*pgquery.Node_List)
			if !ok {
				return nil, errors.Errorf("expect List for VALUES list, but got %T", row.Node)
			}
			for i, value := range list.List.Items {
				sourceColumnSet, err := q.extractSourceColumnSetFromExpressionNode(ctx, value)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract source column set from VALUES expression: %+v", value)
				}
				if i >= len(columnSourceSets) {
					columnSourceSets = append(columnSourceSets, sourceColumnSet)
				} else {
					columnSourceSets[i], _ = base.MergeSourceColumnSet(columnSourceSets[i], sourceColumnSet)
				}
			}
		}

		var querySpanResults []*base.QuerySpanResult
		for i, columnSourceSet := range columnSourceSets {
			querySpanResults = append(querySpanResults, &base.QuerySpanResult{
				Name:          fmt.Sprintf("column%d", i+1),
				SourceColumns: columnSourceSet,
			})
		}
		// FIXME(zp): Consider the alias case to give a name to table.
		// => SELECT * FROM (VALUES (1, 'one'), (2, 'two'), (3, 'three')) AS t (num,letter);
		return base.PseudoTable{
			Name:    "",
			Columns: querySpanResults,
		}, nil
	}

	// UNION/INTERSECT/EXCEPT case.
	switch node.SelectStmt.Op {
	case pgquery.SetOperation_SETOP_UNION, pgquery.SetOperation_SETOP_INTERSECT, pgquery.SetOperation_SETOP_EXCEPT:
		leftSpanResults, err := q.extractTableSourceFromSelect(ctx, &pgquery.Node_SelectStmt{SelectStmt: node.SelectStmt.Larg})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract span result from left select: %+v", node.SelectStmt.Larg)
		}
		rightSpanResults, err := q.extractTableSourceFromSelect(ctx, &pgquery.Node_SelectStmt{SelectStmt: node.SelectStmt.Rarg})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract span result from right select: %+v", node.SelectStmt.Rarg)
		}
		leftQuerySpanResult, rightQuerySpanResult := leftSpanResults.GetQuerySpanResult(), rightSpanResults.GetQuerySpanResult()
		if len(leftQuerySpanResult) != len(leftQuerySpanResult) {
			return nil, errors.Wrapf(err, "left select has %d columns, but right select has %d columns", len(leftQuerySpanResult), len(leftQuerySpanResult))
		}
		var result []*base.QuerySpanResult
		for i, leftSpanResult := range leftQuerySpanResult {
			rightSpanResult := rightQuerySpanResult[i]
			newResourceColumns, _ := base.MergeSourceColumnSet(leftSpanResult.SourceColumns, rightSpanResult.SourceColumns)
			result = append(result, &base.QuerySpanResult{
				Name:          leftSpanResult.Name,
				SourceColumns: newResourceColumns,
			})
		}
		// FIXME(zp): Consider UNION alias.
		return base.PseudoTable{
			Name:    "",
			Columns: result,
		}, nil
	case pgquery.SetOperation_SETOP_NONE:
	default:
		return nil, errors.Errorf("unsupported set operation: %s", node.SelectStmt.Op)
	}

	// The FROM clause.

	return nil, nil
}

func (q *querySpanExtractor) extractTemporaryTableResourceFromNonRecursiveCTE(ctx context.Context, cteExpr *pgquery.Node_CommonTableExpr) (base.PseudoTable, error) {
	tableSource, err := q.extractTableSourceFromNode(ctx, cteExpr.CommonTableExpr.Ctequery)
	if err != nil {
		return base.PseudoTable{}, errors.Wrapf(err, "failed to extract span result from CTE query: %+v", cteExpr.CommonTableExpr.Ctequery)
	}

	querySpanResults := tableSource.GetQuerySpanResult()
	if len(cteExpr.CommonTableExpr.Aliascolnames) > 0 {
		if len(cteExpr.CommonTableExpr.Aliascolnames) != len(querySpanResults) {
			return base.PseudoTable{}, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(querySpanResults), len(cteExpr.CommonTableExpr.Aliascolnames))
		}
		for i, name := range cteExpr.CommonTableExpr.Aliascolnames {
			stringNode, ok := name.Node.(*pgquery.Node_String_)
			if !ok {
				return base.PseudoTable{}, errors.Errorf("expect string node for alias column name, but got %T", name.Node)
			}
			querySpanResults[i].Name = stringNode.String_.Sval
		}
	}

	return base.PseudoTable{
		Name:    cteExpr.CommonTableExpr.Ctename,
		Columns: querySpanResults,
	}, nil
}

func (q *querySpanExtractor) extractTemporaryTableResourceFromRecursiveCTE(ctx context.Context, cteExpr *pgquery.Node_CommonTableExpr) (base.PseudoTable, error) {
	switch selectNode := cteExpr.CommonTableExpr.Ctequery.Node.(type) {
	case *pgquery.Node_SelectStmt:
		if selectNode.SelectStmt.Op != pgquery.SetOperation_SETOP_UNION {
			return q.extractTemporaryTableResourceFromNonRecursiveCTE(ctx, cteExpr)
		}
		// For PostgreSQL, recursive CTE would be a UNION statement, and the left node is the initial part,
		// the right node is the recursive part.
		// https://www.postgresql.org/docs/15/queries-with.html#QUERIES-WITH-RECURSIVE
		initialTableSource, err := q.extractTableSourceFromSelect(ctx, &pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Larg})
		if err != nil {
			return base.PseudoTable{}, errors.Wrapf(err, "failed to extract span result from CTE initial query: %+v", selectNode.SelectStmt.Larg)
		}
		initialQuerySpanResult := initialTableSource.GetQuerySpanResult()
		if len(cteExpr.CommonTableExpr.Aliascolnames) > 0 {
			if len(cteExpr.CommonTableExpr.Aliascolnames) != len(initialQuerySpanResult) {
				return base.PseudoTable{}, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(initialQuerySpanResult), len(cteExpr.CommonTableExpr.Aliascolnames))
			}
			for i, name := range cteExpr.CommonTableExpr.Aliascolnames {
				stringNode, ok := name.Node.(*pgquery.Node_String_)
				if !ok {
					return base.PseudoTable{}, errors.Errorf("expect string node for alias column name, but got %T", name.Node)
				}
				initialQuerySpanResult[i].Name = stringNode.String_.Sval
			}
		}

		cteTableResource := base.PseudoTable{Name: cteExpr.CommonTableExpr.Ctename, Columns: initialQuerySpanResult}

		// Compute dependent closures.
		// There are two ways to compute dependent closures:
		//   1. find the all dependent edges, then use graph theory traversal to find the closure.
		//   2. Iterate to simulate the CTE recursive process, each turn check whether the columns have changed, and stop if not change.
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
			recursiveTableSource, err := q.extractTableSourceFromSelect(ctx, &pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Rarg})
			if err != nil {
				return base.PseudoTable{}, errors.Wrapf(err, "failed to extract span result from CTE recursive query: %+v", selectNode.SelectStmt.Rarg)
			}
			recursiveQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
			if len(recursiveQuerySpanResult) != len(initialQuerySpanResult) {
				return base.PseudoTable{}, errors.Errorf("cte table expr has %d columns, but recursive query has %d columns", len(initialQuerySpanResult), len(recursiveQuerySpanResult))
			}

			changed := false
			for i, spanQueryResult := range recursiveQuerySpanResult {
				newResourceColumns, hasDiff := base.MergeSourceColumnSet(initialQuerySpanResult[i].SourceColumns, spanQueryResult.SourceColumns)
				if hasDiff {
					changed = true
					initialQuerySpanResult[i].SourceColumns = newResourceColumns
				}
			}

			if !changed {
				break
			}
			q.cteOuterSchemaInfo[len(q.cteOuterSchemaInfo)-1].Columns = initialQuerySpanResult
		}
		return cteTableResource, nil
	default:
		return q.extractTemporaryTableResourceFromNonRecursiveCTE(ctx, cteExpr)
	}
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpressionNode(ctx context.Context, node *pgquery.Node) (base.SourceColumnSet, error) {
	// TODO(zp): implement me.
	return nil, nil
}

// extractSourceColumnSetFromExpressionNodeList is the helper function to extract the source column set from the given expression node list,
// which iterates the list and merge each set.
func (q *querySpanExtractor) extractSourceColumnSetFromExpressionNodeList(ctx context.Context, list []*pgquery.Node) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	for _, node := range list {
		sourceColumnSet, err := q.extractSourceColumnSetFromExpressionNode(ctx, node)
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, sourceColumnSet)
	}
	return result, nil
}
