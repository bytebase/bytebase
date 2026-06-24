package cassandra

import (
	"context"

	"github.com/bytebase/omni/cassandra/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type querySpanExtractor struct {
	ctx             context.Context
	defaultKeyspace string
	gCtx            base.GetQuerySpanContext
	querySpan       *base.QuerySpan
}

func newQuerySpanExtractor(ctx context.Context, defaultKeyspace string, gCtx base.GetQuerySpanContext) *querySpanExtractor {
	return &querySpanExtractor{
		ctx:             ctx,
		defaultKeyspace: defaultKeyspace,
		gCtx:            gCtx,
		querySpan: &base.QuerySpan{
			Type:             base.QueryTypeUnknown,
			Results:          []base.QuerySpanResult{},
			SourceColumns:    base.SourceColumnSet{},
			PredicateColumns: base.SourceColumnSet{},
		},
	}
}

func (e *querySpanExtractor) extract(node ast.Node) *base.QuerySpan {
	switch stmt := node.(type) {
	case *ast.SelectStmt:
		e.extractSelect(stmt)
	case *ast.InsertStmt:
		e.extractInsert(stmt)
	case *ast.UpdateStmt:
		e.extractUpdate(stmt)
	case *ast.DeleteStmt:
		e.extractDelete(stmt)
	case *ast.BatchStmt:
		e.querySpan.Type = base.DML
		for _, child := range stmt.Statements {
			e.extract(child)
		}
	case *ast.TruncateStmt:
		e.querySpan.Type = base.DML
	case *ast.CreateTableStmt, *ast.AlterTableStmt, *ast.DropTableStmt,
		*ast.CreateKeyspaceStmt, *ast.AlterKeyspaceStmt, *ast.DropKeyspaceStmt,
		*ast.CreateIndexStmt, *ast.DropIndexStmt,
		*ast.CreateMVStmt, *ast.AlterMVStmt, *ast.DropMVStmt,
		*ast.CreateTypeStmt, *ast.AlterTypeStmt, *ast.DropTypeStmt,
		*ast.CreateFunctionStmt, *ast.DropFunctionStmt,
		*ast.CreateAggregateStmt, *ast.DropAggregateStmt,
		*ast.CreateTriggerStmt, *ast.DropTriggerStmt:
		e.querySpan.Type = base.DDL
	default:
	}
	return e.querySpan
}

func (e *querySpanExtractor) extractSelect(stmt *ast.SelectStmt) {
	e.querySpan.Type = base.Select

	keyspace, table := e.qualifiedNameParts(stmt.From)

	if table != "" {
		e.querySpan.SourceColumns[base.ColumnResource{
			Database: keyspace,
			Table:    table,
		}] = true
	}

	e.querySpan.Results = e.selectElements(stmt.Elements, keyspace, table)

	e.extractWhereColumns(stmt.Where, keyspace, table)
}

func (e *querySpanExtractor) extractInsert(stmt *ast.InsertStmt) {
	e.querySpan.Type = base.DML
	keyspace, table := e.qualifiedNameParts(stmt.Table)
	if table != "" {
		e.querySpan.SourceColumns[base.ColumnResource{
			Database: keyspace,
			Table:    table,
		}] = true
	}
}

func (e *querySpanExtractor) extractUpdate(stmt *ast.UpdateStmt) {
	e.querySpan.Type = base.DML
	keyspace, table := e.qualifiedNameParts(stmt.Table)
	if table != "" {
		e.querySpan.SourceColumns[base.ColumnResource{
			Database: keyspace,
			Table:    table,
		}] = true
	}
	e.extractWhereColumns(stmt.Where, keyspace, table)
}

func (e *querySpanExtractor) extractDelete(stmt *ast.DeleteStmt) {
	e.querySpan.Type = base.DML
	keyspace, table := e.qualifiedNameParts(stmt.From)
	if table != "" {
		e.querySpan.SourceColumns[base.ColumnResource{
			Database: keyspace,
			Table:    table,
		}] = true
	}
	e.extractWhereColumns(stmt.Where, keyspace, table)
}

func (e *querySpanExtractor) qualifiedNameParts(qn *ast.QualifiedName) (keyspace, table string) {
	if qn == nil {
		return e.defaultKeyspace, ""
	}
	switch len(qn.Parts) {
	case 2:
		return qn.Parts[0].Name, qn.Parts[1].Name
	case 1:
		return e.defaultKeyspace, qn.Parts[0].Name
	default:
		return e.defaultKeyspace, ""
	}
}

func (e *querySpanExtractor) selectElements(elems []*ast.SelectElement, keyspace, table string) []base.QuerySpanResult {
	if len(elems) == 0 {
		return nil
	}

	var results []base.QuerySpanResult
	for _, elem := range elems {
		if _, isStar := elem.Expr.(*ast.StarExpr); isStar {
			return e.expandSelectAsterisk(keyspace, table)
		}

		sourceName := exprColumnName(elem.Expr)
		resultName := sourceName
		if elem.Alias != nil {
			resultName = elem.Alias.Name
		}
		if resultName == "" && sourceName == "" {
			results = append(results, base.QuerySpanResult{
				Name:          "",
				SourceColumns: base.SourceColumnSet{},
			})
			continue
		}

		sc := base.SourceColumnSet{}
		sc[base.ColumnResource{
			Database: keyspace,
			Table:    table,
			Column:   sourceName,
		}] = true

		results = append(results, base.QuerySpanResult{
			Name:          resultName,
			SourceColumns: sc,
			IsPlainField:  true,
		})
	}
	return results
}

func exprColumnName(expr ast.ExprNode) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Name
	case *ast.DotAccess:
		if e.Field != nil {
			return e.Field.Name
		}
	default:
	}
	return ""
}

func (e *querySpanExtractor) expandSelectAsterisk(keyspace, table string) []base.QuerySpanResult {
	if e.gCtx.GetDatabaseMetadataFunc == nil || table == "" {
		return []base.QuerySpanResult{{
			SourceColumns:  base.SourceColumnSet{},
			SelectAsterisk: true,
		}}
	}

	_, metadata, err := e.gCtx.GetDatabaseMetadataFunc(e.ctx, e.gCtx.InstanceID, keyspace)
	if err != nil || metadata == nil {
		return []base.QuerySpanResult{{
			SourceColumns:  base.SourceColumnSet{},
			SelectAsterisk: true,
		}}
	}

	var results []base.QuerySpanResult
	for _, schemaName := range metadata.ListSchemaNames() {
		schema := metadata.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		tbl := schema.GetTable(table)
		if tbl == nil {
			continue
		}
		for _, col := range tbl.GetProto().GetColumns() {
			sc := base.SourceColumnSet{}
			sc[base.ColumnResource{
				Database: keyspace,
				Table:    table,
				Column:   col.GetName(),
			}] = true
			results = append(results, base.QuerySpanResult{
				Name:          col.GetName(),
				SourceColumns: sc,
				IsPlainField:  true,
			})
		}
		return results
	}

	return []base.QuerySpanResult{{
		SourceColumns:  base.SourceColumnSet{},
		SelectAsterisk: true,
	}}
}

func (e *querySpanExtractor) extractWhereColumns(where []ast.ExprNode, keyspace, table string) {
	for _, expr := range where {
		e.extractColumnRefsFromExpr(expr, keyspace, table)
	}
}

func (e *querySpanExtractor) extractColumnRefsFromExpr(expr ast.ExprNode, keyspace, table string) {
	switch ex := expr.(type) {
	case *ast.BinaryExpr:
		e.addPredicateColumnsFromExpr(ex.Left, keyspace, table)
	case *ast.InExpr:
		e.addPredicateColumnsFromExpr(ex.Column, keyspace, table)
	case *ast.ContainsExpr:
		e.addPredicateColumnsFromExpr(ex.Column, keyspace, table)
	case *ast.TupleInExpr:
		for _, col := range ex.Columns {
			e.addPredicateColumnsFromExpr(col, keyspace, table)
		}
	case *ast.TupleCompareExpr:
		for _, col := range ex.Columns {
			e.addPredicateColumnsFromExpr(col, keyspace, table)
		}
	default:
	}
}

func (e *querySpanExtractor) addPredicateColumnsFromExpr(expr ast.ExprNode, keyspace, table string) {
	for _, name := range collectColumnNames(expr) {
		e.querySpan.PredicateColumns[base.ColumnResource{
			Database: keyspace,
			Table:    table,
			Column:   name,
		}] = true
	}
}

func collectColumnNames(expr ast.ExprNode) []string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return []string{e.Name}
	case *ast.DotAccess:
		var names []string
		names = append(names, collectColumnNames(e.Object)...)
		if e.Field != nil {
			names = append(names, e.Field.Name)
		}
		return names
	case *ast.FunctionCall:
		var names []string
		for _, arg := range e.Args {
			names = append(names, collectColumnNames(arg)...)
		}
		return names
	default:
		return nil
	}
}
