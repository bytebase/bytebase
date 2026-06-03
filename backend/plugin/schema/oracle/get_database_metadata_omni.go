package oracle

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

// GetDatabaseMetadataOmni parses Oracle schema DDL text and returns database metadata
// using the omni parser AST.
func GetDatabaseMetadataOmni(schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	list, err := plsqlparser.ParsePLSQLOmni(schemaText)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse Oracle schema")
	}
	if list == nil || len(list.Items) == 0 {
		return nil, errors.New("no parse results")
	}

	extractor := &oracleOmniMetadataExtractor{
		tables:            make(map[string]*storepb.TableMetadata),
		views:             make(map[string]*storepb.ViewMetadata),
		materializedViews: make(map[string]*storepb.MaterializedViewMetadata),
		functions:         make(map[string]*storepb.FunctionMetadata),
		procedures:        make(map[string]*storepb.ProcedureMetadata),
		triggers:          make(map[string]*storepb.TriggerMetadata),
		sequences:         make(map[string]*storepb.SequenceMetadata),
		packages:          make(map[string]*storepb.PackageMetadata),
		checkNames:        make(map[string]map[string]bool),
		checkNameNext:     make(map[string]map[string]int),
		schemaText:        schemaText,
	}

	for _, item := range list.Items {
		raw, ok := item.(*ast.RawStmt)
		if !ok || raw.Stmt == nil {
			continue
		}
		extractor.extractStatement(raw.Stmt)
	}

	return extractor.databaseMetadata(), nil
}

type oracleOmniMetadataExtractor struct {
	currentDatabase   string
	currentSchema     string
	tables            map[string]*storepb.TableMetadata
	views             map[string]*storepb.ViewMetadata
	materializedViews map[string]*storepb.MaterializedViewMetadata
	functions         map[string]*storepb.FunctionMetadata
	procedures        map[string]*storepb.ProcedureMetadata
	triggers          map[string]*storepb.TriggerMetadata
	sequences         map[string]*storepb.SequenceMetadata
	packages          map[string]*storepb.PackageMetadata
	checkNames        map[string]map[string]bool
	checkNameNext     map[string]map[string]int
	schemaText        string
}

func (e *oracleOmniMetadataExtractor) databaseMetadata() *storepb.DatabaseSchemaMetadata {
	e.resolveForeignKeyReferencedColumns()

	schemaMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    e.currentDatabase,
		Schemas: []*storepb.SchemaMetadata{},
	}
	schema := &storepb.SchemaMetadata{
		Name:              e.currentSchema,
		Tables:            []*storepb.TableMetadata{},
		Views:             []*storepb.ViewMetadata{},
		MaterializedViews: []*storepb.MaterializedViewMetadata{},
		Procedures:        []*storepb.ProcedureMetadata{},
		Functions:         []*storepb.FunctionMetadata{},
		Sequences:         []*storepb.SequenceMetadata{},
		Packages:          []*storepb.PackageMetadata{},
	}

	var tableNames []string
	for name := range e.tables {
		if _, ok := e.materializedViews[name]; !ok {
			tableNames = append(tableNames, name)
		}
	}
	slices.Sort(tableNames)
	for _, name := range tableNames {
		schema.Tables = append(schema.Tables, e.tables[name])
	}

	var viewNames []string
	for name := range e.views {
		viewNames = append(viewNames, name)
	}
	slices.Sort(viewNames)
	for _, name := range viewNames {
		schema.Views = append(schema.Views, e.views[name])
	}

	var materializedViewNames []string
	for name := range e.materializedViews {
		materializedViewNames = append(materializedViewNames, name)
	}
	slices.Sort(materializedViewNames)
	for _, name := range materializedViewNames {
		schema.MaterializedViews = append(schema.MaterializedViews, e.materializedViews[name])
	}

	var functionNames []string
	for name := range e.functions {
		functionNames = append(functionNames, name)
	}
	slices.Sort(functionNames)
	for _, name := range functionNames {
		schema.Functions = append(schema.Functions, e.functions[name])
	}

	var procedureNames []string
	for name := range e.procedures {
		procedureNames = append(procedureNames, name)
	}
	slices.Sort(procedureNames)
	for _, name := range procedureNames {
		schema.Procedures = append(schema.Procedures, e.procedures[name])
	}

	var sequenceNames []string
	for name := range e.sequences {
		sequenceNames = append(sequenceNames, name)
	}
	slices.Sort(sequenceNames)
	for _, name := range sequenceNames {
		schema.Sequences = append(schema.Sequences, e.sequences[name])
	}

	var packageNames []string
	for name := range e.packages {
		packageNames = append(packageNames, name)
	}
	slices.Sort(packageNames)
	for _, name := range packageNames {
		schema.Packages = append(schema.Packages, e.packages[name])
	}

	schemaMetadata.Schemas = append(schemaMetadata.Schemas, schema)
	return schemaMetadata
}

func (e *oracleOmniMetadataExtractor) getOrCreateTable(tableName string) *storepb.TableMetadata {
	if table, ok := e.tables[tableName]; ok {
		return table
	}
	table := &storepb.TableMetadata{
		Name:             tableName,
		Columns:          []*storepb.ColumnMetadata{},
		Indexes:          []*storepb.IndexMetadata{},
		ForeignKeys:      []*storepb.ForeignKeyMetadata{},
		CheckConstraints: []*storepb.CheckConstraintMetadata{},
		Triggers:         []*storepb.TriggerMetadata{},
		Partitions:       []*storepb.TablePartitionMetadata{},
	}
	e.tables[tableName] = table
	return table
}

func (e *oracleOmniMetadataExtractor) extractStatement(stmt ast.StmtNode) {
	switch n := stmt.(type) {
	case *ast.CreateTableStmt:
		e.extractCreateTable(n)
	case *ast.CreateIndexStmt:
		e.extractCreateIndex(n)
	case *ast.CreateViewStmt:
		e.extractCreateView(n)
	case *ast.CreateSchemaStmt:
		e.extractCreateSchema(n)
	case *ast.CreateSequenceStmt:
		e.extractCreateSequence(n)
	case *ast.CreateProcedureStmt:
		if name := objectName(n.Name); name != "" {
			e.procedures[name] = &storepb.ProcedureMetadata{Name: name, Definition: e.definitionText(n)}
		}
	case *ast.CreateFunctionStmt:
		if name := objectName(n.Name); name != "" {
			e.functions[name] = &storepb.FunctionMetadata{Name: name, Definition: e.functionDefinitionText(n)}
		}
	case *ast.CreatePackageStmt:
		if name := objectName(n.Name); name != "" {
			e.extractCreatePackage(name, n)
		}
	case *ast.CreateTriggerStmt:
		e.extractCreateTrigger(n)
	case *ast.AlterTableStmt:
		e.extractAlterTable(n)
	case *ast.CommentStmt:
		e.extractComment(n)
	default:
	}
}

func (e *oracleOmniMetadataExtractor) extractCreateSchema(n *ast.CreateSchemaStmt) {
	if n.SchemaName != "" {
		e.currentSchema = n.SchemaName
	}
	for _, item := range listItems(n.Stmts) {
		stmt, ok := item.(ast.StmtNode)
		if !ok {
			continue
		}
		e.extractStatement(stmt)
	}
}

func (e *oracleOmniMetadataExtractor) extractCreateTable(n *ast.CreateTableStmt) {
	tableName := objectName(n.Name)
	if tableName == "" {
		return
	}
	if n.Name.Schema != "" {
		e.currentSchema = n.Name.Schema
	}

	table := e.getOrCreateTable(tableName)
	for _, item := range listItems(n.Columns) {
		column, ok := item.(*ast.ColumnDef)
		if !ok {
			continue
		}
		e.extractColumn(column, table)
	}
	for _, item := range listItems(n.Constraints) {
		constraint, ok := item.(*ast.TableConstraint)
		if !ok {
			continue
		}
		e.extractTableConstraint(constraint, table)
	}
}

func (e *oracleOmniMetadataExtractor) extractColumn(n *ast.ColumnDef, table *storepb.TableMetadata) {
	column := &storepb.ColumnMetadata{
		Name:     n.Name,
		Type:     "VARCHAR2(100)",
		Nullable: true,
		Position: int32(len(table.Columns) + 1),
	}
	if n.TypeName != nil {
		column.Type = normalizeDataTypeText(e.nodeText(n.TypeName))
	} else if n.Domain != nil {
		column.Type = objectName(n.Domain)
	}
	if n.Default != nil {
		column.Default = e.nodeText(n.Default)
	}
	if n.Virtual != nil {
		if n.TypeName == nil && n.Domain == nil {
			column.Type = "NUMBER"
		}
		column.Default = e.exprText(n.Virtual)
	}
	column.DefaultOnNull = n.DefaultOnNull
	if n.Collation != "" {
		column.Collation = n.Collation
	}
	if n.Identity != nil {
		column.IsIdentity = true
		column.Nullable = false
	}
	if n.NotNull {
		column.Nullable = false
	}
	if n.Null {
		column.Nullable = true
	}

	for _, item := range listItems(n.Constraints) {
		constraint, ok := item.(*ast.ColumnConstraint)
		if !ok {
			continue
		}
		e.extractColumnConstraint(constraint, table, column)
	}

	table.Columns = append(table.Columns, column)
}

func (e *oracleOmniMetadataExtractor) extractColumnConstraint(n *ast.ColumnConstraint, table *storepb.TableMetadata, column *storepb.ColumnMetadata) {
	switch n.Type {
	case ast.CONSTRAINT_NOT_NULL:
		column.Nullable = false
	case ast.CONSTRAINT_NULL:
		column.Nullable = true
	case ast.CONSTRAINT_DEFAULT:
		if n.Expr != nil {
			column.Default = e.nodeText(n.Expr)
		}
	case ast.CONSTRAINT_PRIMARY:
		column.Nullable = false
		if !hasPrimaryIndex(table) {
			table.Indexes = append(table.Indexes, &storepb.IndexMetadata{
				Name:         fallbackName(n.Name, fmt.Sprintf("PK_%s", table.Name)),
				Primary:      true,
				Unique:       true,
				Type:         "NORMAL",
				Expressions:  []string{column.Name},
				Visible:      true,
				IsConstraint: true,
			})
		}
	case ast.CONSTRAINT_UNIQUE:
		name := fallbackName(n.Name, fmt.Sprintf("UK_%s_%s", table.Name, column.Name))
		if !hasUniqueIndex(table, []string{column.Name}) {
			table.Indexes = append(table.Indexes, &storepb.IndexMetadata{
				Name:         name,
				Unique:       true,
				Type:         "NORMAL",
				Expressions:  []string{column.Name},
				Visible:      true,
				IsConstraint: true,
			})
		}
	case ast.CONSTRAINT_CHECK:
		e.appendCheckConstraint(table, e.checkConstraintName(table, n.Name, fmt.Sprintf("CHK_%s_%s", table.Name, column.Name)), n.Expr)
	case ast.CONSTRAINT_FOREIGN:
		e.appendForeignKey(table, fallbackName(n.Name, fmt.Sprintf("FK_%s_%s", table.Name, column.Name)), []string{column.Name}, n.RefTable, n.RefColumns, n.OnDelete)
	default:
	}
}

func (e *oracleOmniMetadataExtractor) extractTableConstraint(n *ast.TableConstraint, table *storepb.TableMetadata) {
	columns := stringList(n.Columns)
	switch n.Type {
	case ast.CONSTRAINT_PRIMARY:
		if len(columns) == 0 {
			return
		}
		table.Indexes = append(table.Indexes, &storepb.IndexMetadata{
			Name:         fallbackName(n.Name, fmt.Sprintf("PK_%s", table.Name)),
			Primary:      true,
			Unique:       true,
			Type:         "NORMAL",
			Expressions:  columns,
			Visible:      true,
			IsConstraint: true,
		})
		markColumnsNotNull(table, columns)
	case ast.CONSTRAINT_UNIQUE:
		if len(columns) == 0 {
			return
		}
		table.Indexes = append(table.Indexes, &storepb.IndexMetadata{
			Name:         fallbackName(n.Name, fmt.Sprintf("UK_%s_%d", table.Name, len(table.Indexes)+1)),
			Unique:       true,
			Type:         "NORMAL",
			Expressions:  columns,
			Visible:      false,
			IsConstraint: true,
		})
	case ast.CONSTRAINT_CHECK:
		e.appendCheckConstraint(table, e.checkConstraintName(table, n.Name, fmt.Sprintf("CHK_%s_%d", table.Name, len(table.CheckConstraints)+1)), n.Expr)
	case ast.CONSTRAINT_FOREIGN:
		e.appendForeignKey(table, fallbackName(n.Name, fmt.Sprintf("FK_%s_%d", table.Name, len(table.ForeignKeys)+1)), columns, n.RefTable, n.RefColumns, n.OnDelete)
	default:
	}
}

func (e *oracleOmniMetadataExtractor) extractAlterTable(n *ast.AlterTableStmt) {
	tableName := objectName(n.Name)
	if tableName == "" {
		return
	}
	table := e.getOrCreateTable(tableName)
	for _, item := range listItems(n.Actions) {
		action, ok := item.(*ast.AlterTableCmd)
		if !ok {
			continue
		}
		switch action.Action {
		case ast.AT_ADD_COLUMN:
			if action.ColumnDef != nil {
				e.extractColumn(action.ColumnDef, table)
			}
			for _, item := range listItems(action.ColumnDefs) {
				column, ok := item.(*ast.ColumnDef)
				if !ok {
					continue
				}
				e.extractColumn(column, table)
			}
		case ast.AT_ADD_CONSTRAINT:
			if action.Constraint != nil {
				e.extractTableConstraint(action.Constraint, table)
			}
		default:
		}
	}
}

func (e *oracleOmniMetadataExtractor) appendCheckConstraint(table *storepb.TableMetadata, name string, expr ast.ExprNode) {
	if expr == nil {
		return
	}
	e.reserveCheckConstraintName(table, name)
	table.CheckConstraints = append(table.CheckConstraints, &storepb.CheckConstraintMetadata{
		Name:       name,
		Expression: e.exprText(expr),
	})
}

func (e *oracleOmniMetadataExtractor) checkConstraintName(table *storepb.TableMetadata, name string, fallback string) string {
	if name != "" {
		return name
	}
	return e.uniqueCheckConstraintName(table, fallback)
}

func (e *oracleOmniMetadataExtractor) uniqueCheckConstraintName(table *storepb.TableMetadata, fallback string) string {
	used := e.checkConstraintNameSet(table)
	if !used[fallback] {
		return fallback
	}
	nextByFallback := e.checkConstraintNameNextSet(table)
	next := nextByFallback[fallback]
	if next < 2 {
		next = 2
	}
	for {
		candidate := fmt.Sprintf("%s_%d", fallback, next)
		next++
		if !used[candidate] {
			nextByFallback[fallback] = next
			return candidate
		}
	}
}

func (e *oracleOmniMetadataExtractor) reserveCheckConstraintName(table *storepb.TableMetadata, name string) {
	e.checkConstraintNameSet(table)[name] = true
}

func (e *oracleOmniMetadataExtractor) checkConstraintNameSet(table *storepb.TableMetadata) map[string]bool {
	used := e.checkNames[table.Name]
	if used != nil {
		return used
	}
	used = make(map[string]bool)
	for _, check := range table.CheckConstraints {
		used[check.Name] = true
	}
	e.checkNames[table.Name] = used
	return used
}

func (e *oracleOmniMetadataExtractor) checkConstraintNameNextSet(table *storepb.TableMetadata) map[string]int {
	next := e.checkNameNext[table.Name]
	if next != nil {
		return next
	}
	next = make(map[string]int)
	e.checkNameNext[table.Name] = next
	return next
}

func (*oracleOmniMetadataExtractor) appendForeignKey(table *storepb.TableMetadata, name string, columns []string, refTable *ast.ObjectName, refColumns *ast.List, onDelete string) {
	referencedTable := objectName(refTable)
	referencedColumns := stringList(refColumns)
	if len(columns) == 0 || referencedTable == "" {
		return
	}
	foreignKey := &storepb.ForeignKeyMetadata{
		Name:              name,
		Columns:           columns,
		ReferencedTable:   referencedTable,
		ReferencedColumns: referencedColumns,
		OnDelete:          onDelete,
	}
	table.ForeignKeys = append(table.ForeignKeys, foreignKey)
}

func (e *oracleOmniMetadataExtractor) resolveForeignKeyReferencedColumns() {
	for _, table := range e.tables {
		var foreignKeys []*storepb.ForeignKeyMetadata
		for _, foreignKey := range table.ForeignKeys {
			if len(foreignKey.ReferencedColumns) == 0 {
				referencedTable := e.tables[foreignKey.ReferencedTable]
				foreignKey.ReferencedColumns = primaryKeyColumns(referencedTable)
			}
			if len(foreignKey.ReferencedColumns) == 0 {
				continue
			}
			foreignKeys = append(foreignKeys, foreignKey)
		}
		table.ForeignKeys = foreignKeys
	}
}

func (e *oracleOmniMetadataExtractor) extractCreateIndex(n *ast.CreateIndexStmt) {
	tableName := objectName(n.Table)
	indexName := objectName(n.Name)
	if tableName == "" || indexName == "" {
		return
	}
	if n.Name.Schema != "" {
		e.currentSchema = n.Name.Schema
	}

	index := &storepb.IndexMetadata{
		Name:        indexName,
		Unique:      n.Unique,
		Type:        "NORMAL",
		Expressions: []string{},
		Visible:     !n.Invisible,
	}
	if n.Bitmap {
		index.Type = "BITMAP"
	}

	isFunctionBased := false
	for _, item := range listItems(n.Columns) {
		column, ok := item.(*ast.IndexColumn)
		if !ok {
			continue
		}
		expression := e.indexExpression(column.Expr)
		if expression == "" {
			continue
		}
		if _, ok := column.Expr.(*ast.ColumnRef); !ok {
			isFunctionBased = true
		}
		index.Expressions = append(index.Expressions, expression)
		index.Descending = append(index.Descending, column.Dir == ast.SORTBY_DESC)
	}
	for _, desc := range index.Descending {
		if desc && index.Type == "NORMAL" {
			index.Type = "FUNCTION-BASED NORMAL"
			break
		}
	}
	if isFunctionBased && index.Type == "NORMAL" {
		index.Type = "FUNCTION-BASED NORMAL"
	} else if isFunctionBased && index.Type == "BITMAP" {
		index.Type = "FUNCTION-BASED BITMAP"
	}

	if materializedView := e.materializedViews[tableName]; materializedView != nil {
		materializedView.Indexes = append(materializedView.Indexes, index)
		return
	}
	table := e.getOrCreateTable(tableName)
	table.Indexes = append(table.Indexes, index)
}

func (e *oracleOmniMetadataExtractor) extractCreateView(n *ast.CreateViewStmt) {
	viewName := objectName(n.Name)
	if viewName == "" {
		return
	}
	if n.Name.Schema != "" {
		e.currentSchema = n.Name.Schema
	}

	definition := e.nodeText(n.Query)
	if n.Materialized {
		materializedView := &storepb.MaterializedViewMetadata{
			Name:       viewName,
			Definition: definition,
		}
		if table := e.tables[viewName]; table != nil {
			materializedView.Triggers = append(materializedView.Triggers, table.Triggers...)
			materializedView.Indexes = append(materializedView.Indexes, table.Indexes...)
		}
		if definition != "" && !strings.HasSuffix(definition, "\n") {
			materializedView.Definition += "\n"
		}
		e.materializedViews[viewName] = materializedView
		delete(e.tables, viewName)
		return
	}

	view := &storepb.ViewMetadata{
		Name:       viewName,
		Definition: definition,
	}
	if table := e.tables[viewName]; table != nil {
		view.Triggers = append(view.Triggers, table.Triggers...)
		delete(e.tables, viewName)
	}
	e.views[viewName] = view
}

func (e *oracleOmniMetadataExtractor) extractCreateSequence(n *ast.CreateSequenceStmt) {
	sequenceName := objectName(n.Name)
	if sequenceName == "" {
		return
	}
	sequence := &storepb.SequenceMetadata{Name: sequenceName}
	if start := e.nodeText(n.StartWith); start != "" {
		sequence.Start = start
	}
	if increment := e.nodeText(n.IncrementBy); increment != "" && increment != "1" {
		sequence.Increment = increment
	}
	e.sequences[sequenceName] = sequence
}

func (e *oracleOmniMetadataExtractor) extractCreatePackage(name string, n *ast.CreatePackageStmt) {
	definition := e.definitionText(n)
	if definition == "" {
		return
	}
	pkg := e.packages[name]
	if pkg == nil {
		e.packages[name] = &storepb.PackageMetadata{Name: name, Definition: definition}
		return
	}
	if pkg.Definition == "" {
		pkg.Definition = definition
		return
	}
	pkg.Definition += "\n\n" + definition
}

func (e *oracleOmniMetadataExtractor) extractCreateTrigger(n *ast.CreateTriggerStmt) {
	triggerName := objectName(n.Name)
	tableName := objectName(n.Table)
	if triggerName == "" || tableName == "" {
		return
	}
	trigger := &storepb.TriggerMetadata{
		Name: triggerName,
		Body: e.definitionText(n),
	}
	e.triggers[triggerName] = trigger
	if view := e.views[tableName]; view != nil {
		view.Triggers = append(view.Triggers, trigger)
		return
	}
	if materializedView := e.materializedViews[tableName]; materializedView != nil {
		materializedView.Triggers = append(materializedView.Triggers, trigger)
		return
	}
	table := e.getOrCreateTable(tableName)
	table.Triggers = append(table.Triggers, trigger)
}

func (e *oracleOmniMetadataExtractor) extractComment(n *ast.CommentStmt) {
	if n.Column != "" {
		table := e.tables[objectName(n.Object)]
		if table == nil {
			return
		}
		for _, column := range table.Columns {
			if column.Name == n.Column {
				column.Comment = n.Comment
				return
			}
		}
		return
	}

	switch n.ObjectType {
	case ast.OBJECT_TABLE, ast.OBJECT_VIEW:
		name := objectName(n.Object)
		if table := e.tables[name]; table != nil {
			table.Comment = n.Comment
			return
		}
		if view := e.views[name]; view != nil {
			view.Comment = n.Comment
		}
	case ast.OBJECT_MATERIALIZED_VIEW:
		if materializedView := e.materializedViews[objectName(n.Object)]; materializedView != nil {
			materializedView.Comment = n.Comment
		}
	default:
	}
}

func (e *oracleOmniMetadataExtractor) indexExpression(expr ast.ExprNode) string {
	switch n := expr.(type) {
	case *ast.ColumnRef:
		return n.Column
	default:
		return e.nodeText(expr)
	}
}

func (e *oracleOmniMetadataExtractor) exprText(expr ast.ExprNode) string {
	switch n := expr.(type) {
	case *ast.BinaryExpr:
		return e.exprText(n.Left) + n.Op + e.exprText(n.Right)
	case *ast.BoolExpr:
		var parts []string
		for _, item := range listItems(n.Args) {
			expr, ok := item.(ast.ExprNode)
			if !ok {
				continue
			}
			parts = append(parts, e.exprText(expr))
		}
		switch n.Boolop {
		case ast.BOOL_AND:
			return strings.Join(parts, "AND")
		case ast.BOOL_OR:
			return strings.Join(parts, "OR")
		case ast.BOOL_NOT:
			return "NOT" + strings.Join(parts, "")
		default:
			return strings.Join(parts, "")
		}
	case *ast.ColumnRef:
		if n.Table != "" {
			return n.Table + "." + n.Column
		}
		return n.Column
	case *ast.InExpr:
		operator := "IN"
		if n.Not {
			operator = "NOTIN"
		}
		return e.exprText(n.Expr) + operator + "(" + strings.Join(e.exprListText(n.List), ",") + ")"
	case *ast.BetweenExpr:
		operator := "BETWEEN"
		if n.Not {
			operator = "NOTBETWEEN"
		}
		return e.exprText(n.Expr) + operator + e.exprText(n.Low) + "AND" + e.exprText(n.High)
	case *ast.LikeExpr:
		operator := "LIKE"
		if n.Not {
			operator = "NOTLIKE"
		}
		result := e.exprText(n.Expr) + operator + e.exprText(n.Pattern)
		if n.Escape != nil {
			result += "ESCAPE" + e.exprText(n.Escape)
		}
		return result
	case *ast.IsExpr:
		operator := "IS"
		if n.Not {
			operator = "ISNOT"
		}
		return e.exprText(n.Expr) + operator + n.Test
	case *ast.ParenExpr:
		return "(" + e.exprText(n.Expr) + ")"
	case *ast.StringLiteral:
		prefix := ""
		if n.IsNChar {
			prefix = "N"
		}
		return prefix + "'" + strings.ReplaceAll(n.Val, "'", "''") + "'"
	case *ast.NumberLiteral:
		return n.Val
	case *ast.NullLiteral:
		return "NULL"
	case *ast.FuncCallExpr:
		return objectName(n.FuncName) + "(" + strings.Join(e.exprListText(n.Args), ",") + ")"
	default:
		return strings.ReplaceAll(e.nodeText(expr), " ", "")
	}
}

func (e *oracleOmniMetadataExtractor) exprListText(list *ast.List) []string {
	var result []string
	for _, item := range listItems(list) {
		expr, ok := item.(ast.ExprNode)
		if !ok {
			continue
		}
		result = append(result, e.exprText(expr))
	}
	return result
}

func (e *oracleOmniMetadataExtractor) functionDefinitionText(node ast.Node) string {
	definition := e.definitionText(node)
	upper := strings.ToUpper(definition)
	if strings.HasPrefix(upper, "CREATE OR REPLACE ") {
		definition = definition[len("CREATE OR REPLACE "):]
	} else if strings.HasPrefix(upper, "CREATE ") {
		definition = definition[len("CREATE "):]
	}
	if definition != "" && !strings.HasPrefix(strings.ToUpper(definition), "FUNCTION") {
		definition = "FUNCTION " + definition
	}
	return definition
}

func (e *oracleOmniMetadataExtractor) definitionText(node ast.Node) string {
	loc := ast.NodeLoc(node)
	if loc.Start < 0 || loc.End < 0 || loc.Start > loc.End || loc.End > len(e.schemaText) {
		return ""
	}
	end := loc.End
	for end < len(e.schemaText) {
		switch e.schemaText[end] {
		case ' ', '\t', '\r', '\n':
			end++
		default:
			if e.schemaText[end] == ';' {
				end++
			}
			return strings.TrimSpace(e.schemaText[loc.Start:end])
		}
	}
	return strings.TrimSpace(e.schemaText[loc.Start:end])
}

func (e *oracleOmniMetadataExtractor) nodeText(node ast.Node) string {
	loc := ast.NodeLoc(node)
	if loc.Start < 0 || loc.End < 0 || loc.Start > loc.End || loc.End > len(e.schemaText) {
		return ""
	}
	return e.schemaText[loc.Start:loc.End]
}

func objectName(name *ast.ObjectName) string {
	if name == nil {
		return ""
	}
	return name.Name
}

func listItems(list *ast.List) []ast.Node {
	if list == nil {
		return nil
	}
	return list.Items
}

func stringList(list *ast.List) []string {
	var result []string
	for _, item := range listItems(list) {
		switch n := item.(type) {
		case *ast.String:
			result = append(result, n.Str)
		case *ast.ColumnRef:
			result = append(result, n.Column)
		default:
		}
	}
	return result
}

func fallbackName(name, fallback string) string {
	if name != "" {
		return name
	}
	return fallback
}

func hasPrimaryIndex(table *storepb.TableMetadata) bool {
	for _, index := range table.Indexes {
		if index.Primary {
			return true
		}
	}
	return false
}

func primaryKeyColumns(table *storepb.TableMetadata) []string {
	if table == nil {
		return nil
	}
	for _, index := range table.Indexes {
		if index.Primary {
			return append([]string(nil), index.Expressions...)
		}
	}
	return nil
}

func hasUniqueIndex(table *storepb.TableMetadata, columns []string) bool {
	for _, index := range table.Indexes {
		if !index.Unique || index.Primary || len(index.Expressions) != len(columns) {
			continue
		}
		matches := true
		for i, column := range columns {
			if index.Expressions[i] != column {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func markColumnsNotNull(table *storepb.TableMetadata, columns []string) {
	for _, column := range table.Columns {
		if slices.Contains(columns, column.Name) {
			column.Nullable = false
		}
	}
}
