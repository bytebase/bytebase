package mysql

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_MYSQL, GetMaskedFields)
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_MARIADB, GetMaskedFields)
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_OCEANBASE, GetMaskedFields)
}

func GetMaskedFields(statement, currentDatabase string, schemaInfo *base.SensitiveSchemaInfo) ([]base.SensitiveField, error) {
	extractor := &fieldExtractor{
		currentDatabase: currentDatabase,
		schemaInfo:      schemaInfo,
	}
	result, err := extractor.extractSensitiveFields(statement)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type fieldExtractor struct {
	// For Oracle, we need to know the current database to determine if the table is in the current schema.
	currentDatabase    string
	schemaInfo         *base.SensitiveSchemaInfo
	outerSchemaInfo    []base.FieldInfo
	cteOuterSchemaInfo []base.TableSchema

	// SELECT statement specific field.
	fromFieldList []base.FieldInfo
}

func (extractor *fieldExtractor) extractSensitiveFields(statement string) ([]base.SensitiveField, error) {
	list, err := ParseMySQL(statement)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	if len(list) != 1 {
		return nil, errors.Errorf("MySQL statement should only have one statement, but got %d", len(list))
	}

	listener := &mysqlSensitiveFieldListener{
		extractor: extractor,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, list[0].Tree)
	return listener.result, listener.err
}

type mysqlSensitiveFieldListener struct {
	*mysql.BaseMySQLParserListener

	extractor *fieldExtractor
	result    []base.SensitiveField
	err       error
}

// EnterSelectStatement is called when production selectStatement is entered.
func (l *mysqlSensitiveFieldListener) EnterSelectStatement(ctx *mysql.SelectStatementContext) {
	parent := ctx.GetParent()
	if parent == nil {
		return
	}

	if _, ok := parent.(*mysql.SimpleStatementContext); !ok {
		return
	}

	fieldList, err := l.extractor.mysqlExtractContext(ctx)
	if err != nil {
		l.err = err
		return
	}

	for _, field := range fieldList {
		l.result = append(l.result, base.SensitiveField{
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}
}

// EnterCreateView is called when production createView is entered.
func (l *mysqlSensitiveFieldListener) EnterCreateView(ctx *mysql.CreateViewContext) {
	fieldList, err := l.extractor.mysqlExtractCreateView(ctx)
	if err != nil {
		l.err = err
		return
	}

	for _, field := range fieldList {
		l.result = append(l.result, base.SensitiveField{
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}

	if ctx.ViewTail().ColumnInternalRefList() != nil {
		columnList := mysqlExtractColumnInternalRefList(ctx.ViewTail().ColumnInternalRefList())
		if len(columnList) != len(l.result) {
			l.err = errors.Errorf("MySQL view column list should have the same length, but got %d and %d", len(columnList), len(l.result))
			return
		}
		for i := range l.result {
			l.result[i].Name = columnList[i]
		}
	}
}

func (extractor *fieldExtractor) mysqlExtractCreateView(ctx mysql.ICreateViewContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	return extractor.mysqlExtractQueryExpressionOrParens(ctx.ViewTail().ViewSelect().QueryExpressionOrParens())
}

func (extractor *fieldExtractor) mysqlExtractQueryExpressionOrParens(ctx mysql.IQueryExpressionOrParensContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.QueryExpression() != nil:
		return extractor.mysqlExtractQueryExpression(ctx.QueryExpression())
	case ctx.QueryExpressionParens() != nil:
		return extractor.mysqlExtractQueryExpressionParens(ctx.QueryExpressionParens())
	}

	return nil, nil
}

func (extractor *fieldExtractor) mysqlExtractContext(ctx antlr.ParserRuleContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch ctx := ctx.(type) {
	case mysql.ISelectStatementContext:
		return extractor.mysqlExtractSelectStatement(ctx)
	default:
		return nil, nil
	}
}

func (extractor *fieldExtractor) mysqlExtractSelectStatement(ctx mysql.ISelectStatementContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.QueryExpression() != nil:
		return extractor.mysqlExtractQueryExpression(ctx.QueryExpression())
	case ctx.QueryExpressionParens() != nil:
		return extractor.mysqlExtractQueryExpressionParens(ctx.QueryExpressionParens())
	}

	return nil, nil
}

func (extractor *fieldExtractor) mysqlExtractQueryExpressionParens(ctx mysql.IQueryExpressionParensContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.QueryExpression() != nil:
		return extractor.mysqlExtractQueryExpression(ctx.QueryExpression())
	case ctx.QueryExpressionParens() != nil:
		return extractor.mysqlExtractQueryExpressionParens(ctx.QueryExpressionParens())
	}

	return nil, nil
}

func (extractor *fieldExtractor) mysqlExtractQueryExpression(ctx mysql.IQueryExpressionContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.WithClause() != nil {
		cteOuterLength := len(extractor.cteOuterSchemaInfo)
		defer func() {
			extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:cteOuterLength]
		}()
		recursive := ctx.WithClause().RECURSIVE_SYMBOL() != nil
		for _, cte := range ctx.WithClause().AllCommonTableExpression() {
			cteTable, err := extractor.mysqlExtractCommonTableExpression(cte, recursive)
			if err != nil {
				return nil, err
			}
			extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, cteTable)
		}
	}

	switch {
	case ctx.QueryExpressionParens() != nil:
		return extractor.mysqlExtractQueryExpressionParens(ctx.QueryExpressionParens())
	case ctx.QueryExpressionBody() != nil:
		return extractor.mysqlExtractQueryExpressionBody(ctx.QueryExpressionBody())
	}

	return nil, nil
}

func (extractor *fieldExtractor) mysqlExtractCommonTableExpression(ctx mysql.ICommonTableExpressionContext, recursive bool) (base.TableSchema, error) {
	if ctx == nil {
		return base.TableSchema{}, nil
	}

	if recursive {
		return extractor.mysqlExtractRecursiveCTE(ctx)
	}
	return extractor.mysqlExtractNonRecursiveCTE(ctx)
}

func (extractor *fieldExtractor) mysqlExtractRecursiveCTE(ctx mysql.ICommonTableExpressionContext) (base.TableSchema, error) {
	cteName := NormalizeMySQLIdentifier(ctx.Identifier())
	l := &recursiveCTEListener{
		extractor: extractor,
		cteInfo: base.TableSchema{
			Name:       cteName,
			ColumnList: []base.ColumnInfo{},
		},
		selfName:                      cteName,
		foundFirstQueryExpressionBody: false,
		inCTE:                         false,
	}
	if ctx.ColumnInternalRefList() != nil {
		columnList := mysqlExtractColumnInternalRefList(ctx.ColumnInternalRefList())
		for i := range columnList {
			l.cteInfo.ColumnList = append(l.cteInfo.ColumnList, base.ColumnInfo{
				Name:              columnList[i],
				MaskingAttributes: base.NewDefaultMaskingAttributes(),
			})
		}
	}
	antlr.ParseTreeWalkerDefault.Walk(l, ctx.Subquery())
	if l.err != nil {
		return base.TableSchema{}, l.err
	}

	return l.cteInfo, nil
}

type recursiveCTEListener struct {
	*mysql.BaseMySQLParserListener

	extractor                     *fieldExtractor
	cteInfo                       base.TableSchema
	selfName                      string
	outerCTEs                     []mysql.IWithClauseContext
	foundFirstQueryExpressionBody bool
	inCTE                         bool
	err                           error
}

// EnterQueryExpression is called when production queryExpression is entered.
func (l *recursiveCTEListener) EnterQueryExpression(ctx *mysql.QueryExpressionContext) {
	if l.foundFirstQueryExpressionBody || l.inCTE || l.err != nil {
		return
	}
	if ctx.WithClause() != nil {
		l.outerCTEs = append(l.outerCTEs, ctx.WithClause())
	}
}

// EnterCommonTableExpression is called when production commonTableExpression is entered.
func (l *recursiveCTEListener) EnterWithClause(_ *mysql.WithClauseContext) {
	l.inCTE = true
}

// ExitCommonTableExpression is called when production commonTableExpression is exited.
func (l *recursiveCTEListener) ExitWithClause(_ *mysql.WithClauseContext) {
	l.inCTE = false
}

// EnterQueryExpressionBody is called when production queryExpressionBody is entered.
func (l *recursiveCTEListener) EnterQueryExpressionBody(ctx *mysql.QueryExpressionBodyContext) {
	if l.err != nil {
		return
	}
	if l.inCTE {
		return
	}
	if l.foundFirstQueryExpressionBody {
		return
	}

	l.foundFirstQueryExpressionBody = true

	// Deal with outer CTEs.
	cetOuterLength := len(l.extractor.cteOuterSchemaInfo)
	defer func() {
		l.extractor.cteOuterSchemaInfo = l.extractor.cteOuterSchemaInfo[:cetOuterLength]
	}()
	for _, outerCTE := range l.outerCTEs {
		recursive := outerCTE.RECURSIVE_SYMBOL() != nil
		for _, cte := range outerCTE.AllCommonTableExpression() {
			cteTable, err := l.extractor.mysqlExtractCommonTableExpression(cte, recursive)
			if err != nil {
				l.err = err
				return
			}
			l.extractor.cteOuterSchemaInfo = append(l.extractor.cteOuterSchemaInfo, cteTable)
		}
	}

	var initialPart []base.FieldInfo
	var recursivePart []antlr.ParserRuleContext

	findRecursivePart := false
	for _, child := range ctx.GetChildren() {
		switch child := child.(type) {
		case *mysql.QueryPrimaryContext:
			if !findRecursivePart {
				resource, err := ExtractResourceList("", "", child.GetParser().GetTokenStream().GetTextFromRuleContext(child))
				if err != nil {
					l.err = err
					return
				}

				for _, item := range resource {
					if item.Database == "" && item.Table == l.selfName {
						findRecursivePart = true
						break
					}
				}
			}

			if findRecursivePart {
				recursivePart = append(recursivePart, child)
			} else {
				fieldList, err := l.extractor.mysqlExtractQueryPrimary(child)
				if err != nil {
					l.err = err
					return
				}
				if len(initialPart) == 0 {
					initialPart = fieldList
				} else {
					if len(initialPart) != len(fieldList) {
						l.err = errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(initialPart), len(fieldList))
						return
					}
					for i := range initialPart {
						initialPart[i].MaskingAttributes.TransmittedBy(fieldList[i].MaskingAttributes)
					}
				}
			}
		case *mysql.QueryExpressionParensContext:
			queryExpression := extractQueryExpression(child)
			if queryExpression == nil {
				// Never happen.
				l.err = errors.Errorf("MySQL query expression parens should have query expression, but got nil")
				return
			}

			if !findRecursivePart {
				resource, err := ExtractResourceList("", "", queryExpression.GetParser().GetTokenStream().GetTextFromRuleContext(queryExpression))
				if err != nil {
					l.err = err
					return
				}

				for _, item := range resource {
					if item.Database == "" && item.Table == l.selfName {
						findRecursivePart = true
						break
					}
				}
			}

			if findRecursivePart {
				recursivePart = append(recursivePart, child)
			} else {
				fieldList, err := l.extractor.mysqlExtractQueryExpression(queryExpression)
				if err != nil {
					l.err = err
					return
				}
				if len(initialPart) == 0 {
					initialPart = fieldList
				} else {
					if len(initialPart) != len(fieldList) {
						l.err = errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(initialPart), len(fieldList))
						return
					}
					for i := range initialPart {
						initialPart[i].MaskingAttributes.TransmittedBy(fieldList[i].MaskingAttributes)
					}
				}
			}
		}
	}

	// Compute dependent closures.
	// There are two ways to compute dependent closures:
	//   1. find the all dependent edges, then use graph theory traversal to find the closure.
	//   2. Iterate to simulate the CTE recursive process, each turn check whether the Sensitive state has changed, and stop if no change.
	//
	// Consider the option 2 can easy to implementation, because the simulate process has been written.
	// On the other hand, the number of iterations of the entire algorithm will not exceed the length of fields.
	// In actual use, the length of fields will not be more than 20 generally.
	// So I think it's OK for now.
	// If any performance issues in use, optimize here.
	if len(l.cteInfo.ColumnList) == 0 {
		for _, item := range initialPart {
			l.cteInfo.ColumnList = append(l.cteInfo.ColumnList, base.ColumnInfo{
				Name:              item.Name,
				MaskingAttributes: item.MaskingAttributes,
			})
		}
	} else {
		if len(initialPart) != len(l.cteInfo.ColumnList) {
			l.err = errors.Errorf("The common table expression and column names list have different column counts")
			return
		}
		for i := range initialPart {
			l.cteInfo.ColumnList[i].MaskingAttributes.TransmittedBy(initialPart[i].MaskingAttributes)
		}
	}

	if len(recursivePart) == 0 {
		return
	}

	l.extractor.cteOuterSchemaInfo = append(l.extractor.cteOuterSchemaInfo, l.cteInfo)
	defer func() {
		l.extractor.cteOuterSchemaInfo = l.extractor.cteOuterSchemaInfo[:len(l.extractor.cteOuterSchemaInfo)-1]
	}()
	for {
		var fieldList []base.FieldInfo
		for _, item := range recursivePart {
			var itemFields []base.FieldInfo
			switch item := item.(type) {
			case *mysql.QueryPrimaryContext:
				var err error
				itemFields, err = l.extractor.mysqlExtractQueryPrimary(item)
				if err != nil {
					l.err = err
					return
				}
			case *mysql.QueryExpressionContext:
				var err error
				itemFields, err = l.extractor.mysqlExtractQueryExpression(item)
				if err != nil {
					l.err = err
					return
				}
			}
			if len(fieldList) == 0 {
				fieldList = itemFields
			} else {
				if len(fieldList) != len(itemFields) {
					l.err = errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(fieldList), len(itemFields))
					return
				}
				for i := range fieldList {
					fieldList[i].MaskingAttributes.TransmittedBy(itemFields[i].MaskingAttributes)
				}
			}
		}

		if len(fieldList) != len(l.cteInfo.ColumnList) {
			// The error content comes from MySQL.
			l.err = errors.Errorf("The common table expression and column names list have different column counts")
			return
		}

		changed := false
		for i, field := range fieldList {
			changed = changed || l.cteInfo.ColumnList[i].MaskingAttributes.TransmittedBy(field.MaskingAttributes)
		}

		if !changed {
			break
		}
		l.extractor.cteOuterSchemaInfo[len(l.extractor.cteOuterSchemaInfo)-1] = l.cteInfo
	}
}

func extractQueryExpression(ctx mysql.IQueryExpressionParensContext) mysql.IQueryExpressionContext {
	if ctx == nil {
		return nil
	}

	switch {
	case ctx.QueryExpression() != nil:
		return ctx.QueryExpression()
	case ctx.QueryExpressionParens() != nil:
		return extractQueryExpression(ctx.QueryExpressionParens())
	}

	return nil
}

func (extractor *fieldExtractor) mysqlExtractNonRecursiveCTE(ctx mysql.ICommonTableExpressionContext) (base.TableSchema, error) {
	fieldList, err := extractor.mysqlExtractSubquery(ctx.Subquery())
	if err != nil {
		return base.TableSchema{}, err
	}
	if ctx.ColumnInternalRefList() != nil {
		columnList := mysqlExtractColumnInternalRefList(ctx.ColumnInternalRefList())
		if len(columnList) != len(fieldList) {
			return base.TableSchema{}, errors.Errorf("MySQL CTE column list should have the same length, but got %d and %d", len(columnList), len(fieldList))
		}
		for i := range fieldList {
			fieldList[i].Name = columnList[i]
		}
	}
	cteName := NormalizeMySQLIdentifier(ctx.Identifier())
	result := base.TableSchema{
		Name:       cteName,
		ColumnList: []base.ColumnInfo{},
	}
	for _, field := range fieldList {
		result.ColumnList = append(result.ColumnList, base.ColumnInfo{
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}
	return result, nil
}

func (extractor *fieldExtractor) mysqlExtractQueryExpressionBody(ctx mysql.IQueryExpressionBodyContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []base.FieldInfo

	for _, child := range ctx.GetChildren() {
		switch child := child.(type) {
		case *mysql.QueryPrimaryContext:
			fieldList, err := extractor.mysqlExtractQueryPrimary(child)
			if err != nil {
				return nil, err
			}
			if len(result) == 0 {
				result = fieldList
			} else {
				if len(result) != len(fieldList) {
					return nil, errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(result), len(fieldList))
				}
				for i := range result {
					result[i].MaskingAttributes.TransmittedBy(fieldList[i].MaskingAttributes)
				}
			}
		case *mysql.QueryExpressionParensContext:
			fieldList, err := extractor.mysqlExtractQueryExpressionParens(child)
			if err != nil {
				return nil, err
			}
			if len(result) == 0 {
				result = fieldList
			} else {
				if len(result) != len(fieldList) {
					return nil, errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(result), len(fieldList))
				}
				for i := range result {
					result[i].MaskingAttributes.TransmittedBy(fieldList[i].MaskingAttributes)
				}
			}
		default:
			continue
		}
	}

	return result, nil
}

func (extractor *fieldExtractor) mysqlExtractQueryPrimary(ctx *mysql.QueryPrimaryContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.QuerySpecification() != nil:
		return extractor.mysqlExtractQuerySpecification(ctx.QuerySpecification())
	case ctx.TableValueConstructor() != nil:
		return extractor.mysqlExtractTableValueConstructor(ctx.TableValueConstructor())
	case ctx.ExplicitTable() != nil:
		return extractor.mysqlExtractExplicitTable(ctx.ExplicitTable())
	}

	return nil, nil
}

func (extractor *fieldExtractor) mysqlExtractTableValueConstructor(ctx mysql.ITableValueConstructorContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	firstRow := ctx.RowValueExplicit(0)
	if firstRow == nil {
		return nil, nil
	}

	values := firstRow.Values()
	if values == nil {
		return nil, nil
	}

	var result []base.FieldInfo

	for _, child := range values.GetChildren() {
		switch child := child.(type) {
		case *mysql.ExprContext:
			_, maskingAttributes, err := extractor.mysqlEvalMaskingAttributesInExpr(child)
			if err != nil {
				return nil, err
			}
			result = append(result, base.FieldInfo{
				Name:              child.GetParser().GetTokenStream().GetTextFromRuleContext(child),
				MaskingAttributes: maskingAttributes,
			})
		case antlr.TerminalNode:
			if child.GetSymbol().GetTokenType() == mysql.MySQLParserDEFAULT_SYMBOL {
				result = append(result, base.FieldInfo{
					Name:              "DEFAULT",
					MaskingAttributes: base.NewDefaultMaskingAttributes(),
				})
			}
		}
	}

	return result, nil
}

func (extractor *fieldExtractor) mysqlExtractExplicitTable(ctx mysql.IExplicitTableContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	databaseName, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	databaseName, tableSchema, err := extractor.mysqlFindTableSchema(databaseName, tableName)
	if err != nil {
		return nil, err
	}

	var res []base.FieldInfo
	for _, column := range tableSchema.ColumnList {
		res = append(res, base.FieldInfo{
			Name:              column.Name,
			Table:             tableSchema.Name,
			Database:          databaseName,
			MaskingAttributes: column.MaskingAttributes,
		})
	}

	return res, nil
}

func (extractor *fieldExtractor) mysqlFindTableSchema(databaseName, tableName string) (string, base.TableSchema, error) {
	// Each CTE name in one WITH clause must be unique, but we can use the same name in the different level CTE, such as:
	//
	//  with tt2 as (
	//    with tt2 as (select * from t)
	//    select max(a) from tt2)
	//  select * from tt2
	//
	// This query has two CTE can be called `tt2`, and the FROM clause 'from tt2' uses the closer tt2 CTE.
	// This is the reason we loop the slice in reversed order.
	for i := len(extractor.cteOuterSchemaInfo) - 1; i >= 0; i-- {
		table := extractor.cteOuterSchemaInfo[i]
		if databaseName == "" && table.Name == tableName {
			return "", table, nil
		}
	}

	for _, database := range extractor.schemaInfo.DatabaseList {
		if len(database.SchemaList) == 0 {
			continue
		}
		tableList := database.SchemaList[0].TableList

		if extractor.schemaInfo.IgnoreCaseSensitive {
			lowerDatabase := strings.ToLower(database.Name)
			lowerTable := strings.ToLower(tableName)
			if lowerDatabase == strings.ToLower(databaseName) || (databaseName == "" && lowerDatabase == strings.ToLower(extractor.currentDatabase)) {
				for _, table := range tableList {
					if lowerTable == strings.ToLower(table.Name) {
						explicitDatabase := databaseName
						if explicitDatabase == "" {
							explicitDatabase = extractor.currentDatabase
						}
						return explicitDatabase, table, nil
					}
				}
			}
		} else if databaseName == database.Name || (databaseName == "" && extractor.currentDatabase == database.Name) {
			for _, table := range tableList {
				if tableName == table.Name {
					explicitDatabase := databaseName
					if explicitDatabase == "" {
						explicitDatabase = extractor.currentDatabase
					}
					return explicitDatabase, table, nil
				}
			}
		}
	}

	database, schema, err := extractor.mysqlFindViewSchema(databaseName, tableName)
	if err == nil {
		return database, schema, nil
	}
	return "", base.TableSchema{}, errors.Wrapf(err, "Table or view %q.%q not found", databaseName, tableName)
}

func (extractor *fieldExtractor) mysqlFindViewSchema(databaseName, viewName string) (string, base.TableSchema, error) {
	for _, database := range extractor.schemaInfo.DatabaseList {
		if len(database.SchemaList) == 0 {
			continue
		}
		viewList := database.SchemaList[0].ViewList

		if extractor.schemaInfo.IgnoreCaseSensitive {
			lowerDatabase := strings.ToLower(database.Name)
			lowerView := strings.ToLower(viewName)
			if lowerDatabase == strings.ToLower(databaseName) || (databaseName == "" && lowerDatabase == strings.ToLower(extractor.currentDatabase)) {
				for _, view := range viewList {
					if lowerView == strings.ToLower(view.Name) {
						explicitDatabase := databaseName
						if explicitDatabase == "" {
							explicitDatabase = extractor.currentDatabase
						}

						table, err := extractor.mysqlBuildTableSchemaForView(view.Name, view.Definition)
						return explicitDatabase, table, err
					}
				}
			}
		} else if databaseName == database.Name || (databaseName == "" && extractor.currentDatabase == database.Name) {
			for _, view := range viewList {
				if viewName == view.Name {
					explicitDatabase := databaseName
					if explicitDatabase == "" {
						explicitDatabase = extractor.currentDatabase
					}

					table, err := extractor.mysqlBuildTableSchemaForView(view.Name, view.Definition)
					return explicitDatabase, table, err
				}
			}
		}
	}
	return "", base.TableSchema{}, errors.Errorf("View %q.%q not found", databaseName, viewName)
}

func (extractor *fieldExtractor) mysqlBuildTableSchemaForView(viewName string, viewDefinition string) (base.TableSchema, error) {
	list, err := ParseMySQL(viewDefinition)
	if err != nil {
		return base.TableSchema{}, err
	}
	if len(list) == 0 {
		return base.TableSchema{}, errors.Errorf("MySQL view definition should only have one statement, but got %d", len(list))
	}
	if len(list) != 1 {
		return base.TableSchema{}, errors.Errorf("MySQL view definition should only have one statement, but got %d", len(list))
	}

	listener := &mysqlSensitiveFieldListener{
		extractor: extractor,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, list[0].Tree)
	if listener.err != nil {
		return base.TableSchema{}, listener.err
	}

	result := base.TableSchema{
		Name:       viewName,
		ColumnList: []base.ColumnInfo{},
	}
	for _, field := range listener.result {
		// nolint:gosimple
		result.ColumnList = append(result.ColumnList, base.ColumnInfo{
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}
	return result, nil
}

func (extractor *fieldExtractor) mysqlExtractQuerySpecification(ctx mysql.IQuerySpecificationContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var fromFieldList []base.FieldInfo
	var err error
	if ctx.FromClause() != nil {
		fromFieldList, err = extractor.mysqlExtractFromClause(ctx.FromClause())
		if err != nil {
			return nil, err
		}
		extractor.fromFieldList = fromFieldList
		defer func() {
			extractor.fromFieldList = nil
		}()
	}

	return extractor.mysqlExtractSelectItemList(ctx.SelectItemList())
}

func (extractor *fieldExtractor) mysqlExtractSelectItemList(ctx mysql.ISelectItemListContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []base.FieldInfo

	if ctx.MULT_OPERATOR() != nil {
		result = append(result, extractor.fromFieldList...)
	}

	for _, selectItem := range ctx.AllSelectItem() {
		fieldList, err := extractor.mysqlExtractSelectItem(selectItem)
		if err != nil {
			return nil, err
		}
		result = append(result, fieldList...)
	}

	return result, nil
}

func (extractor *fieldExtractor) mysqlExtractSelectItem(ctx mysql.ISelectItemContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.TableWild() != nil:
		return extractor.mysqlExtractTableWild(ctx.TableWild())
	case ctx.Expr() != nil:
		fieldName, sensitiveLevel, err := extractor.mysqlEvalMaskingAttributesInExpr(ctx.Expr())
		if err != nil {
			return nil, err
		}
		if ctx.SelectAlias() != nil {
			fieldName = NormalizeMySQLSelectAlias(ctx.SelectAlias())
		} else if fieldName == "" {
			fieldName = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
		}
		return []base.FieldInfo{
			{
				Name:              fieldName,
				MaskingAttributes: sensitiveLevel,
			},
		}, nil
	}

	return nil, nil
}

func (extractor *fieldExtractor) mysqlExtractTableWild(ctx mysql.ITableWildContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var ids []string
	for _, identifier := range ctx.AllIdentifier() {
		ids = append(ids, NormalizeMySQLIdentifier(identifier))
	}

	var databaseName, tableName string

	switch {
	case len(ids) == 1:
		tableName = ids[0]
	case len(ids) == 2:
		databaseName = ids[0]
		tableName = ids[1]
	default:
		return nil, errors.Errorf("MySQL table wild should have 1 or 2 identifiers, but got %d", len(ids))
	}

	var result []base.FieldInfo

	for _, field := range extractor.fromFieldList {
		sameDatabase := false
		sameTable := false
		if extractor.schemaInfo.IgnoreCaseSensitive {
			sameDatabase = (strings.EqualFold(field.Database, databaseName)) ||
				(databaseName == "" && strings.EqualFold(field.Database, extractor.currentDatabase)) ||
				(field.Database == "" && strings.EqualFold(extractor.currentDatabase, databaseName)) ||
				(databaseName == "" && field.Database == "")
			sameTable = strings.EqualFold(field.Table, tableName)
		} else {
			sameDatabase = (field.Database == databaseName) ||
				(databaseName == "" && field.Database == extractor.currentDatabase) ||
				(field.Database == "" && extractor.currentDatabase == databaseName) ||
				(databaseName == "" && field.Database == "")
			sameTable = field.Table == tableName
		}

		if sameDatabase && sameTable {
			result = append(result, field)
		}
	}

	return result, nil
}

func (extractor *fieldExtractor) mysqlExtractFromClause(ctx mysql.IFromClauseContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.DUAL_SYMBOL() != nil {
		return []base.FieldInfo{}, nil
	}

	return extractor.mysqlExtractTableReferenceList(ctx.TableReferenceList())
}

func (extractor *fieldExtractor) mysqlExtractTableReference(ctx mysql.ITableReferenceContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.TableFactor() == nil {
		return nil, errors.Errorf("MySQL table reference should have table factor")
	}

	fieldList, err := extractor.mysqlExtractTableFactor(ctx.TableFactor())
	if err != nil {
		return nil, err
	}

	if len(ctx.AllJoinedTable()) == 0 {
		return fieldList, nil
	}

	for _, joinedTable := range ctx.AllJoinedTable() {
		fieldList, err = extractor.mysqlMergeJoin(fieldList, joinedTable)
		if err != nil {
			return nil, err
		}
	}

	return fieldList, nil
}

func (extractor *fieldExtractor) mysqlMergeJoin(leftField []base.FieldInfo, joinedTable mysql.IJoinedTableContext) ([]base.FieldInfo, error) {
	if joinedTable == nil {
		return leftField, nil
	}

	leftFieldMap := make(map[string]base.FieldInfo)
	for _, left := range leftField {
		// Column name in MySQL is NOT case sensitive.
		leftFieldMap[strings.ToLower(left.Name)] = left
	}

	switch {
	case joinedTable.InnerJoinType() != nil:
		rightFiled, err := extractor.mysqlExtractTableReference(joinedTable.TableReference())
		if err != nil {
			return nil, err
		}

		if joinedTable.InnerJoinType().CROSS_SYMBOL() != nil || joinedTable.USING_SYMBOL() == nil {
			return append(leftField, rightFiled...), nil
		}

		// ... JOIN ... USING (...) will merge the column in USING.
		usingMap := make(map[string]bool)
		for _, identifier := range NormalizeMySQLIdentifierList(joinedTable.IdentifierListWithParentheses().IdentifierList()) {
			// Column name in MySQL is NOT case sensitive.
			usingMap[strings.ToLower(identifier)] = true
		}

		var result []base.FieldInfo

		rightFieldMap := make(map[string]base.FieldInfo)
		for _, right := range rightFiled {
			// Column name in MySQL is NOT case sensitive.
			rightFieldMap[strings.ToLower(right.Name)] = right
		}
		for _, left := range leftField {
			_, existsInUsingMap := usingMap[strings.ToLower(left.Name)]
			rField, existsInRightField := rightFieldMap[strings.ToLower(left.Name)]
			// Merge the sensitive attribute for the column in USING.
			if existsInUsingMap && existsInRightField {
				left.MaskingAttributes.TransmittedBy(rField.MaskingAttributes)
			}
			result = append(result, left)
		}

		for _, right := range rightFiled {
			_, existsInUsingMap := usingMap[strings.ToLower(right.Name)]
			_, existsInLeftField := leftFieldMap[strings.ToLower(right.Name)]
			if existsInUsingMap && existsInLeftField {
				continue
			}
			result = append(result, right)
		}
		return result, nil
	case joinedTable.OuterJoinType() != nil:
		rightFiled, err := extractor.mysqlExtractTableReference(joinedTable.TableReference())
		if err != nil {
			return nil, err
		}

		if joinedTable.USING_SYMBOL() == nil {
			return append(leftField, rightFiled...), nil
		}

		// ... JOIN ... USING (...) will merge the column in USING.
		usingMap := make(map[string]bool)
		for _, identifier := range NormalizeMySQLIdentifierList(joinedTable.IdentifierListWithParentheses().IdentifierList()) {
			// Column name in MySQL is NOT case sensitive.
			usingMap[strings.ToLower(identifier)] = true
		}

		var result []base.FieldInfo

		rightFieldMap := make(map[string]base.FieldInfo)
		for _, right := range rightFiled {
			// Column name in MySQL is NOT case sensitive.
			rightFieldMap[strings.ToLower(right.Name)] = right
		}
		for _, left := range leftField {
			_, existsInUsingMap := usingMap[strings.ToLower(left.Name)]
			rField, existsInRightField := rightFieldMap[strings.ToLower(left.Name)]
			// Merge the sensitive attribute for the column in USING.
			if existsInUsingMap && existsInRightField {
				left.MaskingAttributes.TransmittedBy(rField.MaskingAttributes)
			}
			result = append(result, left)
		}

		for _, right := range rightFiled {
			_, existsInUsingMap := usingMap[strings.ToLower(right.Name)]
			_, existsInLeftField := leftFieldMap[strings.ToLower(right.Name)]
			if existsInUsingMap && existsInLeftField {
				continue
			}
			result = append(result, right)
		}
		return result, nil
	case joinedTable.NaturalJoinType() != nil:
		var result []base.FieldInfo
		rightFiled, err := extractor.mysqlExtractTableReference(joinedTable.TableReference())
		if err != nil {
			return nil, err
		}
		rightFieldMap := make(map[string]base.FieldInfo)
		for _, right := range rightFiled {
			// Column name in MySQL is NOT case sensitive.
			rightFieldMap[strings.ToLower(right.Name)] = right
		}

		// Natural join will merge the column with the same name.
		for _, left := range leftField {
			if rField, exists := rightFieldMap[strings.ToLower(left.Name)]; exists {
				left.MaskingAttributes.TransmittedBy(rField.MaskingAttributes)
			}
			result = append(result, left)
		}

		for _, right := range rightFiled {
			if _, exists := leftFieldMap[strings.ToLower(right.Name)]; exists {
				continue
			}
			result = append(result, right)
		}
		return result, nil
	}

	// Never reach here.
	return nil, errors.New("Unsupported join type")
}

func (extractor *fieldExtractor) mysqlExtractTableFactor(ctx mysql.ITableFactorContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.SingleTable() != nil:
		return extractor.mysqlExtractSingleTable(ctx.SingleTable())
	case ctx.SingleTableParens() != nil:
		return extractor.mysqlExtractSingleTableParens(ctx.SingleTableParens())
	case ctx.DerivedTable() != nil:
		return extractor.mysqlExtractDerivedTable(ctx.DerivedTable())
	case ctx.TableReferenceListParens() != nil:
		return extractor.mysqlExtractTableReferenceListParens(ctx.TableReferenceListParens())
	case ctx.TableFunction() != nil:
		return extractor.mysqlExtractTableFunction(ctx.TableFunction())
	}

	return nil, nil
}

func (extractor *fieldExtractor) mysqlExtractSingleTable(ctx mysql.ISingleTableContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	databaseName, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	databaseName, tableSchema, err := extractor.mysqlFindTableSchema(databaseName, tableName)
	if err != nil {
		return nil, err
	}

	tableName = tableSchema.Name
	if ctx.TableAlias() != nil {
		tableName = NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
	}

	var result []base.FieldInfo
	for _, column := range tableSchema.ColumnList {
		result = append(result, base.FieldInfo{
			Name:              column.Name,
			Table:             tableName,
			Database:          databaseName,
			MaskingAttributes: column.MaskingAttributes,
		})
	}

	return result, nil
}

func (extractor *fieldExtractor) mysqlExtractSingleTableParens(ctx mysql.ISingleTableParensContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.SingleTable() != nil:
		return extractor.mysqlExtractSingleTable(ctx.SingleTable())
	case ctx.SingleTableParens() != nil:
		return extractor.mysqlExtractSingleTableParens(ctx.SingleTableParens())
	}

	return nil, nil
}

func (extractor *fieldExtractor) mysqlExtractDerivedTable(ctx mysql.IDerivedTableContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	fieldList, err := extractor.mysqlExtractSubquery(ctx.Subquery())
	if err != nil {
		return nil, err
	}

	if ctx.TableAlias() != nil {
		alias := NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
		for i := range fieldList {
			fieldList[i].Table = alias
		}
	}

	if ctx.ColumnInternalRefList() != nil {
		columnList := mysqlExtractColumnInternalRefList(ctx.ColumnInternalRefList())
		if len(columnList) != len(fieldList) {
			return nil, errors.Errorf("MySQL derived table column list should have the same length, but got %d and %d", len(columnList), len(fieldList))
		}
		for i := range fieldList {
			fieldList[i].Name = columnList[i]
		}
	}

	return fieldList, nil
}

func mysqlExtractColumnInternalRefList(ctx mysql.IColumnInternalRefListContext) []string {
	if ctx == nil {
		return nil
	}

	var result []string
	for _, columnInternalRef := range ctx.AllColumnInternalRef() {
		result = append(result, NormalizeMySQLIdentifier(columnInternalRef.Identifier()))
	}
	return result
}

func (extractor *fieldExtractor) mysqlExtractSubquery(ctx mysql.ISubqueryContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	return extractor.mysqlExtractQueryExpressionParens(ctx.QueryExpressionParens())
}

func (extractor *fieldExtractor) mysqlExtractTableReferenceListParens(ctx mysql.ITableReferenceListParensContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.TableReferenceList() != nil:
		return extractor.mysqlExtractTableReferenceList(ctx.TableReferenceList())
	case ctx.TableReferenceListParens() != nil:
		return extractor.mysqlExtractTableReferenceListParens(ctx.TableReferenceListParens())
	}

	return nil, nil
}

func (extractor *fieldExtractor) mysqlExtractTableReferenceList(ctx mysql.ITableReferenceListContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []base.FieldInfo

	for _, tableRef := range ctx.AllTableReference() {
		fieldList, err := extractor.mysqlExtractTableReference(tableRef)
		if err != nil {
			return nil, err
		}
		result = append(result, fieldList...)
	}

	return result, nil
}

func (extractor *fieldExtractor) mysqlExtractTableFunction(ctx mysql.ITableFunctionContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	tableName, sensitiveLevel, err := extractor.mysqlEvalMaskingAttributesInExpr(ctx.Expr())
	if err != nil {
		return nil, err
	}

	columnList := mysqlExtractColumnsClause(ctx.ColumnsClause())

	if ctx.TableAlias() != nil {
		tableName = NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
	} else if tableName == "" {
		tableName = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Expr())
	}

	var result []base.FieldInfo
	for _, column := range columnList {
		result = append(result, base.FieldInfo{
			Name:              column,
			Table:             tableName,
			MaskingAttributes: sensitiveLevel,
		})
	}
	return result, nil
}

func mysqlExtractColumnsClause(ctx mysql.IColumnsClauseContext) []string {
	if ctx == nil {
		return nil
	}

	var result []string
	for _, column := range ctx.AllJtColumn() {
		result = append(result, mysqlExtractJtColumn(column)...)
	}

	return result
}

func mysqlExtractJtColumn(ctx mysql.IJtColumnContext) []string {
	if ctx == nil {
		return []string{}
	}

	switch {
	case ctx.Identifier() != nil:
		return []string{NormalizeMySQLIdentifier(ctx.Identifier())}
	case ctx.ColumnsClause() != nil:
		return mysqlExtractColumnsClause(ctx.ColumnsClause())
	}

	return []string{}
}

func (extractor *fieldExtractor) mysqlEvalMaskingAttributesInExpr(ctx antlr.ParserRuleContext) (string, base.MaskingAttributes, error) {
	if ctx == nil {
		return "", base.NewDefaultMaskingAttributes(), nil
	}

	attributes := base.NewDefaultMaskingAttributes()

	switch ctx := ctx.(type) {
	case mysql.ISubqueryContext:
		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &fieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.mysqlExtractSubquery(ctx)
		if err != nil {
			return "", base.NewDefaultMaskingAttributes(), err
		}
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, field := range fieldList {
			attributes.TransmittedByInExpression(field.MaskingAttributes)
			if attributes.IsNeverChangeInTransmission() {
				return "", attributes, nil
			}
		}
		return "", finalAttributes, nil
	case mysql.IColumnRefContext:
		databaseName, tableName, fieldName := NormalizeMySQLFieldIdentifier(ctx.FieldIdentifier())
		maskingAttributes := extractor.mysqlCheckFieldMaskingAttributes(databaseName, tableName, fieldName)
		return fieldName, maskingAttributes, nil
	}

	var list []antlr.ParserRuleContext
	for _, child := range ctx.GetChildren() {
		if child, ok := child.(antlr.ParserRuleContext); ok {
			list = append(list, child)
		}
	}

	fieldName, maskingAttributes, err := extractor.mysqlEvalMaskingAttributesInExprList(list)
	if err != nil {
		return "", base.NewDefaultMaskingAttributes(), err
	}
	if len(ctx.GetChildren()) > 1 {
		fieldName = ""
	}
	return fieldName, maskingAttributes, nil
}

func (extractor *fieldExtractor) mysqlEvalMaskingAttributesInExprList(list []antlr.ParserRuleContext) (string, base.MaskingAttributes, error) {
	finalAttributes := base.NewDefaultMaskingAttributes()
	var fieldName string
	var err error
	var attributes base.MaskingAttributes
	for _, ctx := range list {
		fieldName, attributes, err = extractor.mysqlEvalMaskingAttributesInExpr(ctx)
		if err != nil {
			return "", base.NewDefaultMaskingAttributes(), err
		}
		if len(list) != 1 {
			fieldName = ""
		}
		finalAttributes.TransmittedByInExpression(attributes)
		if finalAttributes.IsNeverChangeInTransmission() {
			return fieldName, finalAttributes, nil
		}
	}
	return fieldName, finalAttributes, nil
}

func (extractor *fieldExtractor) mysqlCheckFieldMaskingAttributes(databaseName string, tableName string, columnName string) base.MaskingAttributes {
	// One sub-query may have multi-outer schemas and the multi-outer schemas can use the same name, such as:
	//
	//  select (
	//    select (
	//      select max(a) > x1.a from t
	//    )
	//    from t1 as x1
	//    limit 1
	//  )
	//  from t as x1;
	//
	// This query has two tables can be called `x1`, and the expression x1.a uses the closer x1 table.
	// This is the reason we loop the slice in reversed order.
	for i := len(extractor.outerSchemaInfo) - 1; i >= 0; i-- {
		field := extractor.outerSchemaInfo[i]
		var sameDatabase, sameTable, sameField bool
		if extractor.schemaInfo.IgnoreCaseSensitive {
			sameDatabase = (strings.EqualFold(databaseName, field.Database) ||
				(databaseName == "" && strings.EqualFold(field.Database, extractor.currentDatabase))) ||
				(databaseName == "" && field.Database == "")
			sameTable = (strings.EqualFold(tableName, field.Table) || tableName == "")
		} else {
			sameDatabase = (databaseName == field.Database ||
				(databaseName == "" && field.Database == extractor.currentDatabase) ||
				(databaseName == "" && field.Database == ""))
			sameTable = (tableName == field.Table || tableName == "")
		}
		// Column name in MySQL is NOT case sensitive.
		sameField = strings.EqualFold(columnName, field.Name)
		if sameDatabase && sameTable && sameField {
			return field.MaskingAttributes
		}
	}

	for _, field := range extractor.fromFieldList {
		var sameDatabase, sameTable, sameField bool
		if extractor.schemaInfo.IgnoreCaseSensitive {
			sameDatabase = (strings.EqualFold(databaseName, field.Database) ||
				(databaseName == "" && strings.EqualFold(field.Database, extractor.currentDatabase)) ||
				(databaseName == "" && field.Database == ""))
			sameTable = (strings.EqualFold(tableName, field.Table) || tableName == "")
		} else {
			sameDatabase = (databaseName == field.Database ||
				(databaseName == "" && field.Database == extractor.currentDatabase) ||
				(databaseName == "" && field.Database == ""))
			sameTable = (tableName == field.Table || tableName == "")
		}
		// Column name in MySQL is NOT case sensitive.
		sameField = strings.EqualFold(columnName, field.Name)
		if sameDatabase && sameTable && sameField {
			return field.MaskingAttributes
		}
	}
	return base.NewDefaultMaskingAttributes()
}
