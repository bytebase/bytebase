package mysql

import (
	"context"
	"strconv"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// omniQuerySpanExtractor extracts query span information from omni MySQL AST.
// It embeds querySpanExtractor so shared metadata and identifier-resolution
// helpers can be reused as the omni implementation grows.
type omniQuerySpanExtractor struct {
	*querySpanExtractor
	source string
}

func newOmniQuerySpanExtractor(defaultDatabase string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *omniQuerySpanExtractor {
	return &omniQuerySpanExtractor{
		querySpanExtractor: newQuerySpanExtractor(defaultDatabase, gCtx, ignoreCaseSensitive),
	}
}

// getOmniQuerySpan is the package-internal omni entry point used by migration
// tests and the production MySQL query-span path after cutover.
func (q *omniQuerySpanExtractor) getOmniQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx
	q.source = statement

	stmts, err := ParseMySQLOmni(statement)
	if err != nil {
		return nil, err
	}
	if stmts == nil || len(stmts.Items) == 0 {
		return &base.QuerySpan{
			Type:             base.Select,
			SourceColumns:    base.SourceColumnSet{},
			Results:          []base.QuerySpanResult{},
			PredicateColumns: base.SourceColumnSet{},
		}, nil
	}
	if len(stmts.Items) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(stmts.Items))
	}

	root := stmts.Items[0]
	accessTables := collectOmniAccessTables(root, q.defaultDatabase, q.gCtx.Engine == storepb.Engine_STARROCKS)
	allSystems, mixed := isMixedQuery(accessTables, q.ignoreCaseSensitive)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryType := classifyOmniQueryType(root, allSystems, q.source)
	if allSystems {
		accessTables = base.SourceColumnSet{}
	}
	resultSourceColumns := accessTables
	if queryType != base.Select {
		resultSourceColumns = base.SourceColumnSet{}
	}

	var results []base.QuerySpanResult
	if queryType == base.Select {
		tableSource, err := q.extractOmniSelectRoot(root)
		if err != nil {
			var resourceNotFound *base.ResourceNotFoundError
			if errors.As(err, &resourceNotFound) {
				return &base.QuerySpan{
					Type:             base.Select,
					SourceColumns:    accessTables,
					Results:          []base.QuerySpanResult{},
					PredicateColumns: base.SourceColumnSet{},
					NotFoundError:    resourceNotFound,
				}, nil
			}
			return nil, err
		}
		results = tableSource.GetQuerySpanResult()
	}
	return &base.QuerySpan{
		Type:             queryType,
		SourceColumns:    resultSourceColumns,
		Results:          results,
		PredicateColumns: base.SourceColumnSet{},
	}, nil
}

func (q *omniQuerySpanExtractor) extractOmniSelectRoot(root ast.Node) (*base.PseudoTable, error) {
	switch n := root.(type) {
	case *ast.SelectStmt:
		return q.extractOmniSelectStmt(n)
	case *ast.ExplainStmt:
		return &base.PseudoTable{}, nil
	case *ast.TableStmt:
		return q.extractOmniTableStmt(n)
	case *ast.ValuesStmt:
		return q.extractOmniValuesStmt(n)
	default:
		return &base.PseudoTable{}, nil
	}
}

func (q *omniQuerySpanExtractor) extractOmniSelectStmt(stmt *ast.SelectStmt) (*base.PseudoTable, error) {
	if stmt == nil {
		return &base.PseudoTable{}, nil
	}
	if stmt.Into != nil {
		return nil, errors.New("meet unsupported select statement with into")
	}

	originalTableSourceFrom := q.tableSourceFrom
	originalPriorLength := len(q.priorTableInFrom)
	originalCTELength := len(q.ctes)
	defer func() {
		q.tableSourceFrom = originalTableSourceFrom
		q.priorTableInFrom = q.priorTableInFrom[:originalPriorLength]
		q.ctes = q.ctes[:originalCTELength]
	}()

	if err := q.processOmniCTEs(stmt.CTEs); err != nil {
		return nil, err
	}
	if stmt.SetOp != ast.SetOpNone {
		return q.extractOmniSetOp(stmt)
	}

	fromSources, err := q.extractOmniTableSources(stmt.From)
	if err != nil {
		return nil, err
	}
	q.tableSourceFrom = append(q.tableSourceFrom, fromSources...)

	results, err := q.extractOmniTargetList(stmt.TargetList, fromSources)
	if err != nil {
		return nil, err
	}
	return &base.PseudoTable{
		Columns: results,
	}, nil
}

func (q *omniQuerySpanExtractor) extractOmniTableStmt(stmt *ast.TableStmt) (*base.PseudoTable, error) {
	if stmt == nil || stmt.Table == nil {
		return &base.PseudoTable{}, nil
	}
	tableSource, err := q.findTableSchema(stmt.Table.Schema, stmt.Table.Name)
	if err != nil {
		return nil, err
	}
	return &base.PseudoTable{Columns: tableSource.GetQuerySpanResult()}, nil
}

func (q *omniQuerySpanExtractor) extractOmniValuesStmt(stmt *ast.ValuesStmt) (*base.PseudoTable, error) {
	if stmt == nil || len(stmt.Rows) == 0 {
		return &base.PseudoTable{}, nil
	}
	var columns []base.QuerySpanResult
	for _, expr := range stmt.Rows[0] {
		field, err := q.extractOmniExpr(expr)
		if err != nil {
			return nil, err
		}
		columns = append(columns, field)
	}
	return &base.PseudoTable{Columns: columns}, nil
}

func (q *omniQuerySpanExtractor) extractOmniSetOp(stmt *ast.SelectStmt) (*base.PseudoTable, error) {
	if stmt == nil || stmt.SetOp == ast.SetOpNone {
		return q.extractOmniSelectStmt(stmt)
	}
	left, err := q.extractOmniSelectStmt(stmt.Left)
	if err != nil {
		return nil, err
	}
	right, err := q.extractOmniSelectStmt(stmt.Right)
	if err != nil {
		return nil, err
	}
	leftResults := left.GetQuerySpanResult()
	rightResults := right.GetQuerySpanResult()
	if len(leftResults) != len(rightResults) {
		return nil, errors.Errorf("MySQL UNION operator left has %d fields, right has %d fields", len(leftResults), len(rightResults))
	}
	results := make([]base.QuerySpanResult, 0, len(leftResults))
	for i := range leftResults {
		sourceColumns, _ := base.MergeSourceColumnSet(leftResults[i].SourceColumns, rightResults[i].SourceColumns)
		results = append(results, base.QuerySpanResult{
			Name:          leftResults[i].Name,
			SourceColumns: sourceColumns,
			IsPlainField:  false,
		})
	}
	return &base.PseudoTable{Columns: results}, nil
}

func (q *omniQuerySpanExtractor) processOmniCTEs(ctes []*ast.CommonTableExpr) error {
	for _, cte := range ctes {
		tableSource, err := q.extractOmniCTE(cte)
		if err != nil {
			return err
		}
		if tableSource != nil {
			q.ctes = append(q.ctes, tableSource)
		}
	}
	return nil
}

func (q *omniQuerySpanExtractor) extractOmniCTE(cte *ast.CommonTableExpr) (*base.PseudoTable, error) {
	if cte == nil {
		return nil, nil
	}
	if cte.Recursive {
		return q.extractOmniRecursiveCTE(cte)
	}
	return q.extractOmniNonRecursiveCTE(cte)
}

func (q *omniQuerySpanExtractor) extractOmniNonRecursiveCTE(cte *ast.CommonTableExpr) (*base.PseudoTable, error) {
	tableSource, err := q.extractOmniSelectStmt(cte.Select)
	if err != nil {
		return nil, err
	}
	results := cloneOmniQuerySpanResults(tableSource.GetQuerySpanResult())
	if len(cte.Columns) > 0 {
		if len(cte.Columns) != len(results) {
			return nil, errors.Errorf("MySQL CTE column list should have the same length, but got %d and %d", len(cte.Columns), len(results))
		}
		for i := range cte.Columns {
			results[i].Name = cte.Columns[i]
		}
	}
	return &base.PseudoTable{Name: cte.Name, Columns: results}, nil
}

func (q *omniQuerySpanExtractor) extractOmniRecursiveCTE(cte *ast.CommonTableExpr) (*base.PseudoTable, error) {
	if cte.Select == nil || cte.Select.SetOp == ast.SetOpNone {
		return q.extractOmniNonRecursiveCTE(cte)
	}
	initialTable, err := q.extractOmniRecursiveCTEAnchor(cte.Select)
	if err != nil {
		return nil, err
	}
	results := cloneOmniQuerySpanResults(initialTable.GetQuerySpanResult())
	if len(cte.Columns) > 0 {
		if len(cte.Columns) != len(results) {
			return nil, errors.Errorf("The common table expression and column names list have different column counts")
		}
		for i := range cte.Columns {
			results[i].Name = cte.Columns[i]
		}
	}
	cteInfo := &base.PseudoTable{Name: cte.Name, Columns: results}
	q.ctes = append(q.ctes, cteInfo)
	defer func() {
		q.ctes = q.ctes[:len(q.ctes)-1]
	}()

	for {
		recursiveTable, err := q.extractOmniSelectStmt(cte.Select)
		if err != nil {
			return nil, err
		}
		recursiveResults := recursiveTable.GetQuerySpanResult()
		if len(recursiveResults) != len(cteInfo.Columns) {
			return nil, errors.Errorf("The common table expression and column names list have different column counts")
		}
		changed := false
		for i := range recursiveResults {
			var ok bool
			cteInfo.Columns[i].SourceColumns, ok = base.MergeSourceColumnSet(cteInfo.Columns[i].SourceColumns, recursiveResults[i].SourceColumns)
			cteInfo.Columns[i].IsPlainField = false
			changed = changed || ok
		}
		if !changed {
			break
		}
	}
	return cteInfo, nil
}

func (q *omniQuerySpanExtractor) extractOmniRecursiveCTEAnchor(stmt *ast.SelectStmt) (*base.PseudoTable, error) {
	if stmt == nil || stmt.SetOp == ast.SetOpNone {
		return q.extractOmniSelectStmt(stmt)
	}
	if stmt.Left != nil {
		return q.extractOmniRecursiveCTEAnchor(stmt.Left)
	}
	return q.extractOmniRecursiveCTEAnchor(stmt.Right)
}

func cloneOmniQuerySpanResults(results []base.QuerySpanResult) []base.QuerySpanResult {
	clone := make([]base.QuerySpanResult, len(results))
	copy(clone, results)
	return clone
}

func (q *omniQuerySpanExtractor) extractOmniTableSources(tableExprs []ast.TableExpr) ([]base.TableSource, error) {
	var result []base.TableSource
	for _, tableExpr := range tableExprs {
		tableSource, err := q.extractOmniTableSource(tableExpr)
		if err != nil {
			return nil, err
		}
		if tableSource == nil {
			continue
		}
		q.priorTableInFrom = append(q.priorTableInFrom, tableSource)
		result = append(result, tableSource)
	}
	return result, nil
}

func (q *omniQuerySpanExtractor) extractOmniTableSource(tableExpr ast.TableExpr) (base.TableSource, error) {
	switch t := tableExpr.(type) {
	case nil:
		return nil, nil
	case *ast.TableRef:
		if isOmniDualTable(t) {
			return &base.PseudoTable{Name: t.Name}, nil
		}
		tableSource, err := q.findTableSchema(t.Schema, t.Name)
		if err != nil {
			return nil, err
		}
		if t.Alias == "" {
			return tableSource, nil
		}
		return &base.PseudoTable{
			Name:    t.Alias,
			Columns: tableSource.GetQuerySpanResult(),
		}, nil
	case *ast.SubqueryExpr:
		return q.extractOmniSubquery(t)
	case *ast.JoinClause:
		return q.extractOmniJoin(t)
	case *ast.JsonTableExpr:
		return q.extractOmniJSONTable(t)
	default:
		return nil, errors.Errorf("unsupported omni MySQL table source %T", tableExpr)
	}
}

func (q *omniQuerySpanExtractor) extractOmniJSONTable(expr *ast.JsonTableExpr) (base.TableSource, error) {
	if expr == nil {
		return nil, nil
	}
	name := expr.Alias
	if name == "" {
		name = q.omniExprName(expr.Expr)
	}
	field, err := q.extractOmniExpr(expr.Expr)
	if err != nil {
		return nil, err
	}
	var columns []base.QuerySpanResult
	for _, column := range flattenOmniJSONTableColumns(expr.Columns) {
		columns = append(columns, base.QuerySpanResult{
			Name:          column.Name,
			SourceColumns: field.SourceColumns,
			IsPlainField:  field.IsPlainField,
		})
	}
	return &base.PseudoTable{Name: name, Columns: columns}, nil
}

func flattenOmniJSONTableColumns(columns []*ast.JsonTableColumn) []*ast.JsonTableColumn {
	var result []*ast.JsonTableColumn
	for _, column := range columns {
		if column == nil {
			continue
		}
		if column.Nested {
			result = append(result, flattenOmniJSONTableColumns(column.NestedCols)...)
			continue
		}
		result = append(result, column)
	}
	return result
}

func (q *omniQuerySpanExtractor) extractOmniJoin(join *ast.JoinClause) (base.TableSource, error) {
	if join == nil {
		return nil, nil
	}
	left, err := q.extractOmniTableSource(join.Left)
	if err != nil {
		return nil, err
	}
	if left != nil {
		q.tableSourceFrom = append(q.tableSourceFrom, left)
	}
	right, err := q.extractOmniTableSource(join.Right)
	if err != nil {
		return nil, err
	}
	if right != nil {
		q.tableSourceFrom = append(q.tableSourceFrom, right)
	}

	var using []string
	if condition, ok := join.Condition.(*ast.UsingCondition); ok && condition != nil {
		using = condition.Columns
	}
	return joinTableSources(left, right, omniJoinType(join.Type), using), nil
}

func omniJoinType(joinType ast.JoinType) joinType {
	switch joinType {
	case ast.JoinInner:
		return InnerJoin
	case ast.JoinLeft:
		return LeftOuterJoin
	case ast.JoinRight:
		return RightOuterJoin
	case ast.JoinCross:
		return CrossJoin
	case ast.JoinNatural:
		return NaturalInnerJoin
	case ast.JoinStraight:
		return StraightJoin
	case ast.JoinNaturalLeft:
		return NaturalLeftOuterJoin
	case ast.JoinNaturalRight:
		return NaturalRightOuterJoin
	default:
		return Join
	}
}

func (q *omniQuerySpanExtractor) extractOmniSubquery(expr *ast.SubqueryExpr) (*base.PseudoTable, error) {
	if expr == nil {
		return &base.PseudoTable{}, nil
	}
	subqueryExtractor := q.cloneOmniForSubquery()
	if expr.Lateral {
		subqueryExtractor.outerTableSources = append(subqueryExtractor.outerTableSources, q.priorTableInFrom...)
	}
	tableSource, err := subqueryExtractor.extractOmniSelectStmt(expr.Select)
	if err != nil {
		return nil, err
	}
	results := cloneOmniQuerySpanResults(tableSource.GetQuerySpanResult())
	if len(expr.Columns) > 0 {
		if len(expr.Columns) != len(results) {
			return nil, errors.Errorf("derived table column list length %d doesn't match result column length %d", len(expr.Columns), len(results))
		}
		for i := range expr.Columns {
			results[i].Name = expr.Columns[i]
		}
	}
	if expr.Alias == "" {
		return &base.PseudoTable{Name: tableSource.Name, Columns: results}, nil
	}
	return &base.PseudoTable{
		Name:    expr.Alias,
		Columns: results,
	}, nil
}

func (q *omniQuerySpanExtractor) cloneOmniForSubquery() *omniQuerySpanExtractor {
	return &omniQuerySpanExtractor{
		querySpanExtractor: &querySpanExtractor{
			ctx:                 q.ctx,
			defaultDatabase:     q.defaultDatabase,
			gCtx:                q.gCtx,
			ctes:                q.ctes,
			outerTableSources:   append(q.outerTableSources, q.tableSourceFrom...),
			tableSourceFrom:     []base.TableSource{},
			ignoreCaseSensitive: q.ignoreCaseSensitive,
			viewResolutionStack: cloneViewResolutionStack(q.viewResolutionStack),
		},
		source: q.source,
	}
}

func (q *omniQuerySpanExtractor) extractOmniTargetList(targets []ast.ExprNode, fromSources []base.TableSource) ([]base.QuerySpanResult, error) {
	var result []base.QuerySpanResult
	for _, target := range targets {
		switch t := target.(type) {
		case *ast.ResTarget:
			if _, ok := t.Val.(*ast.StarExpr); ok {
				fields, err := q.extractOmniTableWild("", "", fromSources)
				if err != nil {
					return nil, err
				}
				result = append(result, fields...)
				continue
			}
			if columnRef, ok := t.Val.(*ast.ColumnRef); ok && columnRef.Star {
				fields, err := q.extractOmniColumnWild(columnRef, fromSources)
				if err != nil {
					return nil, err
				}
				result = append(result, fields...)
				continue
			}
		case *ast.StarExpr:
			fields, err := q.extractOmniTableWild("", "", fromSources)
			if err != nil {
				return nil, err
			}
			result = append(result, fields...)
			continue
		case *ast.ColumnRef:
			if t.Star {
				fields, err := q.extractOmniColumnWild(t, fromSources)
				if err != nil {
					return nil, err
				}
				result = append(result, fields...)
				continue
			}
		default:
		}

		field, err := q.extractOmniTarget(target)
		if err != nil {
			return nil, err
		}
		result = append(result, field)
	}
	return result, nil
}

func (q *omniQuerySpanExtractor) extractOmniTarget(target ast.ExprNode) (base.QuerySpanResult, error) {
	switch t := target.(type) {
	case *ast.ResTarget:
		field, err := q.extractOmniExpr(t.Val)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		if t.Name != "" {
			field.Name = t.Name
		}
		return field, nil
	default:
		return q.extractOmniExpr(target)
	}
}

func (q *omniQuerySpanExtractor) extractOmniExpr(expr ast.ExprNode) (base.QuerySpanResult, error) {
	switch e := expr.(type) {
	case nil:
		return base.QuerySpanResult{
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  true,
		}, nil
	case *ast.ColumnRef:
		if e.Star {
			return base.QuerySpanResult{}, errors.Errorf("asterisk is not a scalar expression")
		}
		sourceColumns, err := q.getFieldColumnSource(e.Schema, e.Table, e.Column)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{
			Name:          e.Column,
			SourceColumns: sourceColumns,
			IsPlainField:  true,
		}, nil
	case *ast.StarExpr:
		return base.QuerySpanResult{}, errors.Errorf("asterisk is not a scalar expression")
	case *ast.IntLit, *ast.FloatLit, *ast.StringLit, *ast.BoolLit, *ast.NullLit, *ast.HexLit, *ast.BitLit, *ast.TemporalLit:
		return base.QuerySpanResult{
			Name:          q.omniExprName(e),
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  true,
		}, nil
	case *ast.VariableRef:
		return base.QuerySpanResult{
			Name:          q.omniExprName(e),
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  true,
		}, nil
	case *ast.ParenExpr:
		field, err := q.extractOmniExpr(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		field.Name = q.omniExprName(e)
		field.IsPlainField = false
		return field, nil
	case *ast.BinaryExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Left, e.Right)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.UnaryExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Operand)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.FuncCallExpr:
		exprs := append([]ast.ExprNode{}, e.Args...)
		exprs = append(exprs, e.Separator)
		for _, item := range e.OrderBy {
			if item != nil {
				exprs = append(exprs, item.Expr)
			}
		}
		if e.Over != nil {
			exprs = append(exprs, e.Over.PartitionBy...)
			for _, item := range e.Over.OrderBy {
				if item != nil {
					exprs = append(exprs, item.Expr)
				}
			}
		}
		sourceColumns, err := q.mergeOmniExprSources(exprs...)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.SubqueryExpr:
		sourceColumns, isPlain, err := q.extractOmniSelectSourceColumns(e.Select)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: isPlain}, nil
	case *ast.ExistsExpr:
		sourceColumns, _, err := q.extractOmniSelectSourceColumns(e.Select)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.CaseExpr:
		var exprs []ast.ExprNode
		exprs = append(exprs, e.Operand)
		for _, when := range e.Whens {
			if when == nil {
				continue
			}
			exprs = append(exprs, when.Cond, when.Result)
		}
		exprs = append(exprs, e.Default)
		sourceColumns, err := q.mergeOmniExprSources(exprs...)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.BetweenExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Expr, e.Low, e.High)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.InExpr:
		exprs := append([]ast.ExprNode{e.Expr}, e.List...)
		sourceColumns, err := q.mergeOmniExprSources(exprs...)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		if e.Select != nil {
			selectSourceColumns, _, err := q.extractOmniSelectSourceColumns(e.Select)
			if err != nil {
				return base.QuerySpanResult{}, err
			}
			sourceColumns, _ = base.MergeSourceColumnSet(sourceColumns, selectSourceColumns)
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.LikeExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Expr, e.Pattern, e.Escape)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.IsExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.CastExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.ExtractExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.IntervalExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Value)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.CollateExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.MatchExpr:
		exprs := make([]ast.ExprNode, 0, len(e.Columns)+1)
		for _, column := range e.Columns {
			exprs = append(exprs, column)
		}
		exprs = append(exprs, e.Against)
		sourceColumns, err := q.mergeOmniExprSources(exprs...)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.ConvertExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Expr)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.DefaultExpr:
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: base.SourceColumnSet{}, IsPlainField: true}, nil
	case *ast.RowExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Items...)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	case *ast.MemberOfExpr:
		sourceColumns, err := q.mergeOmniExprSources(e.Value, e.Array)
		if err != nil {
			return base.QuerySpanResult{}, err
		}
		return base.QuerySpanResult{Name: q.omniExprName(e), SourceColumns: sourceColumns, IsPlainField: false}, nil
	default:
		return base.QuerySpanResult{}, errors.Errorf("unsupported omni MySQL expression %T", expr)
	}
}

func (q *omniQuerySpanExtractor) extractOmniSelectSourceColumns(selectStmt *ast.SelectStmt) (base.SourceColumnSet, bool, error) {
	subqueryExtractor := q.cloneOmniForSubquery()
	tableSource, err := subqueryExtractor.extractOmniSelectStmt(selectStmt)
	if err != nil {
		return nil, false, err
	}
	spanResults := tableSource.GetQuerySpanResult()
	sourceColumns := make(base.SourceColumnSet)
	for _, field := range spanResults {
		sourceColumns, _ = base.MergeSourceColumnSet(sourceColumns, field.SourceColumns)
	}
	isPlain := false
	if len(spanResults) == 1 {
		isPlain = spanResults[0].IsPlainField
	}
	return sourceColumns, isPlain, nil
}

func (q *omniQuerySpanExtractor) mergeOmniExprSources(exprs ...ast.ExprNode) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	for _, expr := range exprs {
		if expr == nil {
			continue
		}
		field, err := q.extractOmniExpr(expr)
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, field.SourceColumns)
	}
	return result, nil
}

func (q *omniQuerySpanExtractor) extractOmniColumnWild(columnRef *ast.ColumnRef, fromSources []base.TableSource) ([]base.QuerySpanResult, error) {
	if columnRef == nil {
		return nil, nil
	}
	return q.extractOmniTableWild(columnRef.Schema, columnRef.Table, fromSources)
}

func (q *omniQuerySpanExtractor) extractOmniTableWild(databaseName, tableName string, fromSources []base.TableSource) ([]base.QuerySpanResult, error) {
	if databaseName == "" && tableName == "" {
		var result []base.QuerySpanResult
		for _, fromSource := range fromSources {
			result = append(result, fromSource.GetQuerySpanResult()...)
		}
		return result, nil
	}

	result, ok := q.getAllTableColumnSources(databaseName, tableName)
	if !ok {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
			Table:    &tableName,
		}
	}
	return result, nil
}

func (q *omniQuerySpanExtractor) omniExprName(expr ast.ExprNode) string {
	switch e := expr.(type) {
	case *ast.ColumnRef:
		var parts []string
		if e.Schema != "" {
			parts = append(parts, e.Schema)
		}
		if e.Table != "" {
			parts = append(parts, e.Table)
		}
		if e.Star {
			parts = append(parts, "*")
		} else {
			parts = append(parts, e.Column)
		}
		return strings.Join(parts, ".")
	case *ast.BinaryExpr:
		return q.omniExprName(e.Left) + omniBinaryOpName(e) + q.omniExprName(e.Right)
	case *ast.InExpr:
		var list []string
		for _, expr := range e.List {
			list = append(list, q.omniExprName(expr))
		}
		operator := " in "
		if e.Not {
			operator = " not in "
		}
		return q.omniExprName(e.Expr) + operator + "(" + strings.Join(list, ", ") + ")"
	case *ast.IntLit:
		if name := q.omniSlice(e.Loc); name != "" {
			return name
		}
		return strconv.FormatInt(e.Value, 10)
	case *ast.FloatLit:
		if name := q.omniSlice(e.Loc); name != "" {
			return name
		}
		return e.Value
	case *ast.StringLit:
		if name := q.omniSlice(e.Loc); name != "" {
			return name
		}
		return e.Value
	case *ast.BoolLit:
		if name := q.omniSlice(e.Loc); name != "" {
			return name
		}
		return strconv.FormatBool(e.Value)
	case *ast.NullLit:
		if name := q.omniSlice(e.Loc); name != "" {
			return name
		}
		return "NULL"
	case *ast.HexLit:
		if name := q.omniSlice(e.Loc); name != "" {
			return name
		}
		return e.Value
	case *ast.BitLit:
		if name := q.omniSlice(e.Loc); name != "" {
			return name
		}
		return e.Value
	case *ast.TemporalLit:
		if name := q.omniSlice(e.Loc); name != "" {
			return name
		}
		return e.Value
	default:
		if loc, ok := omniNodeLoc(e); ok {
			return q.omniSlice(loc)
		}
		return ""
	}
}

func omniBinaryOpName(expr *ast.BinaryExpr) string {
	if expr == nil {
		return ""
	}
	if expr.OriginalOp != "" {
		return expr.OriginalOp
	}
	return omniBinaryOpNames[expr.Op]
}

var omniBinaryOpNames = map[ast.BinaryOp]string{
	ast.BinOpAdd:         "+",
	ast.BinOpSub:         "-",
	ast.BinOpMul:         "*",
	ast.BinOpDiv:         "/",
	ast.BinOpMod:         "%",
	ast.BinOpEq:          "=",
	ast.BinOpNe:          "!=",
	ast.BinOpLt:          "<",
	ast.BinOpGt:          ">",
	ast.BinOpLe:          "<=",
	ast.BinOpGe:          ">=",
	ast.BinOpAnd:         " and ",
	ast.BinOpOr:          " or ",
	ast.BinOpXor:         " xor ",
	ast.BinOpBitAnd:      "&",
	ast.BinOpBitOr:       "|",
	ast.BinOpBitXor:      "^",
	ast.BinOpShiftLeft:   "<<",
	ast.BinOpShiftRight:  ">>",
	ast.BinOpDivInt:      " div ",
	ast.BinOpRegexp:      " regexp ",
	ast.BinOpLikeEscape:  " like ",
	ast.BinOpNullSafeEq:  "<=>",
	ast.BinOpAssign:      ":=",
	ast.BinOpJsonExtract: "->",
	ast.BinOpJsonUnquote: "->>",
	ast.BinOpSoundsLike:  " sounds like ",
}

func (q *omniQuerySpanExtractor) omniSlice(loc ast.Loc) string {
	if loc.Start < 0 || loc.End <= loc.Start || loc.End > len(q.source) {
		return ""
	}
	return strings.TrimSpace(q.source[loc.Start:loc.End])
}

func omniNodeLoc(node ast.Node) (ast.Loc, bool) {
	if loc, ok := omniExprNodeLoc(node); ok {
		return loc, true
	}
	return omniLiteralNodeLoc(node)
}

func omniExprNodeLoc(node ast.Node) (ast.Loc, bool) {
	switch n := node.(type) {
	case *ast.ResTarget:
		return n.Loc, true
	case *ast.ColumnRef:
		return n.Loc, true
	case *ast.BinaryExpr:
		return n.Loc, true
	case *ast.UnaryExpr:
		return n.Loc, true
	case *ast.FuncCallExpr:
		return n.Loc, true
	case *ast.SubqueryExpr:
		return n.Loc, true
	case *ast.CaseExpr:
		return n.Loc, true
	case *ast.BetweenExpr:
		return n.Loc, true
	case *ast.InExpr:
		return n.Loc, true
	case *ast.LikeExpr:
		return n.Loc, true
	case *ast.IsExpr:
		return n.Loc, true
	case *ast.ExistsExpr:
		return n.Loc, true
	case *ast.CastExpr:
		return n.Loc, true
	case *ast.ExtractExpr:
		return n.Loc, true
	case *ast.IntervalExpr:
		return n.Loc, true
	case *ast.CollateExpr:
		return n.Loc, true
	case *ast.MatchExpr:
		return n.Loc, true
	case *ast.ConvertExpr:
		return n.Loc, true
	case *ast.DefaultExpr:
		return n.Loc, true
	case *ast.RowExpr:
		return n.Loc, true
	case *ast.MemberOfExpr:
		return n.Loc, true
	case *ast.ParenExpr:
		return n.Loc, true
	case *ast.StarExpr:
		return n.Loc, true
	default:
		return ast.Loc{}, false
	}
}

func omniLiteralNodeLoc(node ast.Node) (ast.Loc, bool) {
	switch n := node.(type) {
	case *ast.IntLit:
		return n.Loc, true
	case *ast.FloatLit:
		return n.Loc, true
	case *ast.StringLit:
		return n.Loc, true
	case *ast.BoolLit:
		return n.Loc, true
	case *ast.NullLit:
		return n.Loc, true
	case *ast.HexLit:
		return n.Loc, true
	case *ast.BitLit:
		return n.Loc, true
	case *ast.TemporalLit:
		return n.Loc, true
	case *ast.VariableRef:
		return n.Loc, true
	default:
		return ast.Loc{}, false
	}
}

func classifyOmniQueryType(node ast.Node, allSystems bool, source string) base.QueryType {
	switch n := node.(type) {
	case *ast.SelectStmt, *ast.TableStmt, *ast.ValuesStmt:
		if allSystems {
			return base.SelectInfoSchema
		}
		return base.Select
	case *ast.ExplainStmt:
		if isOmniDescribeStatement(source, n.Loc) {
			return base.SelectInfoSchema
		}
		if n.Analyze {
			switch n.Stmt.(type) {
			case *ast.SelectStmt, *ast.TableStmt, *ast.ValuesStmt:
				return base.Select
			default:
				return base.DML
			}
		}
		return base.Explain
	case *ast.ShowStmt:
		return base.SelectInfoSchema
	case *ast.SetStmt, *ast.SetDefaultRoleStmt, *ast.SetResourceGroupStmt, *ast.SetRoleStmt, *ast.SetTransactionStmt:
		return base.Select
	case *ast.SetPasswordStmt, *ast.HelpStmt:
		return base.QueryTypeUnknown
	case *ast.CreateDatabaseStmt, *ast.CreateTableStmt, *ast.CreateIndexStmt, *ast.CreateViewStmt,
		*ast.CreateEventStmt, *ast.CreateTriggerStmt, *ast.CreateFunctionStmt,
		*ast.CreateUserStmt, *ast.CreateRoleStmt, *ast.CreateTablespaceStmt, *ast.CreateServerStmt,
		*ast.CreateLogfileGroupStmt, *ast.CreateSpatialRefSysStmt, *ast.CreateResourceGroupStmt,
		*ast.AlterDatabaseStmt, *ast.AlterTableStmt, *ast.AlterViewStmt, *ast.AlterEventStmt,
		*ast.AlterUserStmt, *ast.AlterRoutineStmt, *ast.AlterTablespaceStmt, *ast.AlterServerStmt,
		*ast.AlterLogfileGroupStmt, *ast.AlterResourceGroupStmt, *ast.AlterInstanceStmt,
		*ast.DropDatabaseStmt, *ast.DropTableStmt, *ast.DropIndexStmt, *ast.DropViewStmt,
		*ast.DropEventStmt, *ast.DropTriggerStmt, *ast.DropRoutineStmt,
		*ast.DropUserStmt, *ast.DropRoleStmt, *ast.DropTablespaceStmt, *ast.DropServerStmt,
		*ast.DropLogfileGroupStmt, *ast.DropSpatialRefSysStmt, *ast.DropResourceGroupStmt,
		*ast.RenameTableStmt, *ast.RenameUserStmt, *ast.TruncateStmt, *ast.ImportTableStmt,
		*ast.GrantStmt, *ast.RevokeStmt, *ast.GrantRoleStmt, *ast.RevokeRoleStmt,
		*ast.InstallPluginStmt, *ast.UninstallPluginStmt, *ast.InstallComponentStmt,
		*ast.UninstallComponentStmt, *ast.CloneStmt:
		return base.DDL
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.BeginStmt, *ast.CommitStmt,
		*ast.RollbackStmt, *ast.SavepointStmt, *ast.LockTablesStmt, *ast.UnlockTablesStmt,
		*ast.LoadDataStmt, *ast.PrepareStmt, *ast.ExecuteStmt, *ast.DeallocateStmt,
		*ast.CallStmt, *ast.DoStmt, *ast.HandlerOpenStmt, *ast.HandlerReadStmt, *ast.HandlerCloseStmt,
		*ast.AnalyzeTableStmt, *ast.OptimizeTableStmt, *ast.CheckTableStmt, *ast.RepairTableStmt,
		*ast.FlushStmt, *ast.KillStmt, *ast.ShutdownStmt, *ast.RestartStmt, *ast.XAStmt,
		*ast.SignalStmt, *ast.ResignalStmt, *ast.GetDiagnosticsStmt, *ast.DeclareVarStmt,
		*ast.DeclareConditionStmt, *ast.DeclareHandlerStmt, *ast.DeclareCursorStmt,
		*ast.IfStmt, *ast.WhileStmt, *ast.RepeatStmt, *ast.LoopStmt, *ast.LeaveStmt,
		*ast.IterateStmt, *ast.ReturnStmt, *ast.OpenCursorStmt, *ast.FetchCursorStmt,
		*ast.CloseCursorStmt, *ast.LockInstanceStmt, *ast.UnlockInstanceStmt, *ast.BinlogStmt,
		*ast.CacheIndexStmt, *ast.LoadIndexIntoCacheStmt, *ast.ResetPersistStmt,
		*ast.ChangeReplicationSourceStmt, *ast.ChangeReplicationFilterStmt,
		*ast.StartReplicaStmt, *ast.StopReplicaStmt, *ast.ResetReplicaStmt,
		*ast.PurgeBinaryLogsStmt, *ast.ResetMasterStmt,
		*ast.StartGroupReplicationStmt, *ast.StopGroupReplicationStmt:
		return base.DML
	default:
		return base.QueryTypeUnknown
	}
}

func isOmniDescribeStatement(source string, loc ast.Loc) bool {
	statement := strings.TrimSpace(source)
	if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(source) {
		statement = strings.TrimSpace(source[loc.Start:loc.End])
	}
	upper := strings.ToUpper(statement)
	return strings.HasPrefix(upper, "DESCRIBE") || strings.HasPrefix(upper, "DESC ")
}

func collectOmniAccessTables(root ast.Node, defaultDatabase string, normalizeStarRocksCluster bool) base.SourceColumnSet {
	result := make(base.SourceColumnSet)
	collectOmniAccessTablesFromNode(result, root, defaultDatabase, normalizeStarRocksCluster)
	return result
}

func collectOmniAccessTablesFromNode(result base.SourceColumnSet, node ast.Node, defaultDatabase string, normalizeStarRocksCluster bool) {
	switch n := node.(type) {
	case nil:
	case *ast.SelectStmt:
		if n == nil {
			return
		}
		for _, cte := range n.CTEs {
			if cte != nil {
				collectOmniAccessTablesFromNode(result, cte.Select, defaultDatabase, normalizeStarRocksCluster)
			}
		}
		if n.Left != nil {
			collectOmniAccessTablesFromNode(result, n.Left, defaultDatabase, normalizeStarRocksCluster)
		}
		if n.Right != nil {
			collectOmniAccessTablesFromNode(result, n.Right, defaultDatabase, normalizeStarRocksCluster)
		}
		for _, target := range n.TargetList {
			collectOmniAccessTablesFromExpr(result, target, defaultDatabase, normalizeStarRocksCluster)
		}
		for _, tableExpr := range n.From {
			collectOmniAccessTablesFromTableExpr(result, tableExpr, defaultDatabase, normalizeStarRocksCluster)
		}
		collectOmniAccessTablesFromExpr(result, n.Where, defaultDatabase, normalizeStarRocksCluster)
		for _, groupBy := range n.GroupBy {
			collectOmniAccessTablesFromExpr(result, groupBy, defaultDatabase, normalizeStarRocksCluster)
		}
		collectOmniAccessTablesFromExpr(result, n.Having, defaultDatabase, normalizeStarRocksCluster)
		for _, orderBy := range n.OrderBy {
			if orderBy != nil {
				collectOmniAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase, normalizeStarRocksCluster)
			}
		}
	case *ast.ExplainStmt:
		collectOmniAccessTablesFromNode(result, n.Stmt, defaultDatabase, normalizeStarRocksCluster)
	case *ast.InsertStmt:
		collectOmniAccessTablesFromTableRef(result, n.Table, defaultDatabase, normalizeStarRocksCluster)
		for _, row := range n.Values {
			for _, expr := range row {
				collectOmniAccessTablesFromExpr(result, expr, defaultDatabase, normalizeStarRocksCluster)
			}
		}
		collectOmniAccessTablesFromNode(result, n.Select, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromNode(result, n.TableSource, defaultDatabase, normalizeStarRocksCluster)
		for _, assignment := range n.SetList {
			if assignment != nil {
				collectOmniAccessTablesFromExpr(result, assignment.Value, defaultDatabase, normalizeStarRocksCluster)
			}
		}
		for _, assignment := range n.OnDuplicateKey {
			if assignment != nil {
				collectOmniAccessTablesFromExpr(result, assignment.Value, defaultDatabase, normalizeStarRocksCluster)
			}
		}
	case *ast.UpdateStmt:
		for _, tableExpr := range n.Tables {
			collectOmniAccessTablesFromTableExpr(result, tableExpr, defaultDatabase, normalizeStarRocksCluster)
		}
		for _, assignment := range n.SetList {
			if assignment != nil {
				collectOmniAccessTablesFromExpr(result, assignment.Value, defaultDatabase, normalizeStarRocksCluster)
			}
		}
		collectOmniAccessTablesFromExpr(result, n.Where, defaultDatabase, normalizeStarRocksCluster)
		for _, orderBy := range n.OrderBy {
			if orderBy != nil {
				collectOmniAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase, normalizeStarRocksCluster)
			}
		}
	case *ast.DeleteStmt:
		for _, tableExpr := range n.Tables {
			collectOmniAccessTablesFromTableExpr(result, tableExpr, defaultDatabase, normalizeStarRocksCluster)
		}
		for _, tableExpr := range n.Using {
			collectOmniAccessTablesFromTableExpr(result, tableExpr, defaultDatabase, normalizeStarRocksCluster)
		}
		collectOmniAccessTablesFromExpr(result, n.Where, defaultDatabase, normalizeStarRocksCluster)
		for _, orderBy := range n.OrderBy {
			if orderBy != nil {
				collectOmniAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase, normalizeStarRocksCluster)
			}
		}
	case *ast.CreateTableStmt:
		collectOmniAccessTablesFromTableRef(result, n.Table, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromTableRef(result, n.Like, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromNode(result, n.Select, defaultDatabase, normalizeStarRocksCluster)
	case *ast.TableStmt:
		if n == nil {
			return
		}
		collectOmniAccessTablesFromTableRef(result, n.Table, defaultDatabase, normalizeStarRocksCluster)
	case *ast.ValuesStmt:
		for _, row := range n.Rows {
			for _, expr := range row {
				collectOmniAccessTablesFromExpr(result, expr, defaultDatabase, normalizeStarRocksCluster)
			}
		}
		for _, orderBy := range n.OrderBy {
			if orderBy != nil {
				collectOmniAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase, normalizeStarRocksCluster)
			}
		}
	case *ast.CallStmt:
		for _, arg := range n.Args {
			collectOmniAccessTablesFromExpr(result, arg, defaultDatabase, normalizeStarRocksCluster)
		}
	case *ast.DoStmt:
		for _, expr := range n.Exprs {
			collectOmniAccessTablesFromExpr(result, expr, defaultDatabase, normalizeStarRocksCluster)
		}
	case *ast.HandlerOpenStmt:
		collectOmniAccessTablesFromTableRef(result, n.Table, defaultDatabase, normalizeStarRocksCluster)
	case *ast.HandlerReadStmt:
		collectOmniAccessTablesFromTableRef(result, n.Table, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromExpr(result, n.Where, defaultDatabase, normalizeStarRocksCluster)
	case *ast.HandlerCloseStmt:
		collectOmniAccessTablesFromTableRef(result, n.Table, defaultDatabase, normalizeStarRocksCluster)
	default:
	}
}

func collectOmniAccessTablesFromTableRef(result base.SourceColumnSet, tableRef *ast.TableRef, defaultDatabase string, normalizeStarRocksCluster bool) {
	if tableRef == nil {
		return
	}
	if isOmniDualTable(tableRef) {
		return
	}
	database := tableRef.Schema
	if database == "" {
		database = defaultDatabase
	} else if normalizeStarRocksCluster {
		database = filterStarRocksClusterName(database)
	}
	result[base.ColumnResource{
		Database: database,
		Table:    tableRef.Name,
	}] = true
}

func isOmniDualTable(tableRef *ast.TableRef) bool {
	return tableRef != nil && tableRef.Schema == "" && strings.EqualFold(tableRef.Name, "dual")
}

func collectOmniAccessTablesFromTableExpr(result base.SourceColumnSet, tableExpr ast.TableExpr, defaultDatabase string, normalizeStarRocksCluster bool) {
	switch n := tableExpr.(type) {
	case *ast.TableRef:
		collectOmniAccessTablesFromTableRef(result, n, defaultDatabase, normalizeStarRocksCluster)
	case *ast.JoinClause:
		collectOmniAccessTablesFromTableExpr(result, n.Left, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromTableExpr(result, n.Right, defaultDatabase, normalizeStarRocksCluster)
		switch condition := n.Condition.(type) {
		case *ast.OnCondition:
			collectOmniAccessTablesFromExpr(result, condition.Expr, defaultDatabase, normalizeStarRocksCluster)
		default:
		}
	case *ast.SubqueryExpr:
		collectOmniAccessTablesFromNode(result, n.Select, defaultDatabase, normalizeStarRocksCluster)
	case *ast.JsonTableExpr:
		collectOmniAccessTablesFromExpr(result, n.Expr, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromExpr(result, n.Path, defaultDatabase, normalizeStarRocksCluster)
	default:
	}
}

func collectOmniAccessTablesFromExpr(result base.SourceColumnSet, expr ast.ExprNode, defaultDatabase string, normalizeStarRocksCluster bool) {
	switch e := expr.(type) {
	case nil:
	case *ast.ResTarget:
		collectOmniAccessTablesFromExpr(result, e.Val, defaultDatabase, normalizeStarRocksCluster)
	case *ast.BinaryExpr:
		collectOmniAccessTablesFromExpr(result, e.Left, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromExpr(result, e.Right, defaultDatabase, normalizeStarRocksCluster)
	case *ast.UnaryExpr:
		collectOmniAccessTablesFromExpr(result, e.Operand, defaultDatabase, normalizeStarRocksCluster)
	case *ast.FuncCallExpr:
		for _, arg := range e.Args {
			collectOmniAccessTablesFromExpr(result, arg, defaultDatabase, normalizeStarRocksCluster)
		}
		collectOmniAccessTablesFromExpr(result, e.Separator, defaultDatabase, normalizeStarRocksCluster)
		for _, orderBy := range e.OrderBy {
			if orderBy != nil {
				collectOmniAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase, normalizeStarRocksCluster)
			}
		}
		if e.Over != nil {
			for _, partitionBy := range e.Over.PartitionBy {
				collectOmniAccessTablesFromExpr(result, partitionBy, defaultDatabase, normalizeStarRocksCluster)
			}
			for _, orderBy := range e.Over.OrderBy {
				if orderBy != nil {
					collectOmniAccessTablesFromExpr(result, orderBy.Expr, defaultDatabase, normalizeStarRocksCluster)
				}
			}
		}
	case *ast.SubqueryExpr:
		collectOmniAccessTablesFromNode(result, e.Select, defaultDatabase, normalizeStarRocksCluster)
	case *ast.CaseExpr:
		collectOmniAccessTablesFromExpr(result, e.Operand, defaultDatabase, normalizeStarRocksCluster)
		for _, when := range e.Whens {
			if when == nil {
				continue
			}
			collectOmniAccessTablesFromExpr(result, when.Cond, defaultDatabase, normalizeStarRocksCluster)
			collectOmniAccessTablesFromExpr(result, when.Result, defaultDatabase, normalizeStarRocksCluster)
		}
		collectOmniAccessTablesFromExpr(result, e.Default, defaultDatabase, normalizeStarRocksCluster)
	case *ast.BetweenExpr:
		collectOmniAccessTablesFromExpr(result, e.Expr, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromExpr(result, e.Low, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromExpr(result, e.High, defaultDatabase, normalizeStarRocksCluster)
	case *ast.InExpr:
		collectOmniAccessTablesFromExpr(result, e.Expr, defaultDatabase, normalizeStarRocksCluster)
		for _, item := range e.List {
			collectOmniAccessTablesFromExpr(result, item, defaultDatabase, normalizeStarRocksCluster)
		}
		collectOmniAccessTablesFromNode(result, e.Select, defaultDatabase, normalizeStarRocksCluster)
	case *ast.LikeExpr:
		collectOmniAccessTablesFromExpr(result, e.Expr, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromExpr(result, e.Pattern, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromExpr(result, e.Escape, defaultDatabase, normalizeStarRocksCluster)
	case *ast.IsExpr:
		collectOmniAccessTablesFromExpr(result, e.Expr, defaultDatabase, normalizeStarRocksCluster)
	case *ast.ExistsExpr:
		collectOmniAccessTablesFromNode(result, e.Select, defaultDatabase, normalizeStarRocksCluster)
	case *ast.CastExpr:
		collectOmniAccessTablesFromExpr(result, e.Expr, defaultDatabase, normalizeStarRocksCluster)
	case *ast.ExtractExpr:
		collectOmniAccessTablesFromExpr(result, e.Expr, defaultDatabase, normalizeStarRocksCluster)
	case *ast.IntervalExpr:
		collectOmniAccessTablesFromExpr(result, e.Value, defaultDatabase, normalizeStarRocksCluster)
	case *ast.CollateExpr:
		collectOmniAccessTablesFromExpr(result, e.Expr, defaultDatabase, normalizeStarRocksCluster)
	case *ast.MatchExpr:
		for _, column := range e.Columns {
			collectOmniAccessTablesFromExpr(result, column, defaultDatabase, normalizeStarRocksCluster)
		}
		collectOmniAccessTablesFromExpr(result, e.Against, defaultDatabase, normalizeStarRocksCluster)
	case *ast.ConvertExpr:
		collectOmniAccessTablesFromExpr(result, e.Expr, defaultDatabase, normalizeStarRocksCluster)
	case *ast.RowExpr:
		for _, item := range e.Items {
			collectOmniAccessTablesFromExpr(result, item, defaultDatabase, normalizeStarRocksCluster)
		}
	case *ast.MemberOfExpr:
		collectOmniAccessTablesFromExpr(result, e.Value, defaultDatabase, normalizeStarRocksCluster)
		collectOmniAccessTablesFromExpr(result, e.Array, defaultDatabase, normalizeStarRocksCluster)
	case *ast.ParenExpr:
		collectOmniAccessTablesFromExpr(result, e.Expr, defaultDatabase, normalizeStarRocksCluster)
	default:
	}
}

func filterStarRocksClusterName(databaseName string) string {
	list := strings.Split(databaseName, ":")
	if len(list) > 1 {
		return list[len(list)-1]
	}
	return databaseName
}
