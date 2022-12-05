package util

import (
	"github.com/bytebase/bytebase/plugin/db"

	"github.com/pkg/errors"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
)

type sensitiveFieldExtractor struct {
	currentDatabase string
	schemaInfo      *db.SensitiveSchemaInfo

	// SELECT statement specific field.
	fromFieldList []fieldInfo
}

func extractSensitiveField(dbType db.Type, statement string, currentDatabase string, schemaInfo *db.SensitiveSchemaInfo) ([]db.SensitiveField, error) {
	if schemaInfo == nil {
		return nil, nil
	}

	switch dbType {
	case db.MySQL, db.TiDB:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		return extractor.extractMySQLSensitiveField(statement)
	default:
		return nil, nil
	}
}

func (extractor *sensitiveFieldExtractor) extractMySQLSensitiveField(statement string) ([]db.SensitiveField, error) {
	p := tidbparser.New()

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
	default:
		return nil, errors.Errorf("expect a query statement but found %T", node)
	}

	fieldList, err := extractor.extractNode(node)
	if err != nil {
		return nil, err
	}
	result := []db.SensitiveField{}
	for _, field := range fieldList {
		result = append(result, db.SensitiveField{
			Name:      field.name,
			Sensitive: field.sensitive,
		})
	}
	return result, nil
}

type fieldInfo struct {
	name      string
	table     string
	database  string
	sensitive bool
}

// TODO(rebelice): support Common Table Expression later.
func (extractor *sensitiveFieldExtractor) extractNode(in tidbast.Node) ([]fieldInfo, error) {
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
	}
	return nil, nil
}

func (extractor *sensitiveFieldExtractor) extractSetOpr(node *tidbast.SetOprStmt) ([]fieldInfo, error) {
	result := []fieldInfo{}
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
			for index := 0; index < len(result); index++ {
				if fieldList[index].sensitive {
					result[index].sensitive = true
				}
			}
		}
	}
	return result, nil
}

func (extractor *sensitiveFieldExtractor) extractSelect(node *tidbast.SelectStmt) ([]fieldInfo, error) {
	var fromFieldList []fieldInfo
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

	var result []fieldInfo

	if node.Fields != nil {
		for _, field := range node.Fields.Fields {
			if field.WildCard != nil {
				if field.WildCard.Table.O == "" {
					result = append(result, fromFieldList...)
				} else {
					for _, fromField := range fromFieldList {
						sameDatabase := (field.WildCard.Schema.O == fromField.database || (field.WildCard.Schema.O == "" && fromField.database == extractor.currentDatabase))
						sameTable := (field.WildCard.Table.O == fromField.table)
						if sameDatabase && sameTable {
							result = append(result, fromField)
						}
					}
				}
			} else {
				sensitive, err := extractor.extractColumnFromExprNode(field.Expr)
				if err != nil {
					return nil, err
				}
				fieldName := extractFieldName(field)
				result = append(result, fieldInfo{
					database:  "",
					table:     "",
					name:      fieldName,
					sensitive: sensitive,
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

func (extractor *sensitiveFieldExtractor) extractColumnFromExprNode(in tidbast.ExprNode) (sensitive bool, err error) {
	if in == nil {
		return false, nil
	}

	switch node := in.(type) {
	case *tidbast.ColumnNameExpr:
		for _, field := range extractor.fromFieldList {
			sameDatabase := (node.Name.Schema.O == field.database || (node.Name.Schema.O == "" && field.database == extractor.currentDatabase))
			sameTable := (node.Name.Table.O == field.table || node.Name.Table.O == "")
			sameColumn := (node.Name.Name.O == field.name)
			if sameDatabase && sameTable && sameColumn {
				return field.sensitive, nil
			}
		}
		return false, nil
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
		// It can only be the non-associated subquery.
		// We can extract this subquery as the alone node.
		// The reason for new extractor is that non-associated subquery cannot access the fromFieldList.
		// Also, we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &sensitiveFieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
		}
		fieldList, err := subqueryExtractor.extractNode(node.Query)
		if err != nil {
			return false, err
		}
		for _, field := range fieldList {
			if field.sensitive {
				return true, nil
			}
		}
		return false, nil
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
	return false, nil
}

func (extractor *sensitiveFieldExtractor) extractColumnFromExprNodeList(nodeList []tidbast.ExprNode) (sensitive bool, err error) {
	for _, node := range nodeList {
		nodeSensitive, err := extractor.extractColumnFromExprNode(node)
		if err != nil {
			return false, err
		}
		if nodeSensitive {
			return true, nil
		}
	}
	return false, nil
}

func (extractor *sensitiveFieldExtractor) extractTableSource(node *tidbast.TableSource) ([]fieldInfo, error) {
	fieldList, err := extractor.extractNode(node.Source)
	if err != nil {
		return nil, err
	}
	var res []fieldInfo
	if node.AsName.O != "" {
		for _, field := range fieldList {
			res = append(res, fieldInfo{
				name:      field.name,
				table:     node.AsName.O,
				database:  field.database,
				sensitive: field.sensitive,
			})
		}
	} else {
		res = fieldList
	}
	return res, nil
}

func (extractor *sensitiveFieldExtractor) findTableSchema(databaseName string, tableName string) (db.TableSchema, bool) {
	for _, database := range extractor.schemaInfo.DatabaseList {
		if databaseName == database.Name || (databaseName == "" && extractor.currentDatabase == database.Name) {
			for _, table := range database.TableList {
				if tableName == table.Name {
					return table, true
				}
			}
		}
	}
	return db.TableSchema{}, false
}

func (extractor *sensitiveFieldExtractor) extractTableName(node *tidbast.TableName) ([]fieldInfo, error) {
	tableSchema, exists := extractor.findTableSchema(node.Schema.O, node.Name.O)
	if !exists {
		return nil, errors.Errorf("Table %q.%q not found", node.Schema.O, node.Name.O)
	}

	var res []fieldInfo
	for _, column := range tableSchema.ColumnList {
		res = append(res, fieldInfo{
			name:      column.Name,
			table:     tableSchema.Name,
			database:  node.Schema.O,
			sensitive: column.Sensitive,
		})
	}
	return res, nil
}

func (extractor *sensitiveFieldExtractor) extractJoin(node *tidbast.Join) ([]fieldInfo, error) {
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

func mergeJoinField(node *tidbast.Join, leftField []fieldInfo, rightField []fieldInfo) ([]fieldInfo, error) {
	leftFieldMap := make(map[string]fieldInfo)
	rightFieldMap := make(map[string]fieldInfo)
	var result []fieldInfo
	for _, field := range leftField {
		leftFieldMap[field.name] = field
	}
	for _, field := range rightField {
		rightFieldMap[field.name] = field
	}
	if node.NaturalJoin {
		// Natural Join will merge the same column name field.
		for _, field := range leftField {
			// Merge the sensitive attribute for the same column name field.
			if rField, exists := rightFieldMap[field.name]; exists && rField.sensitive {
				field.sensitive = true
			}
			result = append(result, field)
		}

		for _, field := range rightField {
			if _, exists := leftFieldMap[field.name]; !exists {
				result = append(result, field)
			}
		}
	} else {
		if len(node.Using) != 0 {
			// ... JOIN ... USING (...) will merge the column in USING.
			usingMap := make(map[string]bool)
			for _, column := range node.Using {
				usingMap[column.Name.O] = true
			}

			for _, field := range leftField {
				_, existsInUsingMap := usingMap[field.name]
				rField, existsInRightField := rightFieldMap[field.name]
				// Merge the sensitive attribute for the column name field in USING.
				if existsInUsingMap && existsInRightField && rField.sensitive {
					field.sensitive = true
				}
				result = append(result, field)
			}

			for _, field := range rightField {
				_, existsInUsingMap := usingMap[field.name]
				_, existsInLeftField := leftFieldMap[field.name]
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
