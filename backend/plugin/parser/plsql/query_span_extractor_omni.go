package plsql

import (
	"context"
	"reflect"
	"strings"
	"unicode"

	"github.com/pkg/errors"

	oracleast "github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type omniQuerySpanExtractor struct {
	*querySpanExtractor

	source                   string
	topLevelTableSourcesFrom []base.TableSource
}

func newOmniQuerySpanExtractor(connectionDatabase string, gCtx base.GetQuerySpanContext) *omniQuerySpanExtractor {
	return &omniQuerySpanExtractor{
		querySpanExtractor: newQuerySpanExtractor(connectionDatabase, gCtx),
	}
}

func (q *omniQuerySpanExtractor) getOmniQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx
	q.source = statement

	list, err := ParsePLSQLOmni(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement: %s", statement)
	}
	if list == nil || len(list.Items) == 0 {
		return nil, nil
	}
	if len(list.Items) > 1 {
		return nil, errors.Errorf("query span extraction only supports single statement, got %d statements", len(list.Items))
	}
	raw, ok := list.Items[0].(*oracleast.RawStmt)
	if !ok || raw.Stmt == nil {
		return nil, nil
	}

	accessTables := collectOmniAccessTables(q.defaultDatabase, raw.Stmt)
	allSystem, mixed := isMixedQuery(accessTables)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryType := omniQueryType(raw.Stmt, allSystem)
	if queryType != base.Select {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	columnSet := make(base.SourceColumnSet)
	for _, resource := range accessTables {
		if !q.existsTableMetadata(resource) {
			continue
		}
		columnSet[base.ColumnResource{
			Server:   resource.LinkedServer,
			Database: resource.Database,
			Table:    resource.Table,
		}] = true
	}

	selectStmt, ok := raw.Stmt.(*oracleast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("expected SELECT statement, got %T", raw.Stmt)
	}
	tableSource, err := q.extractOmniSelect(selectStmt)
	if err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			if len(columnSet) == 0 {
				columnSet[base.ColumnResource{
					Database: q.defaultDatabase,
				}] = true
			}
			return &base.QuerySpan{
				Type:          base.Select,
				SourceColumns: columnSet,
				Results:       []base.QuerySpanResult{},
				NotFoundError: resourceNotFound,
			}, nil
		}
		return nil, err
	}
	var results []base.QuerySpanResult
	if tableSource != nil {
		results = tableSource.GetQuerySpanResult()
	}
	if resourceNotFound := q.findMissingOmniAccessTable(raw.Stmt, accessTables); resourceNotFound != nil {
		if len(columnSet) == 0 {
			columnSet[base.ColumnResource{
				Database: q.defaultDatabase,
			}] = true
		}
		return &base.QuerySpan{
			Type:          base.Select,
			SourceColumns: columnSet,
			Results:       []base.QuerySpanResult{},
			NotFoundError: resourceNotFound,
		}, nil
	}
	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: columnSet,
		Results:       results,
	}, nil
}

func omniQueryType(stmt oracleast.StmtNode, allSystem bool) base.QueryType {
	if stmt == nil {
		return base.QueryTypeUnknown
	}
	switch stmt.(type) {
	case *oracleast.ExplainPlanStmt:
		return base.Explain
	case *oracleast.SelectStmt:
		if allSystem {
			return base.SelectInfoSchema
		}
		return base.Select
	case *oracleast.InsertStmt, *oracleast.UpdateStmt, *oracleast.DeleteStmt, *oracleast.MergeStmt,
		*oracleast.CallStmt, *oracleast.LockTableStmt:
		return base.DML
	case *oracleast.CreateTableStmt, *oracleast.AlterTableStmt, *oracleast.DropStmt,
		*oracleast.CreateIndexStmt, *oracleast.CreateViewStmt,
		*oracleast.CreateSequenceStmt, *oracleast.CreateSynonymStmt,
		*oracleast.CreateDatabaseLinkStmt, *oracleast.CreateTypeStmt,
		*oracleast.CreatePackageStmt, *oracleast.CreateProcedureStmt,
		*oracleast.CreateFunctionStmt, *oracleast.CreateTriggerStmt,
		*oracleast.TruncateStmt, *oracleast.AlterSessionStmt,
		*oracleast.AlterSystemStmt, *oracleast.CreateUserStmt,
		*oracleast.AlterUserStmt, *oracleast.CreateRoleStmt,
		*oracleast.AlterRoleStmt, *oracleast.CreateProfileStmt,
		*oracleast.AlterProfileStmt, *oracleast.AlterResourceCostStmt,
		*oracleast.AdminDDLStmt, *oracleast.CreateSchemaStmt,
		*oracleast.AlterDatabaseLinkStmt, *oracleast.AlterSynonymStmt,
		*oracleast.AlterMaterializedViewStmt,
		*oracleast.CreateAuditPolicyStmt, *oracleast.AlterAuditPolicyStmt,
		*oracleast.DropAuditPolicyStmt, *oracleast.CreateTablespaceStmt,
		*oracleast.AlterTablespaceStmt, *oracleast.CreateTablespaceSetStmt,
		*oracleast.DropTablespaceStmt, *oracleast.CreateClusterStmt,
		*oracleast.CreateDimensionStmt, *oracleast.AlterClusterStmt,
		*oracleast.AlterDimensionStmt, *oracleast.CreateMaterializedZonemapStmt,
		*oracleast.AlterMaterializedZonemapStmt, *oracleast.CreateInmemoryJoinGroupStmt,
		*oracleast.AlterInmemoryJoinGroupStmt, *oracleast.AlterIndexStmt,
		*oracleast.AlterViewStmt, *oracleast.AlterSequenceStmt,
		*oracleast.AlterProcedureStmt, *oracleast.AlterFunctionStmt,
		*oracleast.AlterPackageStmt, *oracleast.AlterTriggerStmt,
		*oracleast.AlterTypeStmt, *oracleast.CreateIndextypeStmt,
		*oracleast.AlterIndextypeStmt, *oracleast.CreateOperatorStmt,
		*oracleast.AlterOperatorStmt, *oracleast.CreateMviewLogStmt,
		*oracleast.AlterMviewLogStmt, *oracleast.CreateAnalyticViewStmt,
		*oracleast.AlterAnalyticViewStmt, *oracleast.CreateJsonDualityViewStmt,
		*oracleast.AlterJsonDualityViewStmt, *oracleast.CreateAttributeDimensionStmt,
		*oracleast.AlterAttributeDimensionStmt, *oracleast.CreateHierarchyStmt,
		*oracleast.AlterHierarchyStmt, *oracleast.CreateDomainStmt,
		*oracleast.AlterDomainStmt, *oracleast.CreatePropertyGraphStmt,
		*oracleast.CreateVectorIndexStmt, *oracleast.CreateLockdownProfileStmt,
		*oracleast.AlterLockdownProfileStmt, *oracleast.CreateOutlineStmt,
		*oracleast.AlterOutlineStmt:
		return base.DDL
	}
	return base.QueryTypeUnknown
}

func (q *omniQuerySpanExtractor) findMissingOmniAccessTable(stmt oracleast.StmtNode, accessTables []base.SchemaResource) *base.ResourceNotFoundError {
	cteNames := collectOmniCTENames(stmt)
	for _, resource := range accessTables {
		if cteNames[resource.Table] || q.existsTableMetadata(resource) {
			continue
		}
		database := resource.Database
		if database == "" {
			database = q.defaultDatabase
		}
		table := resource.Table
		return &base.ResourceNotFoundError{
			Database: &database,
			Table:    &table,
		}
	}
	return nil
}

func collectOmniCTENames(stmt oracleast.StmtNode) map[string]bool {
	result := make(map[string]bool)
	oracleast.Inspect(stmt, func(node oracleast.Node) bool {
		cte, ok := node.(*oracleast.CTE)
		if ok {
			result[cte.Name] = true
		}
		return true
	})
	return result
}

func collectOmniAccessTables(defaultDatabase string, stmt oracleast.StmtNode) []base.SchemaResource {
	seen := make(map[base.SchemaResource]bool)
	var result []base.SchemaResource
	addResource := func(name *oracleast.ObjectName) {
		if name == nil || name.Name == "DUAL" || name.DBLink != "" {
			return
		}
		database := name.Schema
		if database == "" {
			database = defaultDatabase
		}
		resource := base.SchemaResource{
			Database: database,
			Table:    name.Name,
		}
		if !seen[resource] {
			seen[resource] = true
			result = append(result, resource)
		}
	}
	oracleast.Inspect(stmt, func(node oracleast.Node) bool {
		switch node := node.(type) {
		case *oracleast.TableRef:
			if node.Dblink == "" {
				addResource(node.Name)
			}
		case *oracleast.ContainersExpr:
			addResource(node.Name)
		default:
		}
		return true
	})
	return result
}

func (q *omniQuerySpanExtractor) clone() *omniQuerySpanExtractor {
	copyExtractor := *q.querySpanExtractor
	return &omniQuerySpanExtractor{
		querySpanExtractor:       &copyExtractor,
		source:                   q.source,
		topLevelTableSourcesFrom: cloneTableSourceSlice(q.topLevelTableSourcesFrom),
	}
}

func (q *omniQuerySpanExtractor) extractOmniSelect(stmt *oracleast.SelectStmt) (base.TableSource, error) {
	if stmt == nil {
		return nil, nil
	}

	oldCTEs := q.ctes
	if stmt.WithClause != nil {
		for _, node := range listItems(stmt.WithClause.CTEs) {
			cte, ok := node.(*oracleast.CTE)
			if !ok {
				continue
			}
			table, err := q.extractOmniCTE(cte)
			if err != nil {
				q.ctes = oldCTEs
				return nil, err
			}
			q.ctes = append(q.ctes, table)
		}
	}
	defer func() {
		q.ctes = oldCTEs
	}()

	if stmt.Op != 0 {
		return q.extractOmniSetSelect(stmt)
	}
	if stmt.Pivot != nil {
		source, err := q.extractOmniPivot(stmt.Pivot)
		if err != nil {
			return nil, err
		}
		return q.projectOmniTransformedSelect(stmt, source)
	}
	if stmt.Unpivot != nil {
		source, err := q.extractOmniUnpivot(stmt.Unpivot)
		if err != nil {
			return nil, err
		}
		return q.projectOmniTransformedSelect(stmt, source)
	}
	if stmt.ModelClause != nil {
		source, err := q.extractOmniModelSelect(stmt)
		if err != nil {
			return nil, err
		}
		return q.projectOmniTransformedSelect(stmt, source)
	}

	oldFrom := q.tableSourcesFrom
	oldTopLevelFrom := q.topLevelTableSourcesFrom
	q.tableSourcesFrom = nil
	q.topLevelTableSourcesFrom = nil
	defer func() {
		q.tableSourcesFrom = oldFrom
		q.topLevelTableSourcesFrom = oldTopLevelFrom
	}()

	for _, node := range listItems(stmt.FromClause) {
		tableExpr, ok := node.(oracleast.TableExpr)
		if !ok {
			continue
		}
		tableSource, err := q.extractOmniTableExpr(tableExpr)
		if err != nil {
			return nil, err
		}
		if tableSource != nil {
			q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
			q.topLevelTableSourcesFrom = append(q.topLevelTableSourcesFrom, tableSource)
		}
	}

	results, err := q.extractOmniTargetList(stmt.TargetList)
	if err != nil {
		return nil, err
	}
	return &base.PseudoTable{
		Columns: results,
	}, nil
}

func (q *omniQuerySpanExtractor) projectOmniTransformedSelect(stmt *oracleast.SelectStmt, source base.TableSource) (base.TableSource, error) {
	oldFrom := q.tableSourcesFrom
	oldTopLevelFrom := q.topLevelTableSourcesFrom
	q.tableSourcesFrom = nil
	q.topLevelTableSourcesFrom = nil
	if source != nil {
		q.tableSourcesFrom = append(q.tableSourcesFrom, source)
		q.topLevelTableSourcesFrom = append(q.topLevelTableSourcesFrom, source)
	}
	defer func() {
		q.tableSourcesFrom = oldFrom
		q.topLevelTableSourcesFrom = oldTopLevelFrom
	}()

	results, err := q.extractOmniTargetList(stmt.TargetList)
	if err != nil {
		return nil, err
	}
	return &base.PseudoTable{Columns: results}, nil
}

func (q *omniQuerySpanExtractor) extractOmniSetSelect(stmt *oracleast.SelectStmt) (base.TableSource, error) {
	left, err := q.extractOmniSelect(omniSetLeftSelect(stmt))
	if err != nil {
		return nil, err
	}
	right, err := q.extractOmniSelect(stmt.Rarg)
	if err != nil {
		return nil, err
	}
	return mergeOmniSetTableSources(left, right)
}

func mergeOmniSetTableSources(left, right base.TableSource) (base.TableSource, error) {
	if left == nil {
		return right, nil
	}
	if right == nil {
		return left, nil
	}
	leftResults := left.GetQuerySpanResult()
	rightResults := right.GetQuerySpanResult()
	if len(leftResults) != len(rightResults) {
		return nil, errors.Errorf("left and right query span result length mismatch: %d != %d", len(leftResults), len(rightResults))
	}
	result := make([]base.QuerySpanResult, 0, len(leftResults))
	for i, leftResult := range leftResults {
		sourceColumns, _ := base.MergeSourceColumnSet(leftResult.SourceColumns, rightResults[i].SourceColumns)
		result = append(result, base.QuerySpanResult{
			Name:          leftResult.Name,
			SourceColumns: sourceColumns,
			IsPlainField:  false,
		})
	}
	return &base.PseudoTable{Columns: result}, nil
}

func (q *omniQuerySpanExtractor) extractOmniCTE(cte *oracleast.CTE) (*base.PseudoTable, error) {
	if cte == nil {
		return nil, nil
	}
	name := cte.Name
	columnNames := omniStringList(cte.Columns)

	selectStmt, ok := cte.Query.(*oracleast.SelectStmt)
	if ok && selectStmt.Op != 0 && len(columnNames) > 0 {
		initialSource, err := q.extractOmniSelect(omniSetLeftSelect(selectStmt))
		if err != nil {
			return nil, err
		}
		initial := cloneQuerySpanResults(initialSource.GetQuerySpanResult())
		applyOmniColumnAliases(initial, columnNames)

		placeholder := &base.PseudoTable{Name: name, Columns: initial}
		q.ctes = append(q.ctes, placeholder)
		defer func() {
			q.ctes = q.ctes[:len(q.ctes)-1]
		}()

		columns := initial
		for range 16 {
			placeholder.Columns = columns
			rightSource, err := q.extractOmniSelect(selectStmt.Rarg)
			if err != nil {
				return nil, err
			}
			mergedSource, err := mergeOmniSetTableSources(&base.PseudoTable{Columns: initial}, rightSource)
			if err != nil {
				return nil, err
			}
			next := cloneQuerySpanResults(mergedSource.GetQuerySpanResult())
			applyOmniColumnAliases(next, columnNames)
			if querySpanResultsEqual(columns, next) {
				return &base.PseudoTable{Name: name, Columns: next}, nil
			}
			columns = next
		}
		return &base.PseudoTable{Name: name, Columns: columns}, nil
	}

	child := q.clone()
	tableSource, err := child.extractOmniStmt(cte.Query)
	if err != nil {
		return nil, err
	}
	columns := cloneQuerySpanResults(tableSource.GetQuerySpanResult())
	applyOmniColumnAliases(columns, columnNames)
	return &base.PseudoTable{Name: name, Columns: columns}, nil
}

func (q *omniQuerySpanExtractor) extractOmniStmt(stmt oracleast.StmtNode) (base.TableSource, error) {
	selectStmt, ok := stmt.(*oracleast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("unsupported statement in query span extractor: %T", stmt)
	}
	return q.extractOmniSelect(selectStmt)
}

func (q *omniQuerySpanExtractor) extractOmniTargetList(list *oracleast.List) ([]base.QuerySpanResult, error) {
	if list == nil || list.Len() == 0 {
		return q.expandOmniAsterisk("", "")
	}

	var results []base.QuerySpanResult
	for _, node := range listItems(list) {
		target, ok := node.(*oracleast.ResTarget)
		if !ok || target.Expr == nil {
			continue
		}
		if isOmniStar(target.Expr) {
			expanded, err := q.expandOmniAsterisk("", "")
			if err != nil {
				return nil, err
			}
			results = append(results, expanded...)
			continue
		}
		if ref, ok := target.Expr.(*oracleast.ColumnRef); ok && ref.Column == "*" {
			expanded, err := q.expandOmniAsterisk(ref.Schema, ref.Table)
			if err != nil {
				return nil, err
			}
			results = append(results, expanded...)
			continue
		}

		name, sourceColumns, err := q.extractOmniExpr(target.Expr)
		if err != nil {
			return nil, err
		}
		if target.Name != "" {
			name = target.Name
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: sourceColumns,
			IsPlainField:  false,
		})
	}
	return results, nil
}

func (q *omniQuerySpanExtractor) extractOmniTableExpr(expr oracleast.TableExpr) (base.TableSource, error) {
	switch expr := expr.(type) {
	case *oracleast.TableRef:
		if expr.Name == nil {
			return nil, nil
		}
		dbLink := expr.Name.DBLink
		if dbLink == "" {
			dbLink = expr.Dblink
		}
		database := expr.Name.Schema
		if database == "" && dbLink == "" {
			database = q.defaultDatabase
		}
		tableSource, err := q.plsqlFindTableSchema(splitOmniDBLink(dbLink), database, expr.Name.Name)
		if err != nil {
			return nil, err
		}
		return aliasOmniTableSource(tableSource, expr.Alias), nil
	case *oracleast.SubqueryRef:
		child := q.clone()
		tableSource, err := child.extractOmniStmt(expr.Subquery)
		if err != nil {
			return nil, err
		}
		return aliasOmniTableSource(tableSource, expr.Alias), nil
	case *oracleast.SubqueryExpr:
		child := q.clone()
		tableSource, err := child.extractOmniStmt(expr.Subquery)
		if err != nil {
			return nil, err
		}
		return tableSource, nil
	case *oracleast.LateralRef:
		child := q.clone()
		child.outerTableSources = append(cloneTableSourceSlice(child.outerTableSources), q.tableSourcesFrom...)
		tableSource, err := child.extractOmniStmt(expr.Subquery)
		if err != nil {
			return nil, err
		}
		return aliasOmniTableSource(tableSource, expr.Alias), nil
	case *oracleast.JoinClause:
		left, err := q.extractOmniTableExpr(expr.Left)
		if err != nil {
			return nil, err
		}
		if left != nil {
			q.tableSourcesFrom = append(q.tableSourcesFrom, left)
		}
		right, err := q.extractOmniTableExpr(expr.Right)
		if err != nil {
			return nil, err
		}
		if right != nil {
			q.tableSourcesFrom = append(q.tableSourcesFrom, right)
		}
		return mergeOmniJoinTableSource(expr, left, right)
	case *oracleast.XmlTableRef:
		return q.extractOmniXMLTable(expr)
	case *oracleast.JsonTableRef:
		return q.extractOmniJSONTable(expr)
	case *oracleast.ContainersExpr:
		if expr.Name == nil {
			return nil, nil
		}
		database := expr.Name.Schema
		if database == "" {
			database = q.defaultDatabase
		}
		tableSource, err := q.plsqlFindTableSchema(nil, database, expr.Name.Name)
		if err != nil {
			return nil, err
		}
		return aliasOmniTableSource(tableSource, expr.Alias), nil
	case *oracleast.InlineExternalTable:
		return extractOmniInlineExternalTable(expr), nil
	case *oracleast.TableCollectionExpr:
		return q.extractOmniTableCollection(expr)
	case *oracleast.PivotClause:
		return q.extractOmniPivot(expr)
	case *oracleast.UnpivotClause:
		return q.extractOmniUnpivot(expr)
	case *oracleast.MatchRecognizeClause:
		return q.extractOmniMatchRecognize(expr)
	default:
		return nil, errors.Errorf("unsupported oracle table source: %T", expr)
	}
}

func (q *omniQuerySpanExtractor) extractOmniPivot(pivot *oracleast.PivotClause) (base.TableSource, error) {
	if pivot == nil {
		return nil, nil
	}
	source, err := q.extractOmniTableExpr(pivot.Source)
	if err != nil {
		return nil, err
	}
	oldFrom := q.tableSourcesFrom
	q.tableSourcesFrom = append(q.tableSourcesFrom, source)
	defer func() {
		q.tableSourcesFrom = oldFrom
	}()

	excluded := make(map[string]bool)
	pivotSources := make(base.SourceColumnSet)
	for _, expr := range omniExprList(pivot.ForColumns) {
		name, sourceColumns, err := q.extractOmniExpr(expr)
		if err != nil {
			return nil, err
		}
		excluded[name] = true
		pivotSources, _ = base.MergeSourceColumnSet(pivotSources, sourceColumns)
	}

	var aggregates []*oracleast.PivotAggregate
	for _, item := range listItems(pivot.Aggregates) {
		aggregate, ok := item.(*oracleast.PivotAggregate)
		if ok {
			aggregates = append(aggregates, aggregate)
		}
	}
	var inItems []*oracleast.PivotInItem
	for _, item := range listItems(pivot.InItems) {
		inItem, ok := item.(*oracleast.PivotInItem)
		if ok {
			inItems = append(inItems, inItem)
		}
	}

	aggregateSources := make([]base.SourceColumnSet, len(aggregates))
	for i, aggregate := range aggregates {
		_, sourceColumns, err := q.extractOmniExpr(aggregate.Expr)
		if err != nil {
			return nil, err
		}
		aggregateSources[i] = sourceColumns
		excludeOmniColumnRefNames(excluded, aggregate.Expr)
		excludeSourceColumnNames(excluded, sourceColumns)
	}

	var results []base.QuerySpanResult
	if source != nil {
		for _, result := range source.GetQuerySpanResult() {
			if !excluded[result.Name] {
				results = append(results, cloneQuerySpanResult(result))
			}
		}
	}
	for _, inItem := range inItems {
		inName := pivotInItemName(q, inItem)
		for i, aggregate := range aggregates {
			sourceColumns, _ := base.MergeSourceColumnSet(pivotSources, aggregateSources[i])
			results = append(results, base.QuerySpanResult{
				Name:          pivotColumnName(inName, aggregate.Alias, len(aggregates)),
				SourceColumns: sourceColumns,
				IsPlainField:  false,
			})
		}
	}
	return aliasOmniTableSource(&base.PseudoTable{Columns: results}, pivot.Alias), nil
}

func (q *omniQuerySpanExtractor) extractOmniUnpivot(unpivot *oracleast.UnpivotClause) (base.TableSource, error) {
	if unpivot == nil {
		return nil, nil
	}
	source, err := q.extractOmniTableExpr(unpivot.Source)
	if err != nil {
		return nil, err
	}
	oldFrom := q.tableSourcesFrom
	q.tableSourcesFrom = append(q.tableSourcesFrom, source)
	defer func() {
		q.tableSourcesFrom = oldFrom
	}()

	valueColumns := omniExprList(unpivot.ValueColumns)
	inputSources := make([]base.SourceColumnSet, len(valueColumns))
	excluded := make(map[string]bool)
	for _, item := range listItems(unpivot.InputMappings) {
		mapping, ok := item.(*oracleast.UnpivotInItem)
		if !ok {
			continue
		}
		for i, input := range omniExprList(mapping.InputColumns) {
			name, sourceColumns, err := q.extractOmniExpr(input)
			if err != nil {
				return nil, err
			}
			excluded[name] = true
			if i < len(inputSources) {
				inputSources[i], _ = base.MergeSourceColumnSet(inputSources[i], sourceColumns)
			}
		}
	}

	var results []base.QuerySpanResult
	if source != nil {
		for _, result := range source.GetQuerySpanResult() {
			if !excluded[result.Name] {
				results = append(results, cloneQuerySpanResult(result))
			}
		}
	}
	for i, expr := range valueColumns {
		name, _, err := q.extractOmniExpr(expr)
		if err != nil {
			return nil, err
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: inputSources[i],
			IsPlainField:  false,
		})
	}
	if unpivot.PivotColumn != nil {
		name, _, err := q.extractOmniExpr(unpivot.PivotColumn)
		if err != nil {
			return nil, err
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  false,
		})
	}
	return aliasOmniTableSource(&base.PseudoTable{Columns: results}, unpivot.Alias), nil
}

func (q *omniQuerySpanExtractor) extractOmniModelSelect(stmt *oracleast.SelectStmt) (base.TableSource, error) {
	oldFrom := q.tableSourcesFrom
	oldTopLevelFrom := q.topLevelTableSourcesFrom
	q.tableSourcesFrom = nil
	q.topLevelTableSourcesFrom = nil
	defer func() {
		q.tableSourcesFrom = oldFrom
		q.topLevelTableSourcesFrom = oldTopLevelFrom
	}()

	for _, node := range listItems(stmt.FromClause) {
		tableExpr, ok := node.(oracleast.TableExpr)
		if !ok {
			continue
		}
		tableSource, err := q.extractOmniTableExpr(tableExpr)
		if err != nil {
			return nil, err
		}
		if tableSource != nil {
			q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
			q.topLevelTableSourcesFrom = append(q.topLevelTableSourcesFrom, tableSource)
		}
	}

	var results []base.QuerySpanResult
	if stmt.ModelClause != nil && stmt.ModelClause.MainModel != nil && stmt.ModelClause.MainModel.ColumnClauses != nil {
		columns := stmt.ModelClause.MainModel.ColumnClauses
		for _, list := range []*oracleast.List{columns.PartitionBy, columns.DimensionBy, columns.Measures} {
			if list == nil {
				continue
			}
			extracted, err := q.extractOmniTargetList(list)
			if err != nil {
				return nil, err
			}
			results = append(results, extracted...)
		}
	}
	return &base.PseudoTable{Columns: results}, nil
}

func (*omniQuerySpanExtractor) extractOmniTableCollection(ref *oracleast.TableCollectionExpr) (base.TableSource, error) {
	if ref == nil {
		return nil, nil
	}
	var columns []base.QuerySpanResult
	for _, name := range omniStringList(ref.ColumnAliases) {
		columns = append(columns, base.QuerySpanResult{
			Name:          name,
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  false,
		})
	}
	if len(columns) == 0 {
		return nil, errors.Errorf("unsupported oracle table collection without column aliases")
	}
	return aliasOmniTableSource(&base.PseudoTable{Columns: columns}, ref.Alias), nil
}

func (q *omniQuerySpanExtractor) extractOmniMatchRecognize(ref *oracleast.MatchRecognizeClause) (base.TableSource, error) {
	if ref == nil {
		return nil, nil
	}
	source, err := q.extractOmniTableExpr(ref.Source)
	if err != nil {
		return nil, err
	}
	oldFrom := q.tableSourcesFrom
	q.tableSourcesFrom = append(q.tableSourcesFrom, source)
	defer func() {
		q.tableSourcesFrom = oldFrom
	}()

	partitionResults, err := q.extractOmniResultList(ref.PartitionBy)
	if err != nil {
		return nil, err
	}
	measureResults, err := q.extractOmniMatchRecognizeMeasureList(ref.Measures)
	if err != nil {
		return nil, err
	}
	var results []base.QuerySpanResult
	if strings.HasPrefix(strings.ToUpper(ref.RowsPerMatch), "ALL ROWS PER MATCH") && source != nil {
		results = cloneQuerySpanResults(source.GetQuerySpanResult())
	} else {
		results = append(results, partitionResults...)
	}
	results = append(results, measureResults...)
	return aliasOmniTableSource(&base.PseudoTable{Columns: results}, ref.Alias), nil
}

func (q *omniQuerySpanExtractor) extractOmniResultList(list *oracleast.List) ([]base.QuerySpanResult, error) {
	return q.extractOmniResultListWithSourceColumns(list, nil)
}

func (q *omniQuerySpanExtractor) extractOmniMatchRecognizeMeasureList(list *oracleast.List) ([]base.QuerySpanResult, error) {
	return q.extractOmniResultListWithSourceColumns(list, q.extractOmniMatchRecognizeMeasureSourceColumns)
}

func (q *omniQuerySpanExtractor) extractOmniResultListWithSourceColumns(list *oracleast.List, sourceColumnsExtractor func(oracleast.ExprNode) (base.SourceColumnSet, error)) ([]base.QuerySpanResult, error) {
	var results []base.QuerySpanResult
	for _, node := range listItems(list) {
		var expr oracleast.ExprNode
		name := ""
		switch node := node.(type) {
		case *oracleast.ResTarget:
			expr = node.Expr
			name = node.Name
		case oracleast.ExprNode:
			expr = node
		default:
			continue
		}
		if expr == nil {
			continue
		}

		extractedName, sourceColumns, err := q.extractOmniExpr(expr)
		if err != nil {
			return nil, err
		}
		if sourceColumnsExtractor != nil {
			sourceColumns, err = sourceColumnsExtractor(expr)
			if err != nil {
				return nil, err
			}
		}
		if name == "" {
			name = extractedName
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: sourceColumns,
			IsPlainField:  false,
		})
	}
	return results, nil
}

func (q *omniQuerySpanExtractor) extractOmniMatchRecognizeMeasureSourceColumns(expr oracleast.ExprNode) (base.SourceColumnSet, error) {
	return q.extractOmniExprSourceColumnsWithResolver(expr, func(node *oracleast.ColumnRef) base.SourceColumnSet {
		sourceColumns := q.getOmniFieldColumnSource(node.Schema, node.Table, node.Column)
		if len(sourceColumns) == 0 && node.Schema == "" && node.Table != "" {
			sourceColumns = q.getOmniFieldColumnSource("", "", node.Column)
		}
		return sourceColumns
	})
}

func omniSetLeftSelect(stmt *oracleast.SelectStmt) *oracleast.SelectStmt {
	if stmt == nil {
		return nil
	}
	if stmt.Larg != nil {
		return stmt.Larg
	}
	body := *stmt
	body.Op = 0
	body.SetAll = false
	body.Larg = nil
	body.Rarg = nil
	return &body
}

func mergeOmniJoinTableSource(join *oracleast.JoinClause, left, right base.TableSource) (base.TableSource, error) {
	if left == nil {
		return right, nil
	}
	if right == nil {
		return left, nil
	}
	leftResults := left.GetQuerySpanResult()
	rightResults := right.GetQuerySpanResult()
	result := new(base.PseudoTable)

	leftIndex := make(map[string]int)
	rightIndex := make(map[string]int)
	for i, field := range leftResults {
		leftIndex[field.Name] = i
	}
	for i, field := range rightResults {
		rightIndex[field.Name] = i
	}

	if isOmniNaturalJoin(join.Type) {
		for _, field := range leftResults {
			if rightIdx, ok := rightIndex[field.Name]; ok {
				field.SourceColumns, _ = base.MergeSourceColumnSet(field.SourceColumns, rightResults[rightIdx].SourceColumns)
			}
			result.Columns = append(result.Columns, field)
		}
		for _, field := range rightResults {
			if _, ok := leftIndex[field.Name]; !ok {
				result.Columns = append(result.Columns, field)
			}
		}
		return result, nil
	}

	if len(listItems(join.Using)) != 0 {
		usingMap := make(map[string]bool)
		for _, node := range listItems(join.Using) {
			switch node := node.(type) {
			case *oracleast.String:
				usingMap[node.Str] = true
			case *oracleast.ColumnRef:
				usingMap[node.Column] = true
			default:
			}
		}
		for _, field := range leftResults {
			if usingMap[field.Name] {
				if rightIdx, ok := rightIndex[field.Name]; ok {
					field.SourceColumns, _ = base.MergeSourceColumnSet(field.SourceColumns, rightResults[rightIdx].SourceColumns)
				}
			}
			result.Columns = append(result.Columns, field)
		}
		for _, field := range rightResults {
			if !usingMap[field.Name] {
				result.Columns = append(result.Columns, field)
			}
		}
		return result, nil
	}

	result.Columns = append(result.Columns, leftResults...)
	result.Columns = append(result.Columns, rightResults...)
	return result, nil
}

func isOmniNaturalJoin(joinType oracleast.JoinType) bool {
	switch joinType {
	case oracleast.JOIN_NATURAL_INNER, oracleast.JOIN_NATURAL_LEFT, oracleast.JOIN_NATURAL_RIGHT, oracleast.JOIN_NATURAL_FULL:
		return true
	default:
		return false
	}
}

func (q *omniQuerySpanExtractor) extractOmniExpr(expr oracleast.ExprNode) (string, base.SourceColumnSet, error) {
	if expr == nil {
		return "", base.SourceColumnSet{}, nil
	}

	switch expr := expr.(type) {
	case *oracleast.ColumnRef:
		if expr.Column == "*" {
			return "*", base.SourceColumnSet{}, nil
		}
		return expr.Column, q.getOmniFieldColumnSource(expr.Schema, expr.Table, expr.Column), nil
	case *oracleast.Star:
		return "*", base.SourceColumnSet{}, nil
	case *oracleast.NumberLiteral:
		return q.omniExprName(expr.Loc, expr.Val), base.SourceColumnSet{}, nil
	case *oracleast.StringLiteral:
		return q.omniExprName(expr.Loc, expr.Val), base.SourceColumnSet{}, nil
	case *oracleast.NullLiteral:
		return q.omniExprName(expr.Loc, "NULL"), base.SourceColumnSet{}, nil
	case *oracleast.DateTimeLiteral:
		return q.omniExprName(expr.Loc, expr.TypeName+" "+expr.Val), base.SourceColumnSet{}, nil
	case *oracleast.SubqueryExpr:
		child := q.clone()
		child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
		tableSource, err := child.extractOmniStmt(expr.Subquery)
		if err != nil {
			return "", nil, err
		}
		return q.omniExprName(expr.Loc, ""), mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()), nil
	case *oracleast.ExistsExpr:
		child := q.clone()
		child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
		tableSource, err := child.extractOmniStmt(expr.Subquery)
		if err != nil {
			return "", nil, err
		}
		return q.omniExprName(expr.Loc, ""), mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()), nil
	case *oracleast.CursorExpr:
		child := q.clone()
		child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
		tableSource, err := child.extractOmniStmt(expr.Subquery)
		if err != nil {
			return "", nil, err
		}
		return q.omniExprName(expr.Loc, ""), mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()), nil
	default:
		name := q.omniExprName(getOmniExprFullLoc(expr), oracleast.NodeToString(expr))
		sourceColumns, err := q.extractOmniExprSourceColumns(expr)
		return name, sourceColumns, err
	}
}

func (q *omniQuerySpanExtractor) extractOmniExprSourceColumns(expr oracleast.ExprNode) (base.SourceColumnSet, error) {
	return q.extractOmniExprSourceColumnsWithResolver(expr, func(node *oracleast.ColumnRef) base.SourceColumnSet {
		return q.getOmniFieldColumnSource(node.Schema, node.Table, node.Column)
	})
}

func (q *omniQuerySpanExtractor) extractOmniExprSourceColumnsWithResolver(expr oracleast.ExprNode, resolveColumn func(*oracleast.ColumnRef) base.SourceColumnSet) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	var walkErr error
	oracleast.Inspect(expr, func(node oracleast.Node) bool {
		if walkErr != nil || node == nil {
			return false
		}
		switch node := node.(type) {
		case *oracleast.ColumnRef:
			if node.Column != "*" {
				result, _ = base.MergeSourceColumnSet(result, resolveColumn(node))
			}
			return false
		case *oracleast.FuncCallExpr:
			return true
		case *oracleast.SubqueryExpr:
			child := q.clone()
			child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
			tableSource, err := child.extractOmniStmt(node.Subquery)
			if err != nil {
				walkErr = err
				return false
			}
			result, _ = base.MergeSourceColumnSet(result, mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()))
			return false
		case *oracleast.ExistsExpr:
			child := q.clone()
			child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
			tableSource, err := child.extractOmniStmt(node.Subquery)
			if err != nil {
				walkErr = err
				return false
			}
			result, _ = base.MergeSourceColumnSet(result, mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()))
			return false
		case *oracleast.CursorExpr:
			child := q.clone()
			child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
			tableSource, err := child.extractOmniStmt(node.Subquery)
			if err != nil {
				walkErr = err
				return false
			}
			result, _ = base.MergeSourceColumnSet(result, mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()))
			return false
		default:
			return true
		}
	})
	return result, walkErr
}

func (q *omniQuerySpanExtractor) getOmniFieldColumnSource(schemaName, tableName, columnName string) base.SourceColumnSet {
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if schemaName != "" && schemaName != tableSource.GetDatabaseName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}
		for _, field := range tableSource.GetQuerySpanResult() {
			if field.Name == columnName {
				return field.SourceColumns, true
			}
		}
		return nil, false
	}

	for _, tableSource := range q.tableSourcesFrom {
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet
		}
	}
	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.outerTableSources[i]); ok {
			return sourceColumnSet
		}
	}
	return base.SourceColumnSet{}
}

func (q *omniQuerySpanExtractor) expandOmniAsterisk(schemaName, tableName string) ([]base.QuerySpanResult, error) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if schemaName != "" && schemaName != tableSource.GetDatabaseName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}
		return tableSource.GetQuerySpanResult(), true
	}

	if tableName == "" && schemaName == "" {
		if len(q.topLevelTableSourcesFrom) > 0 {
			return cloneQuerySpanResultsFromTableSources(q.topLevelTableSourcesFrom), nil
		}
		if len(q.tableSourcesFrom) > 0 {
			return cloneQuerySpanResultsFromTableSources(q.tableSourcesFrom), nil
		}
		if len(q.outerTableSources) > 0 {
			return cloneQuerySpanResults(q.outerTableSources[len(q.outerTableSources)-1].GetQuerySpanResult()), nil
		}
		return []base.QuerySpanResult{}, nil
	}

	for i := len(q.tableSourcesFrom) - 1; i >= 0; i-- {
		if results, ok := findInTableSource(q.tableSourcesFrom[i]); ok {
			return cloneQuerySpanResults(results), nil
		}
	}
	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if results, ok := findInTableSource(q.outerTableSources[i]); ok {
			return cloneQuerySpanResults(results), nil
		}
	}
	return nil, errors.Errorf("failed to resolve asterisk for table %q", tableName)
}

func cloneQuerySpanResultsFromTableSources(tableSources []base.TableSource) []base.QuerySpanResult {
	var result []base.QuerySpanResult
	for _, tableSource := range tableSources {
		if tableSource == nil {
			continue
		}
		result = append(result, cloneQuerySpanResults(tableSource.GetQuerySpanResult())...)
	}
	return result
}

func (q *omniQuerySpanExtractor) extractOmniXMLTable(ref *oracleast.XmlTableRef) (base.TableSource, error) {
	_, sourceColumns, err := q.extractOmniExpr(ref.Passing)
	if err != nil {
		return nil, err
	}
	var columns []base.QuerySpanResult
	for _, node := range listItems(ref.Columns) {
		column, ok := node.(*oracleast.XmlTableColumn)
		if !ok {
			continue
		}
		columnSource := base.SourceColumnSet{}
		if !column.ForOrdinality {
			columnSource = sourceColumns
		}
		columns = append(columns, base.QuerySpanResult{
			Name:          column.Name,
			SourceColumns: columnSource,
			IsPlainField:  false,
		})
	}
	name := ""
	if ref.Alias != nil {
		name = ref.Alias.Name
	}
	return &base.PseudoTable{Name: name, Columns: columns}, nil
}

func (q *omniQuerySpanExtractor) extractOmniJSONTable(ref *oracleast.JsonTableRef) (base.TableSource, error) {
	_, sourceColumns, err := q.extractOmniExpr(ref.Expr)
	if err != nil {
		return nil, err
	}
	columns := extractOmniJSONTableColumns(ref.Columns, sourceColumns)
	name := ""
	if ref.Alias != nil {
		name = ref.Alias.Name
	}
	return &base.PseudoTable{Name: name, Columns: columns}, nil
}

func extractOmniJSONTableColumns(list *oracleast.List, sourceColumns base.SourceColumnSet) []base.QuerySpanResult {
	var columns []base.QuerySpanResult
	for _, node := range listItems(list) {
		column, ok := node.(*oracleast.JsonTableColumn)
		if !ok {
			continue
		}
		if column.Nested != nil {
			columns = append(columns, extractOmniJSONTableColumns(column.Nested.Columns, sourceColumns)...)
			continue
		}
		columnSource := sourceColumns
		if column.ForOrdinality {
			columnSource = base.SourceColumnSet{}
		}
		columns = append(columns, base.QuerySpanResult{
			Name:          column.Name,
			SourceColumns: columnSource,
			IsPlainField:  false,
		})
	}
	return columns
}

func extractOmniInlineExternalTable(ref *oracleast.InlineExternalTable) base.TableSource {
	var columns []base.QuerySpanResult
	for _, node := range listItems(ref.Columns) {
		column, ok := node.(*oracleast.ColumnDef)
		if !ok {
			continue
		}
		columns = append(columns, base.QuerySpanResult{
			Name:          column.Name,
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  false,
		})
	}
	name := ""
	if ref.Alias != nil {
		name = ref.Alias.Name
	}
	return &base.PseudoTable{Name: name, Columns: columns}
}

func aliasOmniTableSource(tableSource base.TableSource, alias *oracleast.Alias) base.TableSource {
	if tableSource == nil || alias == nil || alias.Name == "" {
		return tableSource
	}
	columns := cloneQuerySpanResults(tableSource.GetQuerySpanResult())
	applyOmniColumnAliases(columns, omniStringList(alias.Cols))
	return &base.PseudoTable{
		Name:    alias.Name,
		Columns: columns,
	}
}

func applyOmniColumnAliases(columns []base.QuerySpanResult, names []string) {
	for i, name := range names {
		if i >= len(columns) {
			return
		}
		columns[i].Name = name
	}
}

func cloneQuerySpanResult(result base.QuerySpanResult) base.QuerySpanResult {
	return base.QuerySpanResult{
		Name:          result.Name,
		SourceColumns: cloneSourceColumnSet(result.SourceColumns),
		IsPlainField:  result.IsPlainField,
	}
}

func cloneQuerySpanResults(results []base.QuerySpanResult) []base.QuerySpanResult {
	cloned := make([]base.QuerySpanResult, 0, len(results))
	for _, result := range results {
		cloned = append(cloned, cloneQuerySpanResult(result))
	}
	return cloned
}

func cloneSourceColumnSet(source base.SourceColumnSet) base.SourceColumnSet {
	cloned := make(base.SourceColumnSet, len(source))
	for column := range source {
		cloned[column] = true
	}
	return cloned
}

func excludeSourceColumnNames(excluded map[string]bool, source base.SourceColumnSet) {
	for column := range source {
		if column.Column == "" {
			continue
		}
		excluded[column.Column] = true
		excluded[strings.ToUpper(column.Column)] = true
	}
}

func excludeOmniColumnRefNames(excluded map[string]bool, expr oracleast.ExprNode) {
	oracleast.Inspect(expr, func(node oracleast.Node) bool {
		columnRef, ok := node.(*oracleast.ColumnRef)
		if ok && columnRef.Schema == "" && columnRef.Table == "" && columnRef.Column != "*" {
			excluded[columnRef.Column] = true
			excluded[strings.ToUpper(columnRef.Column)] = true
		}
		return true
	})
}

func mergeSourceColumnsFromResults(results []base.QuerySpanResult) base.SourceColumnSet {
	merged := make(base.SourceColumnSet)
	for _, result := range results {
		merged, _ = base.MergeSourceColumnSet(merged, result.SourceColumns)
	}
	return merged
}

func querySpanResultsEqual(left, right []base.QuerySpanResult) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Name != right[i].Name || left[i].IsPlainField != right[i].IsPlainField {
			return false
		}
		if !sourceColumnSetEqual(left[i].SourceColumns, right[i].SourceColumns) {
			return false
		}
	}
	return true
}

func sourceColumnSetEqual(left, right base.SourceColumnSet) bool {
	if len(left) != len(right) {
		return false
	}
	for column := range left {
		if !right[column] {
			return false
		}
	}
	return true
}

func listItems(list *oracleast.List) []oracleast.Node {
	if list == nil {
		return nil
	}
	return list.Items
}

func omniExprList(list *oracleast.List) []oracleast.ExprNode {
	var result []oracleast.ExprNode
	for _, node := range listItems(list) {
		expr, ok := node.(oracleast.ExprNode)
		if ok {
			result = append(result, expr)
		}
	}
	return result
}

func omniStringList(list *oracleast.List) []string {
	var result []string
	for _, node := range listItems(list) {
		switch node := node.(type) {
		case *oracleast.String:
			result = append(result, node.Str)
		case *oracleast.ColumnRef:
			result = append(result, node.Column)
		default:
		}
	}
	return result
}

func pivotInItemName(q *omniQuerySpanExtractor, item *oracleast.PivotInItem) string {
	if item == nil {
		return ""
	}
	if item.Alias != "" {
		return item.Alias
	}
	var parts []string
	for _, expr := range omniExprList(item.Values) {
		name, _, err := q.extractOmniExpr(expr)
		if err != nil {
			continue
		}
		if name != "" {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, "_")
}

func pivotColumnName(inName, aggregateAlias string, _ int) string {
	if aggregateAlias == "" {
		return inName
	}
	if inName == "" {
		return aggregateAlias
	}
	return inName + "_" + aggregateAlias
}

func isOmniStar(expr oracleast.ExprNode) bool {
	_, ok := expr.(*oracleast.Star)
	return ok
}

func splitOmniDBLink(dbLink string) []string {
	if dbLink == "" {
		return nil
	}
	return strings.Split(dbLink, ".")
}

func cloneTableSourceSlice(sources []base.TableSource) []base.TableSource {
	cloned := make([]base.TableSource, len(sources))
	copy(cloned, sources)
	return cloned
}

func (q *omniQuerySpanExtractor) omniExprName(loc oracleast.Loc, fallback string) string {
	if !loc.IsUnknown() && loc.Start >= 0 && loc.End <= len(q.source) && loc.Start < loc.End {
		return removeWhitespace(q.source[loc.Start:loc.End])
	}
	return removeWhitespace(fallback)
}

func removeWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func getOmniNodeLoc(node oracleast.Node) oracleast.Loc {
	value := reflect.ValueOf(node)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return oracleast.NoLoc()
	}
	elem := value.Elem()
	field := elem.FieldByName("Loc")
	if !field.IsValid() || !field.CanInterface() {
		return oracleast.NoLoc()
	}
	loc, ok := field.Interface().(oracleast.Loc)
	if !ok {
		return oracleast.NoLoc()
	}
	return loc
}

func getOmniExprFullLoc(expr oracleast.ExprNode) oracleast.Loc {
	switch expr := expr.(type) {
	case *oracleast.BinaryExpr:
		return mergeOmniLoc(getOmniExprFullLoc(expr.Left), getOmniExprFullLoc(expr.Right), getOmniNodeLoc(expr))
	case *oracleast.UnaryExpr:
		return mergeOmniLoc(getOmniNodeLoc(expr), getOmniExprFullLoc(expr.Operand))
	case *oracleast.BoolExpr:
		return mergeOmniListLoc(expr.Args, getOmniNodeLoc(expr))
	case *oracleast.FuncCallExpr:
		return getOmniNodeLoc(expr)
	case *oracleast.CaseExpr:
		return mergeOmniLoc(getOmniNodeLoc(expr), getOmniListLastLoc(expr.Whens), getOmniExprFullLoc(expr.Default))
	case *oracleast.CaseWhen:
		return mergeOmniLoc(getOmniNodeLoc(expr), getOmniExprFullLoc(expr.Condition), getOmniExprFullLoc(expr.Result))
	case *oracleast.DecodeExpr:
		return mergeOmniLoc(getOmniNodeLoc(expr), getOmniListLastLoc(expr.Pairs), getOmniExprFullLoc(expr.Default))
	case *oracleast.DecodePair:
		return mergeOmniLoc(getOmniExprFullLoc(expr.Search), getOmniExprFullLoc(expr.Result), getOmniNodeLoc(expr))
	case *oracleast.BetweenExpr:
		return mergeOmniLoc(getOmniExprFullLoc(expr.Expr), getOmniExprFullLoc(expr.Low), getOmniExprFullLoc(expr.High), getOmniNodeLoc(expr))
	case *oracleast.InExpr:
		return mergeOmniLoc(getOmniExprFullLoc(expr.Expr), getOmniNodeLoc(expr))
	case *oracleast.LikeExpr:
		return mergeOmniLoc(getOmniExprFullLoc(expr.Expr), getOmniExprFullLoc(expr.Pattern), getOmniExprFullLoc(expr.Escape), getOmniNodeLoc(expr))
	case *oracleast.IsExpr:
		return mergeOmniLoc(getOmniExprFullLoc(expr.Expr), getOmniNodeLoc(expr))
	case *oracleast.CastExpr:
		return mergeOmniLoc(getOmniNodeLoc(expr), getOmniExprFullLoc(expr.Arg))
	case *oracleast.MultisetExpr:
		return mergeOmniLoc(getOmniExprFullLoc(expr.Left), getOmniExprFullLoc(expr.Right), getOmniNodeLoc(expr))
	case *oracleast.CursorExpr:
		return getOmniNodeLoc(expr)
	case *oracleast.TreatExpr:
		return mergeOmniLoc(getOmniNodeLoc(expr), getOmniExprFullLoc(expr.Expr))
	case *oracleast.ParenExpr:
		return mergeOmniLoc(getOmniNodeLoc(expr), getOmniExprFullLoc(expr.Expr))
	default:
		return getOmniNodeLoc(expr)
	}
}

func mergeOmniListLoc(list *oracleast.List, fallback oracleast.Loc) oracleast.Loc {
	loc := fallback
	for _, node := range listItems(list) {
		loc = mergeOmniLoc(loc, getOmniNodeLoc(node))
	}
	return loc
}

func getOmniListLastLoc(list *oracleast.List) oracleast.Loc {
	var loc oracleast.Loc
	for _, node := range listItems(list) {
		loc = mergeOmniLoc(loc, getOmniNodeLoc(node))
	}
	return loc
}

func mergeOmniLoc(locs ...oracleast.Loc) oracleast.Loc {
	result := oracleast.NoLoc()
	for _, loc := range locs {
		if loc.IsUnknown() || loc.Start < 0 || loc.End < loc.Start {
			continue
		}
		if result.IsUnknown() || loc.Start < result.Start {
			result.Start = loc.Start
		}
		if result.IsUnknown() || loc.End > result.End {
			result.End = loc.End
		}
	}
	return result
}
