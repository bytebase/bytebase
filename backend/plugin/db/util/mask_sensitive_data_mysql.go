package util

import (
	"cmp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (extractor *sensitiveFieldExtractor) extractMySQLSensitiveField(statement string) ([]db.SensitiveField, error) {
	list, err := parser.ParseMySQL(statement)
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

	extractor *sensitiveFieldExtractor
	result    []db.SensitiveField
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
		l.result = append(l.result, db.SensitiveField{
			Name:         field.name,
			MaskingLevel: field.maskingLevel,
		})
	}
}

func (extractor *sensitiveFieldExtractor) mysqlExtractContext(ctx antlr.ParserRuleContext) ([]fieldInfo, error) {
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

func (extractor *sensitiveFieldExtractor) mysqlExtractSelectStatement(ctx mysql.ISelectStatementContext) ([]fieldInfo, error) {
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

func (extractor *sensitiveFieldExtractor) mysqlExtractQueryExpressionParens(ctx mysql.IQueryExpressionParensContext) ([]fieldInfo, error) {
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

func (extractor *sensitiveFieldExtractor) mysqlExtractQueryExpression(ctx mysql.IQueryExpressionContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	// TODO: support WITH

	switch {
	case ctx.QueryExpressionParens() != nil:
		return extractor.mysqlExtractQueryExpressionParens(ctx.QueryExpressionParens())
	case ctx.QueryExpressionBody() != nil:
		return extractor.mysqlExtractQueryExpressionBody(ctx.QueryExpressionBody())
	}

	return nil, nil
}

func (extractor *sensitiveFieldExtractor) mysqlExtractQueryExpressionBody(ctx mysql.IQueryExpressionBodyContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []fieldInfo

	for _, child := range ctx.GetChildren() {
		switch child := child.(type) {
		case *mysql.QueryPrimaryContext:
			fieldList, err := extractor.mysqlExtractQueryPrimary(child)
			if err != nil {
				return nil, err
			}
			if len(fieldList) == 0 {
				result = fieldList
			} else {
				if len(result) != len(fieldList) {
					return nil, errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(result), len(fieldList))
				}
				for i := range result {
					if result[i].maskingLevel < fieldList[i].maskingLevel {
						result[i].maskingLevel = fieldList[i].maskingLevel
					}
				}
			}
		case *mysql.QueryExpressionParensContext:
			fieldList, err := extractor.mysqlExtractQueryExpressionParens(child)
			if err != nil {
				return nil, err
			}
			if len(fieldList) == 0 {
				result = fieldList
			} else {
				if len(result) != len(fieldList) {
					return nil, errors.Errorf("MySQL UNION field list should have the same length, but got %d and %d", len(result), len(fieldList))
				}
				for i := range result {
					if result[i].maskingLevel < fieldList[i].maskingLevel {
						result[i].maskingLevel = fieldList[i].maskingLevel
					}
				}
			}
		default:
			continue
		}
	}

	return nil, nil
}

func (extractor *sensitiveFieldExtractor) mysqlExtractQueryPrimary(ctx *mysql.QueryPrimaryContext) ([]fieldInfo, error) {
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

func (extractor *sensitiveFieldExtractor) mysqlExtractTableValueConstructor(ctx mysql.ITableValueConstructorContext) ([]fieldInfo, error) {
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

	var result []fieldInfo

	for _, child := range values.GetChildren() {
		switch child := child.(type) {
		case *mysql.ExprContext:
			_, maskingLevel, err := extractor.mysqlEvalMaskingLevelInExpr(child)
			if err != nil {
				return nil, err
			}
			result = append(result, fieldInfo{
				name:         child.GetParser().GetTokenStream().GetTextFromRuleContext(child),
				maskingLevel: maskingLevel,
			})
		case antlr.TerminalNode:
			if child.GetSymbol().GetTokenType() == mysql.MySQLParserDEFAULT_SYMBOL {
				result = append(result, fieldInfo{
					name:         "DEFAULT",
					maskingLevel: defaultMaskingLevel,
				})
			}
		}
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) mysqlExtractExplicitTable(ctx mysql.IExplicitTableContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	databaseName, tableName := parser.NormalizeMySQLTableRef(ctx.TableRef())
	databaseName, tableSchema, err := extractor.mysqlFindTableSchema(databaseName, tableName)
	if err != nil {
		return nil, err
	}

	var res []fieldInfo
	for _, column := range tableSchema.ColumnList {
		res = append(res, fieldInfo{
			name:         column.Name,
			table:        tableSchema.Name,
			database:     databaseName,
			maskingLevel: column.MaskingLevel,
		})
	}

	return res, nil
}

func (extractor *sensitiveFieldExtractor) mysqlFindTableSchema(databaseName, tableName string) (string, db.TableSchema, error) {
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
	return "", db.TableSchema{}, errors.Wrapf(err, "Table or view %q.%q not found", databaseName, tableName)
}

func (extractor *sensitiveFieldExtractor) mysqlFindViewSchema(databaseName, tableName string) (string, db.TableSchema, error) {
	// TODO: support VIEW
	return "", db.TableSchema{}, errors.Errorf("MySQL VIEW is not supported yet")
}

func (extractor *sensitiveFieldExtractor) mysqlExtractQuerySpecification(ctx mysql.IQuerySpecificationContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var fromFieldList []fieldInfo
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

func (extractor *sensitiveFieldExtractor) mysqlExtractSelectItemList(ctx mysql.ISelectItemListContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []fieldInfo

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

func (extractor *sensitiveFieldExtractor) mysqlExtractSelectItem(ctx mysql.ISelectItemContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch {
	case ctx.TableWild() != nil:
		return extractor.mysqlExtractTableWild(ctx.TableWild())
	case ctx.Expr() != nil:
		fieldName, sensitiveLevel, err := extractor.mysqlEvalMaskingLevelInExpr(ctx.Expr())
		if err != nil {
			return nil, err
		}
		if ctx.SelectAlias() != nil {
			fieldName = parser.NormalizeMySQLSelectAlias(ctx.SelectAlias())
		} else if fieldName == "" {
			fieldName = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
		}
		return []fieldInfo{
			{
				name:         fieldName,
				maskingLevel: sensitiveLevel,
			},
		}, nil
	}

	return nil, nil
}

func (extractor *sensitiveFieldExtractor) mysqlExtractTableWild(ctx mysql.ITableWildContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var ids []string
	for _, identifier := range ctx.AllIdentifier() {
		ids = append(ids, parser.NormalizeMySQLIdentifier(identifier))
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

	var result []fieldInfo

	for _, field := range extractor.fromFieldList {
		sameDatabase := false
		sameTable := false
		if extractor.schemaInfo.IgnoreCaseSensitive {
			sameDatabase = (strings.EqualFold(field.database, databaseName)) ||
				(databaseName == "" && strings.EqualFold(field.database, extractor.currentDatabase)) ||
				(field.database == "" && strings.EqualFold(extractor.currentDatabase, databaseName)) ||
				(databaseName == "" && field.database == "")
			sameTable = strings.EqualFold(field.table, tableName)
		} else {
			sameDatabase = (field.database == databaseName) ||
				(databaseName == "" && field.database == extractor.currentDatabase) ||
				(field.database == "" && extractor.currentDatabase == databaseName) ||
				(databaseName == "" && field.database == "")
			sameTable = field.table == tableName
		}

		if sameDatabase && sameTable {
			result = append(result, field)
		}
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) mysqlExtractFromClause(ctx mysql.IFromClauseContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.DUAL_SYMBOL() != nil {
		return []fieldInfo{}, nil
	}

	return extractor.mysqlExtractTableReferenceList(ctx.TableReferenceList())
}

func (extractor *sensitiveFieldExtractor) mysqlExtractTableReference(ctx mysql.ITableReferenceContext) ([]fieldInfo, error) {
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

func (extractor *sensitiveFieldExtractor) mysqlMergeJoin(leftField []fieldInfo, joinedTable mysql.IJoinedTableContext) ([]fieldInfo, error) {
	if joinedTable == nil {
		return leftField, nil
	}

	leftFieldMap := make(map[string]fieldInfo)
	for _, left := range leftField {
		// Column name in MySQL is NOT case sensitive.
		leftFieldMap[strings.ToLower(left.name)] = left
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
		for _, identifier := range parser.NormalizeMySQLIdentifierList(joinedTable.IdentifierListWithParentheses().IdentifierList()) {
			// Column name in MySQL is NOT case sensitive.
			usingMap[strings.ToLower(identifier)] = true
		}

		var result []fieldInfo

		rightFieldMap := make(map[string]fieldInfo)
		for _, right := range rightFiled {
			// Column name in MySQL is NOT case sensitive.
			rightFieldMap[strings.ToLower(right.name)] = right
		}
		for _, left := range leftField {
			_, existsInUsingMap := usingMap[strings.ToLower(left.name)]
			rField, existsInRightField := rightFieldMap[strings.ToLower(left.name)]
			// Merge the sensitive attribute for the column in USING.
			if existsInUsingMap && existsInRightField && cmp.Less[storepb.MaskingLevel](rField.maskingLevel, left.maskingLevel) {
				left.maskingLevel = rField.maskingLevel
			}
			result = append(result, left)
		}

		for _, right := range rightFiled {
			_, existsInUsingMap := usingMap[strings.ToLower(right.name)]
			_, existsInLeftField := leftFieldMap[strings.ToLower(right.name)]
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
		for _, identifier := range parser.NormalizeMySQLIdentifierList(joinedTable.IdentifierListWithParentheses().IdentifierList()) {
			// Column name in MySQL is NOT case sensitive.
			usingMap[strings.ToLower(identifier)] = true
		}

		var result []fieldInfo

		rightFieldMap := make(map[string]fieldInfo)
		for _, right := range rightFiled {
			// Column name in MySQL is NOT case sensitive.
			rightFieldMap[strings.ToLower(right.name)] = right
		}
		for _, left := range leftField {
			_, existsInUsingMap := usingMap[strings.ToLower(left.name)]
			rField, existsInRightField := rightFieldMap[strings.ToLower(left.name)]
			// Merge the sensitive attribute for the column in USING.
			if existsInUsingMap && existsInRightField && cmp.Less[storepb.MaskingLevel](rField.maskingLevel, left.maskingLevel) {
				left.maskingLevel = rField.maskingLevel
			}
			result = append(result, left)
		}

		for _, right := range rightFiled {
			_, existsInUsingMap := usingMap[strings.ToLower(right.name)]
			_, existsInLeftField := leftFieldMap[strings.ToLower(right.name)]
			if existsInUsingMap && existsInLeftField {
				continue
			}
			result = append(result, right)
		}
		return result, nil
	case joinedTable.NaturalJoinType() != nil:
		var result []fieldInfo
		rightFiled, err := extractor.mysqlExtractTableReference(joinedTable.TableReference())
		if err != nil {
			return nil, err
		}
		rightFieldMap := make(map[string]fieldInfo)
		for _, right := range rightFiled {
			// Column name in MySQL is NOT case sensitive.
			rightFieldMap[strings.ToLower(right.name)] = right
		}

		// Natural join will merge the column with the same name.
		for _, left := range leftField {
			if rField, exists := rightFieldMap[strings.ToLower(left.name)]; exists && cmp.Less[storepb.MaskingLevel](left.maskingLevel, rField.maskingLevel) {
				left.maskingLevel = rField.maskingLevel
			}
			result = append(result, left)
		}

		for _, right := range rightFiled {
			if _, exists := leftFieldMap[strings.ToLower(right.name)]; exists {
				continue
			}
			result = append(result, right)
		}
		return result, nil
	}

	// Never reach here.
	return nil, errors.New("Unsupported join type")
}

func (extractor *sensitiveFieldExtractor) mysqlExtractTableFactor(ctx mysql.ITableFactorContext) ([]fieldInfo, error) {
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

func (extractor *sensitiveFieldExtractor) mysqlExtractSingleTable(ctx mysql.ISingleTableContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	databaseName, tableName := parser.NormalizeMySQLTableRef(ctx.TableRef())
	databaseName, tableSchema, err := extractor.mysqlFindTableSchema(databaseName, tableName)
	if err != nil {
		return nil, err
	}

	tableName = tableSchema.Name
	if ctx.TableAlias() != nil {
		tableName = parser.NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
	}

	var result []fieldInfo
	for _, column := range tableSchema.ColumnList {
		result = append(result, fieldInfo{
			name:         column.Name,
			table:        tableName,
			database:     databaseName,
			maskingLevel: column.MaskingLevel,
		})
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) mysqlExtractSingleTableParens(ctx mysql.ISingleTableParensContext) ([]fieldInfo, error) {
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

func (extractor *sensitiveFieldExtractor) mysqlExtractDerivedTable(ctx mysql.IDerivedTableContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	fieldList, err := extractor.mysqlExtractSubquery(ctx.Subquery())
	if err != nil {
		return nil, err
	}

	if ctx.TableAlias() != nil {
		alias := parser.NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
		for i := range fieldList {
			fieldList[i].table = alias
		}
	}

	if ctx.ColumnInternalRefList() != nil {
		columnList := mysqlExtractColumnInternalRefList(ctx.ColumnInternalRefList())
		if len(columnList) != len(fieldList) {
			return nil, errors.Errorf("MySQL derived table column list should have the same length, but got %d and %d", len(columnList), len(fieldList))
		}
		for i := range fieldList {
			fieldList[i].name = columnList[i]
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
		result = append(result, parser.NormalizeMySQLIdentifier(columnInternalRef.Identifier()))
	}
	return result
}

func (extractor *sensitiveFieldExtractor) mysqlExtractSubquery(ctx mysql.ISubqueryContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	return extractor.mysqlExtractQueryExpressionParens(ctx.QueryExpressionParens())
}

func (extractor *sensitiveFieldExtractor) mysqlExtractTableReferenceListParens(ctx mysql.ITableReferenceListParensContext) ([]fieldInfo, error) {
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

func (extractor *sensitiveFieldExtractor) mysqlExtractTableReferenceList(ctx mysql.ITableReferenceListContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []fieldInfo

	for _, tableRef := range ctx.AllTableReference() {
		fieldList, err := extractor.mysqlExtractTableReference(tableRef)
		if err != nil {
			return nil, err
		}
		result = append(result, fieldList...)
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) mysqlExtractTableFunction(ctx mysql.ITableFunctionContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	tableName, sensitiveLevel, err := extractor.mysqlEvalMaskingLevelInExpr(ctx.Expr())
	if err != nil {
		return nil, err
	}

	columnList := mysqlExtractColumnsClause(ctx.ColumnsClause())

	if ctx.TableAlias() != nil {
		tableName = parser.NormalizeMySQLIdentifier(ctx.TableAlias().Identifier())
	} else if tableName == "" {
		tableName = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Expr())
	}

	var result []fieldInfo
	for _, column := range columnList {
		result = append(result, fieldInfo{
			name:         column,
			table:        tableName,
			maskingLevel: sensitiveLevel,
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
		return []string{parser.NormalizeMySQLIdentifier(ctx.Identifier())}
	case ctx.ColumnsClause() != nil:
		return mysqlExtractColumnsClause(ctx.ColumnsClause())
	}

	return []string{}
}

func (extractor *sensitiveFieldExtractor) mysqlEvalMaskingLevelInExpr(ctx antlr.ParserRuleContext) (string, storepb.MaskingLevel, error) {
	if ctx == nil {
		return "", defaultMaskingLevel, nil
	}

	switch ctx := ctx.(type) {
	case mysql.ISubqueryContext:
		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &sensitiveFieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.mysqlExtractSubquery(ctx)
		if err != nil {
			return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, err
		}
		finalLevel := defaultMaskingLevel
		for _, field := range fieldList {
			if cmp.Less[storepb.MaskingLevel](finalLevel, field.maskingLevel) {
				finalLevel = field.maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return "", finalLevel, nil
			}
		}
		return "", finalLevel, nil
	case mysql.IColumnRefContext:
		databaseName, tableName, fieldName := parser.NormalizeMySQLFieldIdentifier(ctx.FieldIdentifier())
		level := extractor.mysqlCheckFieldMaskingLevel(databaseName, tableName, fieldName)
		return fieldName, level, nil
	}

	var list []antlr.ParserRuleContext
	for _, child := range ctx.GetChildren() {
		if child, ok := child.(antlr.ParserRuleContext); ok {
			list = append(list, child)
		}
	}

	return extractor.mysqlEvalMaskingLevelInExprList(list)
}

func (extractor *sensitiveFieldExtractor) mysqlEvalMaskingLevelInExprList(list []antlr.ParserRuleContext) (string, storepb.MaskingLevel, error) {
	finalLevel := defaultMaskingLevel
	var fieldName string
	var err error
	for _, ctx := range list {
		var level storepb.MaskingLevel
		fieldName, level, err = extractor.mysqlEvalMaskingLevelInExpr(ctx)
		if err != nil {
			return "", defaultMaskingLevel, err
		}
		if len(list) != 1 {
			fieldName = ""
		}
		if cmp.Less[storepb.MaskingLevel](finalLevel, level) {
			finalLevel = level
		}
		if finalLevel == maxMaskingLevel {
			return fieldName, finalLevel, nil
		}

	}
	return fieldName, finalLevel, nil
}

func (extractor *sensitiveFieldExtractor) mysqlCheckFieldMaskingLevel(databaseName string, tableName string, columnName string) storepb.MaskingLevel {
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
			sameDatabase = (strings.EqualFold(databaseName, field.database) ||
				(databaseName == "" && strings.EqualFold(field.database, extractor.currentDatabase))) ||
				(databaseName == "" && field.database == "")
			sameTable = (strings.EqualFold(tableName, field.table) || tableName == "")
		} else {
			sameDatabase = (databaseName == field.database ||
				(databaseName == "" && field.database == extractor.currentDatabase) ||
				(databaseName == "" && field.database == ""))
			sameTable = (tableName == field.table || tableName == "")
		}
		// Column name in MySQL is NOT case sensitive.
		sameField = strings.EqualFold(columnName, field.name)
		if sameDatabase && sameTable && sameField {
			return field.maskingLevel
		}
	}

	for _, field := range extractor.fromFieldList {
		var sameDatabase, sameTable, sameField bool
		if extractor.schemaInfo.IgnoreCaseSensitive {
			sameDatabase = (strings.EqualFold(databaseName, field.database) ||
				(databaseName == "" && strings.EqualFold(field.database, extractor.currentDatabase)) ||
				(databaseName == "" && field.database == ""))
			sameTable = (strings.EqualFold(tableName, field.table) || tableName == "")
		} else {
			sameDatabase = (databaseName == field.database ||
				(databaseName == "" && field.database == extractor.currentDatabase) ||
				(databaseName == "" && field.database == ""))
			sameTable = (tableName == field.table || tableName == "")
		}
		// Column name in MySQL is NOT case sensitive.
		sameField = strings.EqualFold(columnName, field.name)
		if sameDatabase && sameTable && sameField {
			return field.maskingLevel
		}
	}

	return defaultMaskingLevel
}
