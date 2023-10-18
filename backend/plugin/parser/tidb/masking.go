package tidb

import (
	"strings"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/pkg/errors"

	parser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_TIDB, GetMaskedFields)
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
	p := parser.New()
	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)
	nodeList, _, err := p.Parse(statement, "", "")
	if err != nil {
		return nil, err
	}
	if len(nodeList) != 1 {
		return nil, errors.Errorf("expect one statement but found %d", len(nodeList))
	}
	node := nodeList[0]

	switch node.(type) {
	case *tidbast.SelectStmt:
	case *tidbast.SetOprStmt:
	case *tidbast.CreateViewStmt:
	case *tidbast.ExplainStmt:
		// Skip the EXPLAIN statement.
		return nil, nil
	default:
		return nil, errors.Errorf("expect a query statement but found %T", node)
	}

	fieldList, err := extractor.extractNode(node)
	if err != nil {
		return nil, err
	}
	result := []base.SensitiveField{}
	for _, field := range fieldList {
		result = append(result, base.SensitiveField{
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}
	return result, nil
}

func (extractor *fieldExtractor) extractNode(in tidbast.Node) ([]base.FieldInfo, error) {
	if in == nil {
		return nil, nil
	}

	switch node := in.(type) {
	case *tidbast.SelectStmt:
		return extractor.extractSelect(node)
	case *tidbast.Join:
		return extractor.extractJoin(node)
	case *tidbast.TableSource:
		return extractor.extractTableSource(node)
	case *tidbast.TableName:
		return extractor.extractTableName(node)
	case *tidbast.SetOprStmt:
		return extractor.extractSetOpr(node)
	case *tidbast.CreateViewStmt:
		list, err := extractor.extractNode(node.Select)
		if err != nil {
			return nil, err
		}
		var result []base.FieldInfo
		if len(node.Cols) > 0 && len(node.Cols) != len(list) {
			return nil, errors.Errorf("The used SELECT statements have a different number of columns for view %s", node.ViewName.Name.O)
		}
		for i, item := range list {
			field := base.FieldInfo{
				Database: item.Database,
				Schema:   item.Schema,
				Table:    node.ViewName.Name.O,
				Name:     item.Name,
			}
			if len(node.Cols) > 0 {
				// The column name for MySQL is case insensitive.
				field.Name = node.Cols[i].L
			}
			result = append(result, field)
		}
		return result, nil
	}
	return nil, nil
}

func (extractor *fieldExtractor) extractSetOpr(node *tidbast.SetOprStmt) ([]base.FieldInfo, error) {
	if node.With != nil {
		cteOuterLength := len(extractor.cteOuterSchemaInfo)
		defer func() {
			extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:cteOuterLength]
		}()
		for _, cte := range node.With.CTEs {
			cteTable, err := extractor.extractCTE(cte)
			if err != nil {
				return nil, err
			}
			extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, cteTable)
		}
	}

	result := []base.FieldInfo{}
	for i, selectStmt := range node.SelectList.Selects {
		fieldList, err := extractor.extractNode(selectStmt)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			result = fieldList
		} else {
			if len(result) != len(fieldList) {
				// The error content comes from MySQL.
				return nil, errors.Errorf("The used SELECT statements have a different number of columns")
			}
			for i := 0; i < len(result); i++ {
				result[i].MaskingAttributes.TransmittedBy(fieldList[i].MaskingAttributes)
			}
		}
	}
	return result, nil
}

func splitInitialAndRecursivePart(node *tidbast.SetOprStmt, selfName string) ([]tidbast.Node, []tidbast.Node) {
	for i, selectStmt := range node.SelectList.Selects {
		tableList := ExtractMySQLTableList(selectStmt, false /* asName */)
		for _, table := range tableList {
			if table.Schema.O == "" && table.Name.O == selfName {
				return node.SelectList.Selects[:i], node.SelectList.Selects[i:]
			}
		}
	}
	return node.SelectList.Selects, nil
}

func (extractor *fieldExtractor) extractRecursiveCTE(node *tidbast.CommonTableExpression) (base.TableSchema, error) {
	cteInfo := base.TableSchema{Name: node.Name.O}

	switch x := node.Query.Query.(type) {
	case *tidbast.SetOprStmt:
		if x.With != nil {
			cteLength := len(extractor.cteOuterSchemaInfo)
			defer func() {
				extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:cteLength]
			}()
			for _, cte := range x.With.CTEs {
				cteTable, err := extractor.extractCTE(cte)
				if err != nil {
					return base.TableSchema{}, err
				}
				extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, cteTable)
			}

			for i := cteLength; i < len(extractor.cteOuterSchemaInfo); i++ {
				cteTable := extractor.cteOuterSchemaInfo[i]
				if cteTable.Name == node.Name.O {
					// It means this recursive CTE will be hidden by the inner CTE with the same name.
					// In other words, this recursive CTE will be not references by itself sub-query.
					// So, we can build it as non-recursive CTE.
					return extractor.extractNonRecursiveCTE(node)
				}
			}
		}

		initialPart, recursivePart := splitInitialAndRecursivePart(x, node.Name.O)
		if len(initialPart) == 0 {
			return base.TableSchema{}, errors.Errorf("Failed to find initial part for recursive common table expression")
		}
		if len(recursivePart) == 0 {
			return extractor.extractNonRecursiveCTE(node)
		}

		initialField, err := extractor.extractNode(&tidbast.SetOprStmt{
			SelectList: &tidbast.SetOprSelectList{
				Selects: initialPart,
			},
		})
		if err != nil {
			return base.TableSchema{}, err
		}

		if len(node.ColNameList) > 0 {
			if len(node.ColNameList) != len(initialField) {
				// The error content comes from MySQL.
				return base.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
			}
			for i := 0; i < len(initialField); i++ {
				initialField[i].Name = node.ColNameList[i].O
			}
		}
		for _, field := range initialField {
			cteInfo.ColumnList = append(cteInfo.ColumnList, base.ColumnInfo{
				Name:              field.Name,
				MaskingAttributes: field.MaskingAttributes,
			})
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
		extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, cteInfo)
		defer func() {
			extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:len(extractor.cteOuterSchemaInfo)-1]
		}()
		for {
			fieldList, err := extractor.extractNode(&tidbast.SetOprStmt{
				SelectList: &tidbast.SetOprSelectList{
					Selects: recursivePart,
				},
			})
			if err != nil {
				return base.TableSchema{}, err
			}
			if len(fieldList) != len(cteInfo.ColumnList) {
				// The error content comes from MySQL.
				return base.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
			}

			changed := false
			for i, field := range fieldList {
				changed = changed || cteInfo.ColumnList[i].MaskingAttributes.TransmittedBy(field.MaskingAttributes)
			}

			if !changed {
				break
			}
			extractor.cteOuterSchemaInfo[len(extractor.cteOuterSchemaInfo)-1] = cteInfo
		}
		return cteInfo, nil
	default:
		return extractor.extractNonRecursiveCTE(node)
	}
}

func (extractor *fieldExtractor) extractNonRecursiveCTE(node *tidbast.CommonTableExpression) (base.TableSchema, error) {
	fieldList, err := extractor.extractNode(node.Query.Query)
	if err != nil {
		return base.TableSchema{}, err
	}
	if len(node.ColNameList) > 0 {
		if len(node.ColNameList) != len(fieldList) {
			// The error content comes from MySQL.
			return base.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
		}
		for i := 0; i < len(fieldList); i++ {
			fieldList[i].Name = node.ColNameList[i].O
		}
	}
	result := base.TableSchema{
		Name:       node.Name.O,
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

func (extractor *fieldExtractor) extractCTE(node *tidbast.CommonTableExpression) (base.TableSchema, error) {
	if node.IsRecursive {
		return extractor.extractRecursiveCTE(node)
	}
	return extractor.extractNonRecursiveCTE(node)
}

func (extractor *fieldExtractor) extractSelect(node *tidbast.SelectStmt) ([]base.FieldInfo, error) {
	if node.With != nil {
		cteOuterLength := len(extractor.cteOuterSchemaInfo)
		defer func() {
			extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:cteOuterLength]
		}()
		for _, cte := range node.With.CTEs {
			cteTable, err := extractor.extractCTE(cte)
			if err != nil {
				return nil, err
			}
			extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, cteTable)
		}
	}

	var fromFieldList []base.FieldInfo
	var err error
	if node.From != nil {
		fromFieldList, err = extractor.extractNode(node.From.TableRefs)
		if err != nil {
			return nil, err
		}
		extractor.fromFieldList = fromFieldList
		defer func() {
			extractor.fromFieldList = nil
		}()
	}

	var result []base.FieldInfo

	if node.Fields != nil {
		for _, field := range node.Fields.Fields {
			if field.WildCard != nil {
				if field.WildCard.Table.O == "" {
					result = append(result, fromFieldList...)
				} else {
					for _, fromField := range fromFieldList {
						sameDatabase := (field.WildCard.Schema.O == fromField.Database || (field.WildCard.Schema.O == "" && fromField.Database == extractor.currentDatabase))
						sameTable := (field.WildCard.Table.O == fromField.Table)
						if sameDatabase && sameTable {
							result = append(result, fromField)
						}
					}
				}
			} else {
				maskingLevel, err := extractor.extractColumnFromExprNode(field.Expr)
				if err != nil {
					return nil, err
				}
				fieldName := extractFieldName(field)
				result = append(result, base.FieldInfo{
					Database:          "",
					Table:             "",
					Name:              fieldName,
					MaskingAttributes: maskingLevel,
				})
			}
		}
	}

	return result, nil
}

func extractFieldName(in *tidbast.SelectField) string {
	if in.AsName.O != "" {
		return in.AsName.O
	}

	if in.Expr != nil {
		if columnName, ok := in.Expr.(*tidbast.ColumnNameExpr); ok {
			return columnName.Name.Name.O
		}
		return in.Text()
	}
	return ""
}

func (extractor *fieldExtractor) checkFieldMaskingLevel(databaseName string, tableName string, fieldName string) base.MaskingAttributes {
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
		sameDatabase := (databaseName == field.Database || (databaseName == "" && field.Database == extractor.currentDatabase))
		sameTable := (tableName == field.Table || tableName == "")
		sameField := (fieldName == field.Name)
		if sameDatabase && sameTable && sameField {
			return field.MaskingAttributes
		}
	}

	for _, field := range extractor.fromFieldList {
		sameDatabase := (databaseName == field.Database || (databaseName == "" && field.Database == extractor.currentDatabase))
		sameTable := (tableName == field.Table || tableName == "")
		sameField := (fieldName == field.Name)
		if sameDatabase && sameTable && sameField {
			return field.MaskingAttributes
		}
	}

	return base.NewDefaultMaskingAttributes()
}

func (extractor *fieldExtractor) extractColumnFromExprNode(in tidbast.ExprNode) (base.MaskingAttributes, error) {
	if in == nil {
		return base.NewDefaultMaskingAttributes(), nil
	}

	switch node := in.(type) {
	case *tidbast.ColumnNameExpr:
		return extractor.checkFieldMaskingLevel(node.Name.Schema.O, node.Name.Table.O, node.Name.Name.O), nil
	case *tidbast.BinaryOperationExpr:
		return extractor.extractColumnFromExprNodeList([]tidbast.ExprNode{node.L, node.R})
	case *tidbast.UnaryOperationExpr:
		return extractor.extractColumnFromExprNode(node.V)
	case *tidbast.FuncCallExpr:
		return extractor.extractColumnFromExprNodeList(node.Args)
	case *tidbast.FuncCastExpr:
		return extractor.extractColumnFromExprNode(node.Expr)
	case *tidbast.AggregateFuncExpr:
		return extractor.extractColumnFromExprNodeList(node.Args)
	case *tidbast.PatternInExpr:
		nodeList := []tidbast.ExprNode{}
		nodeList = append(nodeList, node.Expr)
		nodeList = append(nodeList, node.List...)
		nodeList = append(nodeList, node.Sel)
		return extractor.extractColumnFromExprNodeList(nodeList)
	case *tidbast.PatternLikeExpr:
		return extractor.extractColumnFromExprNodeList([]tidbast.ExprNode{node.Expr, node.Pattern})
	case *tidbast.PatternRegexpExpr:
		return extractor.extractColumnFromExprNodeList([]tidbast.ExprNode{node.Expr, node.Pattern})
	case *tidbast.SubqueryExpr:
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
		fieldList, err := subqueryExtractor.extractNode(node.Query)
		if err != nil {
			return base.NewDefaultMaskingAttributes(), err
		}
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, field := range fieldList {
			finalAttributes.TransmittedBy(field.MaskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return finalAttributes, nil
			}
		}
		return finalAttributes, nil
	case *tidbast.CompareSubqueryExpr:
		return extractor.extractColumnFromExprNodeList([]tidbast.ExprNode{node.L, node.R})
	case *tidbast.ExistsSubqueryExpr:
		return extractor.extractColumnFromExprNode(node.Sel)
	case *tidbast.IsNullExpr:
		return extractor.extractColumnFromExprNode(node.Expr)
	case *tidbast.IsTruthExpr:
		return extractor.extractColumnFromExprNode(node.Expr)
	case *tidbast.BetweenExpr:
		return extractor.extractColumnFromExprNodeList([]tidbast.ExprNode{node.Expr, node.Left, node.Right})
	case *tidbast.CaseExpr:
		nodeList := []tidbast.ExprNode{}
		nodeList = append(nodeList, node.Value)
		nodeList = append(nodeList, node.ElseClause)
		for _, whenClause := range node.WhenClauses {
			nodeList = append(nodeList, whenClause.Expr)
			nodeList = append(nodeList, whenClause.Result)
		}
		return extractor.extractColumnFromExprNodeList(nodeList)
	case *tidbast.ParenthesesExpr:
		return extractor.extractColumnFromExprNode(node.Expr)
	case *tidbast.RowExpr:
		return extractor.extractColumnFromExprNodeList(node.Values)
	case *tidbast.VariableExpr:
		return extractor.extractColumnFromExprNode(node.Value)
	case *tidbast.PositionExpr:
		return extractor.extractColumnFromExprNode(node.P)
	case *tidbast.MatchAgainst:
		return extractor.extractColumnFromExprNode(node.Against)
	case *tidbast.WindowFuncExpr:
		return extractor.extractColumnFromExprNodeList(node.Args)
	case *tidbast.ValuesExpr,
		*tidbast.TableNameExpr,
		*tidbast.MaxValueExpr,
		*tidbast.SetCollationExpr,
		*tidbast.TrimDirectionExpr,
		*tidbast.TimeUnitExpr,
		*tidbast.GetFormatSelectorExpr,
		*tidbast.DefaultExpr:
		// No expression need to extract.
	}
	return base.NewDefaultMaskingAttributes(), nil
}

func (extractor *fieldExtractor) extractColumnFromExprNodeList(nodeList []tidbast.ExprNode) (base.MaskingAttributes, error) {
	finalAttributes := base.NewDefaultMaskingAttributes()
	for _, node := range nodeList {
		maskingAttributes, err := extractor.extractColumnFromExprNode(node)
		if err != nil {
			return base.NewDefaultMaskingAttributes(), err
		}
		finalAttributes.TransmittedBy(maskingAttributes)
		if finalAttributes.IsNeverChangeInTransmission() {
			return finalAttributes, nil
		}
	}
	return finalAttributes, nil
}

func (extractor *fieldExtractor) extractTableSource(node *tidbast.TableSource) ([]base.FieldInfo, error) {
	fieldList, err := extractor.extractNode(node.Source)
	if err != nil {
		return nil, err
	}
	var res []base.FieldInfo
	if node.AsName.O != "" {
		for _, field := range fieldList {
			res = append(res, base.FieldInfo{
				Name:              field.Name,
				Table:             node.AsName.O,
				Database:          field.Database,
				MaskingAttributes: field.MaskingAttributes,
			})
		}
	} else {
		res = fieldList
	}
	return res, nil
}

func (extractor *fieldExtractor) findTableSchema(databaseName string, tableName string) (string, base.TableSchema, error) {
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

	database, schema, err := extractor.findViewSchema(databaseName, tableName)
	if err == nil {
		return database, schema, nil
	}
	return "", base.TableSchema{}, errors.Wrapf(err, "Table or view %q.%q not found", databaseName, tableName)
}

func (extractor *fieldExtractor) buildTableSchemaForView(viewName string, definition string) (base.TableSchema, error) {
	newExtractor := &fieldExtractor{
		currentDatabase: extractor.currentDatabase,
		schemaInfo:      extractor.schemaInfo,
	}
	fields, err := newExtractor.extractSensitiveFields(definition)
	if err != nil {
		return base.TableSchema{}, err
	}

	result := base.TableSchema{
		Name:       viewName,
		ColumnList: []base.ColumnInfo{},
	}
	for _, field := range fields {
		// nolint:gosimple
		result.ColumnList = append(result.ColumnList, base.ColumnInfo{
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}
	return result, nil
}

func (extractor *fieldExtractor) findViewSchema(databaseName string, viewName string) (string, base.TableSchema, error) {
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

						table, err := extractor.buildTableSchemaForView(view.Name, view.Definition)
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

					table, err := extractor.buildTableSchemaForView(view.Name, view.Definition)
					return explicitDatabase, table, err
				}
			}
		}
	}
	return "", base.TableSchema{}, errors.Errorf("View %q.%q not found", databaseName, viewName)
}

func (extractor *fieldExtractor) extractTableName(node *tidbast.TableName) ([]base.FieldInfo, error) {
	databaseName, tableSchema, err := extractor.findTableSchema(node.Schema.O, node.Name.O)
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

func (extractor *fieldExtractor) extractJoin(node *tidbast.Join) ([]base.FieldInfo, error) {
	if node.Right == nil {
		// This case is not Join
		return extractor.extractNode(node.Left)
	}
	leftFieldInfo, err := extractor.extractNode(node.Left)
	if err != nil {
		return nil, err
	}
	rightFieldInfo, err := extractor.extractNode(node.Right)
	if err != nil {
		return nil, err
	}
	return mergeJoinField(node, leftFieldInfo, rightFieldInfo)
}

func mergeJoinField(node *tidbast.Join, leftField []base.FieldInfo, rightField []base.FieldInfo) ([]base.FieldInfo, error) {
	leftFieldMap := make(map[string]base.FieldInfo)
	rightFieldMap := make(map[string]base.FieldInfo)
	var result []base.FieldInfo
	for _, field := range leftField {
		// Column name in MySQL is NOT case-sensitive.
		leftFieldMap[strings.ToLower(field.Name)] = field
	}
	for _, field := range rightField {
		// Column name in MySQL is NOT case-sensitive.
		rightFieldMap[strings.ToLower(field.Name)] = field
	}
	if node.NaturalJoin {
		// Natural Join will merge the same column name field.
		for _, field := range leftField {
			// Merge the sensitive attribute for the same column name field.
			if rField, exists := rightFieldMap[strings.ToLower(field.Name)]; exists {
				field.MaskingAttributes.TransmittedBy(rField.MaskingAttributes)
			}
			result = append(result, field)
		}

		for _, field := range rightField {
			if _, exists := leftFieldMap[strings.ToLower(field.Name)]; !exists {
				result = append(result, field)
			}
		}
	} else {
		if len(node.Using) != 0 {
			// ... JOIN ... USING (...) will merge the column in USING.
			usingMap := make(map[string]bool)
			for _, column := range node.Using {
				// Column name in MySQL is NOT case-sensitive.
				usingMap[column.Name.L] = true
			}

			for _, field := range leftField {
				_, existsInUsingMap := usingMap[strings.ToLower(field.Name)]
				rField, existsInRightField := rightFieldMap[strings.ToLower(field.Name)]
				// Merge the sensitive attribute for the column name field in USING.
				if existsInUsingMap && existsInRightField {
					field.MaskingAttributes.TransmittedBy(rField.MaskingAttributes)
				}
				result = append(result, field)
			}

			for _, field := range rightField {
				_, existsInUsingMap := usingMap[strings.ToLower(field.Name)]
				_, existsInLeftField := leftFieldMap[strings.ToLower(field.Name)]
				if existsInUsingMap && existsInLeftField {
					continue
				}
				result = append(result, field)
			}
		} else {
			result = append(result, leftField...)
			result = append(result, rightField...)
		}
	}

	return result, nil
}
