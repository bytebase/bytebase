package util

import (
	"fmt"
	"regexp"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v4"

	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"

	"github.com/pkg/errors"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
)

const (
	pgUnknownFieldName = "?column?"
)

type sensitiveFieldExtractor struct {
	// For Oracle, we need to know the current database to determine if the table is in the current schema.
	currentDatabase    string
	schemaInfo         *db.SensitiveSchemaInfo
	outerSchemaInfo    []fieldInfo
	cteOuterSchemaInfo []db.TableSchema

	// SELECT statement specific field.
	fromFieldList []fieldInfo
}

func extractSensitiveField(dbType db.Type, statement string, currentDatabase string, schemaInfo *db.SensitiveSchemaInfo) ([]db.SensitiveField, error) {
	if schemaInfo == nil {
		return nil, nil
	}

	switch dbType {
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		return extractor.extractMySQLSensitiveField(statement)
	case db.Postgres, db.Redshift, db.RisingWave:
		extractor := &sensitiveFieldExtractor{
			schemaInfo: schemaInfo,
		}
		result, err := extractor.extractPostgreSQLSensitiveField(statement)
		if err != nil {
			tableNotFound := regexp.MustCompile("^Table \"(.*)\\.(.*)\" not found$")
			content := tableNotFound.FindStringSubmatch(err.Error())
			if len(content) == 3 && (isPostgreSQLSystemSchema(content[1]) || dbType == db.RisingWave && isRisingWaveSystemSchema(content[1])) {
				// skip for system schema
				return nil, nil
			}
			return nil, err
		}
		return result, nil
	case db.Oracle, db.DM:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		return extractor.extractOracleSensitiveField(statement)
	case db.Snowflake:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		return extractor.extractSnowsqlSensitiveFields(statement)
	case db.MSSQL:
		extractor := &sensitiveFieldExtractor{
			currentDatabase: currentDatabase,
			schemaInfo:      schemaInfo,
		}
		return extractor.extractTSqlSensitiveFields(statement)
	default:
		return nil, nil
	}
}

func isPostgreSQLSystemSchema(schema string) bool {
	switch schema {
	case "information_schema", "pg_catalog":
		return true
	}
	return false
}

func isRisingWaveSystemSchema(schema string) bool {
	switch schema {
	case "information_schema", "pg_catalog", "rw_catalog":
		return true
	}
	return false
}

func (extractor *sensitiveFieldExtractor) extractPostgreSQLSensitiveField(statement string) ([]db.SensitiveField, error) {
	res, err := pgquery.Parse(statement)
	if err != nil {
		return nil, err
	}
	if len(res.Stmts) != 1 {
		return nil, errors.Errorf("expect one statement but found %d", len(res.Stmts))
	}
	node := res.Stmts[0]

	switch node.Stmt.Node.(type) {
	case *pgquery.Node_SelectStmt:
	case *pgquery.Node_ExplainStmt:
		// Skip the EXPLAIN statement.
		return nil, nil
	default:
		return nil, errors.Errorf("expect a query statement but found %T", node.Stmt.Node)
	}

	fieldList, err := extractor.pgExtractNode(node.Stmt)
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

func (extractor *sensitiveFieldExtractor) pgExtractNode(in *pgquery.Node) ([]fieldInfo, error) {
	if in == nil {
		return nil, nil
	}

	switch node := in.Node.(type) {
	case *pgquery.Node_SelectStmt:
		return extractor.pgExtractSelect(node)
	case *pgquery.Node_RangeVar:
		return extractor.pgExtractRangeVar(node)
	case *pgquery.Node_RangeSubselect:
		return extractor.pgExtractRangeSubselect(node)
	case *pgquery.Node_JoinExpr:
		return extractor.pgExtractJoin(node)
	}
	return nil, nil
}

func (extractor *sensitiveFieldExtractor) pgExtractJoin(in *pgquery.Node_JoinExpr) ([]fieldInfo, error) {
	leftFieldInfo, err := extractor.pgExtractNode(in.JoinExpr.Larg)
	if err != nil {
		return nil, err
	}
	rightFieldInfo, err := extractor.pgExtractNode(in.JoinExpr.Rarg)
	if err != nil {
		return nil, err
	}
	return pgMergeJoinField(in, leftFieldInfo, rightFieldInfo)
}

func pgMergeJoinField(node *pgquery.Node_JoinExpr, leftField []fieldInfo, rightField []fieldInfo) ([]fieldInfo, error) {
	leftFieldMap := make(map[string]fieldInfo)
	rightFieldMap := make(map[string]fieldInfo)
	var result []fieldInfo
	for _, field := range leftField {
		leftFieldMap[field.name] = field
	}
	for _, field := range rightField {
		rightFieldMap[field.name] = field
	}
	if node.JoinExpr.IsNatural {
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
		if len(node.JoinExpr.UsingClause) > 0 {
			// ... JOIN ... USING (...) will merge the column in USING.
			var usingList []string
			for _, nameNode := range node.JoinExpr.UsingClause {
				name, yes := nameNode.Node.(*pgquery.Node_String_)
				if !yes {
					return nil, errors.Errorf("expect Node_String_ but found %T", nameNode.Node)
				}
				usingList = append(usingList, name.String_.Sval)
			}
			usingMap := make(map[string]bool)
			for _, column := range usingList {
				usingMap[column] = true
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

func (extractor *sensitiveFieldExtractor) pgExtractRangeSubselect(node *pgquery.Node_RangeSubselect) ([]fieldInfo, error) {
	fieldList, err := extractor.pgExtractNode(node.RangeSubselect.Subquery)
	if err != nil {
		return nil, err
	}
	if node.RangeSubselect.Alias != nil {
		var result []fieldInfo
		aliasName, columnNameList, err := pgExtractAlias(node.RangeSubselect.Alias)
		if err != nil {
			return nil, err
		}
		if len(columnNameList) != 0 && len(columnNameList) != len(fieldList) {
			return nil, errors.Errorf("expect equal length but found %d and %d", len(columnNameList), len(fieldList))
		}
		for i, item := range fieldList {
			columnName := item.name
			if len(columnNameList) > 0 {
				columnName = columnNameList[i]
			}
			result = append(result, fieldInfo{
				table:     fmt.Sprintf("public.%s", aliasName),
				name:      columnName,
				sensitive: item.sensitive,
			})
		}
		return result, nil
	}
	return fieldList, nil
}

func pgNormalizeTableName(schemaName string, tableName string) string {
	if tableName == "" {
		return ""
	}
	if schemaName == "" {
		schemaName = "public"
	}
	return fmt.Sprintf("%s.%s", schemaName, tableName)
}

func pgExtractAlias(alias *pgquery.Alias) (string, []string, error) {
	if alias == nil {
		return "", nil, nil
	}
	var columnNameList []string
	for _, item := range alias.Colnames {
		stringNode, yes := item.Node.(*pgquery.Node_String_)
		if !yes {
			return "", nil, errors.Errorf("expect Node_String_ but found %T", item.Node)
		}
		columnNameList = append(columnNameList, stringNode.String_.Sval)
	}
	return alias.Aliasname, columnNameList, nil
}

func (extractor *sensitiveFieldExtractor) pgExtractRangeVar(node *pgquery.Node_RangeVar) ([]fieldInfo, error) {
	tableSchema, err := extractor.pgFindTableSchema(pgNormalizeTableName(node.RangeVar.Schemaname, node.RangeVar.Relname))
	if err != nil {
		return nil, err
	}

	var res []fieldInfo
	if node.RangeVar.Alias == nil {
		for _, column := range tableSchema.ColumnList {
			res = append(res, fieldInfo{
				name:      column.Name,
				table:     tableSchema.Name,
				sensitive: column.Sensitive,
			})
		}
	} else {
		aliasName, columnNameList, err := pgExtractAlias(node.RangeVar.Alias)
		if err != nil {
			return nil, err
		}
		if len(columnNameList) != 0 && len(columnNameList) != len(tableSchema.ColumnList) {
			return nil, errors.Errorf("expect equal length but found %d and %d", len(node.RangeVar.Alias.Colnames), len(tableSchema.ColumnList))
		}

		for i, column := range tableSchema.ColumnList {
			tableName := fmt.Sprintf("public.%s", aliasName)
			columnName := column.Name
			if len(columnNameList) > 0 {
				columnName = columnNameList[i]
			}
			res = append(res, fieldInfo{
				name:      columnName,
				table:     tableName,
				sensitive: column.Sensitive,
			})
		}
	}

	return res, nil
}

func (extractor *sensitiveFieldExtractor) pgFindTableSchema(tableName string) (db.TableSchema, error) {
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
		if table.Name == tableName {
			return table, nil
		}
	}

	for _, database := range extractor.schemaInfo.DatabaseList {
		for _, table := range database.TableList {
			if tableName == table.Name {
				return table, nil
			}
		}
	}
	return db.TableSchema{}, errors.Errorf("Table %q not found", tableName)
}

func (extractor *sensitiveFieldExtractor) pgExtractRecursiveCTE(node *pgquery.Node_CommonTableExpr) (db.TableSchema, error) {
	switch selectNode := node.CommonTableExpr.Ctequery.Node.(type) {
	case *pgquery.Node_SelectStmt:
		if selectNode.SelectStmt.Op != pgquery.SetOperation_SETOP_UNION {
			return extractor.pgExtractNonRecursiveCTE(node)
		}
		// For PostgreSQL, recursive CTE will be an UNION statement, and the left node is the initial part,
		// the right node is the recursive part.
		initialField, err := extractor.pgExtractSelect(&pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Larg})
		if err != nil {
			return db.TableSchema{}, err
		}
		if len(node.CommonTableExpr.Aliascolnames) > 0 {
			if len(node.CommonTableExpr.Aliascolnames) != len(initialField) {
				return db.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
			}
			for i, nameNode := range node.CommonTableExpr.Aliascolnames {
				stringNode, yes := nameNode.Node.(*pgquery.Node_String_)
				if !yes {
					return db.TableSchema{}, errors.Errorf("expect Node_String_ but found %T", nameNode.Node)
				}
				initialField[i].name = stringNode.String_.Sval
			}
		}

		cteInfo := db.TableSchema{Name: pgNormalizeTableName("public", node.CommonTableExpr.Ctename)}
		for _, field := range initialField {
			cteInfo.ColumnList = append(cteInfo.ColumnList, db.ColumnInfo{
				Name:      field.name,
				Sensitive: field.sensitive,
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
			fieldList, err := extractor.pgExtractSelect(&pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Rarg})
			if err != nil {
				return db.TableSchema{}, err
			}
			if len(fieldList) != len(cteInfo.ColumnList) {
				return db.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
			}

			changed := false
			for i, field := range fieldList {
				if field.sensitive && !cteInfo.ColumnList[i].Sensitive {
					changed = true
					cteInfo.ColumnList[i].Sensitive = true
				}
			}

			if !changed {
				break
			}
			extractor.cteOuterSchemaInfo[len(extractor.cteOuterSchemaInfo)-1] = cteInfo
		}
		return cteInfo, nil
	default:
		return extractor.pgExtractNonRecursiveCTE(node)
	}
}

func (extractor *sensitiveFieldExtractor) pgExtractNonRecursiveCTE(node *pgquery.Node_CommonTableExpr) (db.TableSchema, error) {
	fieldList, err := extractor.pgExtractNode(node.CommonTableExpr.Ctequery)
	if err != nil {
		return db.TableSchema{}, err
	}
	if len(node.CommonTableExpr.Aliascolnames) > 0 {
		if len(node.CommonTableExpr.Aliascolnames) != len(fieldList) {
			return db.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
		}
		var nameList []string
		for _, nameNode := range node.CommonTableExpr.Aliascolnames {
			stringNode, yes := nameNode.Node.(*pgquery.Node_String_)
			if !yes {
				return db.TableSchema{}, errors.Errorf("expect Node_String_ but found %T", nameNode.Node)
			}
			nameList = append(nameList, stringNode.String_.Sval)
		}
		for i := 0; i < len(fieldList); i++ {
			fieldList[i].name = nameList[i]
		}
	}
	result := db.TableSchema{
		Name:       pgNormalizeTableName("public", node.CommonTableExpr.Ctename),
		ColumnList: []db.ColumnInfo{},
	}

	for _, field := range fieldList {
		result.ColumnList = append(result.ColumnList, db.ColumnInfo{
			Name:      field.name,
			Sensitive: field.sensitive,
		})
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) pgExtractSelect(node *pgquery.Node_SelectStmt) ([]fieldInfo, error) {
	if node.SelectStmt.WithClause != nil {
		cteOuterLength := len(extractor.cteOuterSchemaInfo)
		defer func() {
			extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:cteOuterLength]
		}()
		for _, cte := range node.SelectStmt.WithClause.Ctes {
			in, yes := cte.Node.(*pgquery.Node_CommonTableExpr)
			if !yes {
				return nil, errors.Errorf("expect CommonTableExpr but found %T", cte.Node)
			}
			var cteTable db.TableSchema
			var err error
			if node.SelectStmt.WithClause.Recursive {
				cteTable, err = extractor.pgExtractRecursiveCTE(in)
			} else {
				cteTable, err = extractor.pgExtractNonRecursiveCTE(in)
			}
			if err != nil {
				return nil, err
			}
			extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, cteTable)
		}
	}

	// The VALUES case.
	if len(node.SelectStmt.ValuesLists) > 0 {
		var result []fieldInfo
		for _, row := range node.SelectStmt.ValuesLists {
			var sensitiveList []bool
			list, yes := row.Node.(*pgquery.Node_List)
			if !yes {
				return nil, errors.Errorf("expect Node_List but found %T", row.Node)
			}
			for _, item := range list.List.Items {
				sensitive, err := extractor.pgExtractColumnRefFromExpressionNode(item)
				if err != nil {
					return nil, err
				}
				sensitiveList = append(sensitiveList, sensitive)
			}
			if len(result) == 0 {
				for i, item := range sensitiveList {
					result = append(result, fieldInfo{
						name:      fmt.Sprintf("column%d", i+1),
						sensitive: item,
					})
				}
			}
		}
		return result, nil
	}

	switch node.SelectStmt.Op {
	case pgquery.SetOperation_SETOP_UNION, pgquery.SetOperation_SETOP_INTERSECT, pgquery.SetOperation_SETOP_EXCEPT:
		leftField, err := extractor.pgExtractSelect(&pgquery.Node_SelectStmt{SelectStmt: node.SelectStmt.Larg})
		if err != nil {
			return nil, err
		}
		rightField, err := extractor.pgExtractSelect(&pgquery.Node_SelectStmt{SelectStmt: node.SelectStmt.Rarg})
		if err != nil {
			return nil, err
		}
		if len(leftField) != len(rightField) {
			return nil, errors.Errorf("each UNION/INTERSECT/EXCEPT query must have the same number of columns")
		}
		var result []fieldInfo
		for i, field := range leftField {
			result = append(result, fieldInfo{
				name:      field.name,
				table:     field.table,
				sensitive: field.sensitive || rightField[i].sensitive,
			})
		}
		return result, nil
	case pgquery.SetOperation_SETOP_NONE:
	default:
		return nil, errors.Errorf("unknown select op %v", node.SelectStmt.Op)
	}

	// SetOperation_SETOP_NONE case
	var fromFieldList []fieldInfo
	var err error
	// Extract From field list.
	for _, item := range node.SelectStmt.FromClause {
		fromFieldList, err = extractor.pgExtractNode(item)
		if err != nil {
			return nil, err
		}
		extractor.fromFieldList = fromFieldList
	}
	defer func() {
		extractor.fromFieldList = nil
	}()

	var result []fieldInfo

	// Extract Target field list.
	for _, field := range node.SelectStmt.TargetList {
		resTarget, yes := field.Node.(*pgquery.Node_ResTarget)
		if !yes {
			return nil, errors.Errorf("expect Node_ResTarget but found %T", field.Node)
		}
		switch fieldNode := resTarget.ResTarget.Val.Node.(type) {
		case *pgquery.Node_ColumnRef:
			columnRef, err := pg.ConvertNodeListToColumnNameDef(fieldNode.ColumnRef.Fields)
			if err != nil {
				return nil, err
			}
			if columnRef.ColumnName == "*" {
				// SELECT * FROM ... case.
				if columnRef.Table.Name == "" {
					result = append(result, fromFieldList...)
				} else {
					tableName, _ := pgNormalizeColumnName(columnRef)
					for _, fromField := range fromFieldList {
						if fromField.table == tableName {
							result = append(result, fromField)
						}
					}
				}
			} else {
				sensitive, err := extractor.pgExtractColumnRefFromExpressionNode(resTarget.ResTarget.Val)
				if err != nil {
					return nil, err
				}
				columnName := columnRef.ColumnName
				if resTarget.ResTarget.Name != "" {
					columnName = resTarget.ResTarget.Name
				}
				result = append(result, fieldInfo{
					name:      columnName,
					sensitive: sensitive,
				})
			}
		default:
			sensitive, err := extractor.pgExtractColumnRefFromExpressionNode(resTarget.ResTarget.Val)
			if err != nil {
				return nil, err
			}
			fieldName := resTarget.ResTarget.Name
			if fieldName == "" {
				if fieldName, err = pgExtractFieldName(resTarget.ResTarget.Val); err != nil {
					return nil, err
				}
			}
			result = append(result, fieldInfo{
				name:      fieldName,
				sensitive: sensitive,
			})
		}
	}

	return result, nil
}

func pgExtractFieldName(in *pgquery.Node) (string, error) {
	if in == nil || in.Node == nil {
		return pgUnknownFieldName, nil
	}
	switch node := in.Node.(type) {
	case *pgquery.Node_ResTarget:
		if node.ResTarget.Name != "" {
			return node.ResTarget.Name, nil
		}
		return pgExtractFieldName(node.ResTarget.Val)
	case *pgquery.Node_ColumnRef:
		columnRef, err := pg.ConvertNodeListToColumnNameDef(node.ColumnRef.Fields)
		if err != nil {
			return "", err
		}
		return columnRef.ColumnName, nil
	case *pgquery.Node_FuncCall:
		lastNode, yes := node.FuncCall.Funcname[len(node.FuncCall.Funcname)-1].Node.(*pgquery.Node_String_)
		if !yes {
			return "", errors.Errorf("expect Node_string_ but found %T", node.FuncCall.Funcname[len(node.FuncCall.Funcname)-1].Node)
		}
		return lastNode.String_.Sval, nil
	case *pgquery.Node_XmlExpr:
		switch node.XmlExpr.Op {
		case pgquery.XmlExprOp_IS_XMLCONCAT:
			return "xmlconcat", nil
		case pgquery.XmlExprOp_IS_XMLELEMENT:
			return "xmlelement", nil
		case pgquery.XmlExprOp_IS_XMLFOREST:
			return "xmlforest", nil
		case pgquery.XmlExprOp_IS_XMLPARSE:
			return "xmlparse", nil
		case pgquery.XmlExprOp_IS_XMLPI:
			return "xmlpi", nil
		case pgquery.XmlExprOp_IS_XMLROOT:
			return "xmlroot", nil
		case pgquery.XmlExprOp_IS_XMLSERIALIZE:
			return "xmlserialize", nil
		case pgquery.XmlExprOp_IS_DOCUMENT:
			return pgUnknownFieldName, nil
		}
	case *pgquery.Node_TypeCast:
		// return the arg name
		columnName, err := pgExtractFieldName(node.TypeCast.Arg)
		if err != nil {
			return "", err
		}
		if columnName != pgUnknownFieldName {
			return columnName, nil
		}
		// return the type name
		if node.TypeCast.TypeName != nil {
			lastName, yes := node.TypeCast.TypeName.Names[len(node.TypeCast.TypeName.Names)-1].Node.(*pgquery.Node_String_)
			if !yes {
				return "", errors.Errorf("expect Node_string_ but found %T", node.TypeCast.TypeName.Names[len(node.TypeCast.TypeName.Names)-1].Node)
			}
			return lastName.String_.Sval, nil
		}
	case *pgquery.Node_AConst:
		return pgUnknownFieldName, nil
	case *pgquery.Node_AExpr:
		return pgUnknownFieldName, nil
	case *pgquery.Node_CaseExpr:
		return "case", nil
	case *pgquery.Node_AArrayExpr:
		return "array", nil
	case *pgquery.Node_NullTest:
		return pgUnknownFieldName, nil
	case *pgquery.Node_XmlSerialize:
		return "xmlserialize", nil
	case *pgquery.Node_ParamRef:
		return pgUnknownFieldName, nil
	case *pgquery.Node_BoolExpr:
		return pgUnknownFieldName, nil
	case *pgquery.Node_SubLink:
		switch node.SubLink.SubLinkType {
		case pgquery.SubLinkType_EXISTS_SUBLINK:
			return "exists", nil
		case pgquery.SubLinkType_ARRAY_SUBLINK:
			return "array", nil
		case pgquery.SubLinkType_EXPR_SUBLINK:
			if node.SubLink.Subselect != nil {
				selectNode, yes := node.SubLink.Subselect.Node.(*pgquery.Node_SelectStmt)
				if !yes {
					return pgUnknownFieldName, nil
				}
				if len(selectNode.SelectStmt.TargetList) == 1 {
					return pgExtractFieldName(selectNode.SelectStmt.TargetList[0])
				}
				return pgUnknownFieldName, nil
			}
		default:
			return pgUnknownFieldName, nil
		}
	case *pgquery.Node_RowExpr:
		return "row", nil
	case *pgquery.Node_CoalesceExpr:
		return "coalesce", nil
	case *pgquery.Node_SetToDefault:
		return pgUnknownFieldName, nil
	case *pgquery.Node_AIndirection:
		// TODO(rebelice): we do not deal with the A_Indirection. Fix it.
		return pgUnknownFieldName, nil
	case *pgquery.Node_CollateClause:
		return pgExtractFieldName(node.CollateClause.Arg)
	case *pgquery.Node_CurrentOfExpr:
		return pgUnknownFieldName, nil
	case *pgquery.Node_SqlvalueFunction:
		switch node.SqlvalueFunction.Op {
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_DATE:
			return "current_date", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_TIME, pgquery.SQLValueFunctionOp_SVFOP_CURRENT_TIME_N:
			return "current_time", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP, pgquery.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP_N:
			return "current_timestamp", nil
		case pgquery.SQLValueFunctionOp_SVFOP_LOCALTIME, pgquery.SQLValueFunctionOp_SVFOP_LOCALTIME_N:
			return "localtime", nil
		case pgquery.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP, pgquery.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP_N:
			return "localtimestamp", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_ROLE:
			return "current_role", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_USER:
			return "current_user", nil
		case pgquery.SQLValueFunctionOp_SVFOP_USER:
			return "user", nil
		case pgquery.SQLValueFunctionOp_SVFOP_SESSION_USER:
			return "session_user", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_CATALOG:
			return "current_catalog", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_SCHEMA:
			return "current_schema", nil
		default:
			return pgUnknownFieldName, nil
		}
	case *pgquery.Node_MinMaxExpr:
		switch node.MinMaxExpr.Op {
		case pgquery.MinMaxOp_IS_GREATEST:
			return "greatest", nil
		case pgquery.MinMaxOp_IS_LEAST:
			return "least", nil
		default:
			return pgUnknownFieldName, nil
		}
	case *pgquery.Node_BooleanTest:
		return pgUnknownFieldName, nil
	case *pgquery.Node_GroupingFunc:
		return "grouping", nil
	}
	return pgUnknownFieldName, nil
}

func pgNormalizeColumnName(columnName *ast.ColumnNameDef) (string, string) {
	return pgNormalizeTableName(columnName.Table.Schema, columnName.Table.Name), columnName.ColumnName
}

func (extractor *sensitiveFieldExtractor) pgCheckFieldSensitive(tableName string, fieldName string) bool {
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
		sameTable := (tableName == field.table || tableName == "")
		sameField := (fieldName == field.name)
		if sameTable && sameField {
			return field.sensitive
		}
	}

	for _, field := range extractor.fromFieldList {
		sameTable := (tableName == field.table || tableName == "")
		sameField := (fieldName == field.name)
		if sameTable && sameField {
			return field.sensitive
		}
	}

	return false
}

func (extractor *sensitiveFieldExtractor) pgExtractColumnRefFromExpressionNode(in *pgquery.Node) (bool, error) {
	if in == nil {
		return false, nil
	}

	switch node := in.Node.(type) {
	case *pgquery.Node_List:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.List.Items)
	case *pgquery.Node_FuncCall:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.FuncCall.Args...)
		nodeList = append(nodeList, node.FuncCall.AggOrder...)
		nodeList = append(nodeList, node.FuncCall.AggFilter)
		return extractor.pgExtractColumnRefFromExpressionNodeList(nodeList)
	case *pgquery.Node_SortBy:
		return extractor.pgExtractColumnRefFromExpressionNode(node.SortBy.Node)
	case *pgquery.Node_XmlExpr:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.XmlExpr.Args...)
		nodeList = append(nodeList, node.XmlExpr.NamedArgs...)
		return extractor.pgExtractColumnRefFromExpressionNodeList(nodeList)
	case *pgquery.Node_ResTarget:
		return extractor.pgExtractColumnRefFromExpressionNode(node.ResTarget.Val)
	case *pgquery.Node_TypeCast:
		return extractor.pgExtractColumnRefFromExpressionNode(node.TypeCast.Arg)
	case *pgquery.Node_AConst:
		return false, nil
	case *pgquery.Node_ColumnRef:
		columnNameDef, err := pg.ConvertNodeListToColumnNameDef(node.ColumnRef.Fields)
		if err != nil {
			return false, err
		}
		return extractor.pgCheckFieldSensitive(pgNormalizeColumnName(columnNameDef)), nil
	case *pgquery.Node_AExpr:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.AExpr.Lexpr)
		nodeList = append(nodeList, node.AExpr.Rexpr)
		return extractor.pgExtractColumnRefFromExpressionNodeList(nodeList)
	case *pgquery.Node_CaseExpr:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.CaseExpr.Arg)
		nodeList = append(nodeList, node.CaseExpr.Args...)
		nodeList = append(nodeList, node.CaseExpr.Defresult)
		return extractor.pgExtractColumnRefFromExpressionNodeList(nodeList)
	case *pgquery.Node_CaseWhen:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.CaseWhen.Expr)
		nodeList = append(nodeList, node.CaseWhen.Result)
		return extractor.pgExtractColumnRefFromExpressionNodeList(nodeList)
	case *pgquery.Node_AArrayExpr:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.AArrayExpr.Elements)
	case *pgquery.Node_NullTest:
		return extractor.pgExtractColumnRefFromExpressionNode(node.NullTest.Arg)
	case *pgquery.Node_XmlSerialize:
		return extractor.pgExtractColumnRefFromExpressionNode(node.XmlSerialize.Expr)
	case *pgquery.Node_ParamRef:
		return false, nil
	case *pgquery.Node_BoolExpr:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.BoolExpr.Args)
	case *pgquery.Node_SubLink:
		sensitive, err := extractor.pgExtractColumnRefFromExpressionNode(node.SubLink.Testexpr)
		if err != nil {
			return false, err
		}
		if sensitive {
			return true, nil
		}

		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &sensitiveFieldExtractor{
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.pgExtractNode(node.SubLink.Subselect)
		if err != nil {
			return false, err
		}
		for _, field := range fieldList {
			if field.sensitive {
				return true, nil
			}
		}
		return false, nil
	case *pgquery.Node_RowExpr:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.RowExpr.Args)
	case *pgquery.Node_CoalesceExpr:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.CoalesceExpr.Args)
	case *pgquery.Node_SetToDefault:
		return false, nil
	case *pgquery.Node_AIndirection:
		return extractor.pgExtractColumnRefFromExpressionNode(node.AIndirection.Arg)
	case *pgquery.Node_CollateClause:
		return extractor.pgExtractColumnRefFromExpressionNode(node.CollateClause.Arg)
	case *pgquery.Node_CurrentOfExpr:
		return false, nil
	case *pgquery.Node_SqlvalueFunction:
		return false, nil
	case *pgquery.Node_MinMaxExpr:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.MinMaxExpr.Args)
	case *pgquery.Node_BooleanTest:
		return extractor.pgExtractColumnRefFromExpressionNode(node.BooleanTest.Arg)
	case *pgquery.Node_GroupingFunc:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.GroupingFunc.Args)
	}
	return false, nil
}

func (extractor *sensitiveFieldExtractor) pgExtractColumnRefFromExpressionNodeList(list []*pgquery.Node) (bool, error) {
	for _, node := range list {
		sensitive, err := extractor.pgExtractColumnRefFromExpressionNode(node)
		if err != nil {
			return false, err
		}
		if sensitive {
			return true, nil
		}
	}
	return false, nil
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
	schema    string
	database  string
	sensitive bool
}

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
	case *tidbast.CreateViewStmt:
		list, err := extractor.extractNode(node.Select)
		if err != nil {
			return nil, err
		}
		var result []fieldInfo
		if len(node.Cols) > 0 && len(node.Cols) != len(list) {
			return nil, errors.Errorf("The used SELECT statements have a different number of columns for view %s", node.ViewName.Name.O)
		}
		for i, item := range list {
			field := fieldInfo{
				database: item.database,
				schema:   item.schema,
				table:    node.ViewName.Name.O,
				name:     item.name,
			}
			if len(node.Cols) > 0 {
				// The column name for MySQL is case insensitive.
				field.name = node.Cols[i].L
			}
			result = append(result, field)
		}
		return result, nil
	}
	return nil, nil
}

func (extractor *sensitiveFieldExtractor) extractSetOpr(node *tidbast.SetOprStmt) ([]fieldInfo, error) {
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

func splitInitialAndRecursivePart(node *tidbast.SetOprStmt, selfName string) ([]tidbast.Node, []tidbast.Node) {
	for i, selectStmt := range node.SelectList.Selects {
		tableList := parser.ExtractMySQLTableList(selectStmt, false /* asName */)
		for _, table := range tableList {
			if table.Schema.O == "" && table.Name.O == selfName {
				return node.SelectList.Selects[:i], node.SelectList.Selects[i:]
			}
		}
	}
	return node.SelectList.Selects, nil
}

func (extractor *sensitiveFieldExtractor) extractRecursiveCTE(node *tidbast.CommonTableExpression) (db.TableSchema, error) {
	cteInfo := db.TableSchema{Name: node.Name.O}

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
					return db.TableSchema{}, err
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
			return db.TableSchema{}, errors.Errorf("Failed to find initial part for recursive common table expression")
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
			return db.TableSchema{}, err
		}

		if len(node.ColNameList) > 0 {
			if len(node.ColNameList) != len(initialField) {
				// The error content comes from MySQL.
				return db.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
			}
			for i := 0; i < len(initialField); i++ {
				initialField[i].name = node.ColNameList[i].O
			}
		}
		for _, field := range initialField {
			cteInfo.ColumnList = append(cteInfo.ColumnList, db.ColumnInfo{
				Name:      field.name,
				Sensitive: field.sensitive,
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
				return db.TableSchema{}, err
			}
			if len(fieldList) != len(cteInfo.ColumnList) {
				// The error content comes from MySQL.
				return db.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
			}

			changed := false
			for i, field := range fieldList {
				if field.sensitive && !cteInfo.ColumnList[i].Sensitive {
					changed = true
					cteInfo.ColumnList[i].Sensitive = true
				}
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

func (extractor *sensitiveFieldExtractor) extractNonRecursiveCTE(node *tidbast.CommonTableExpression) (db.TableSchema, error) {
	fieldList, err := extractor.extractNode(node.Query.Query)
	if err != nil {
		return db.TableSchema{}, err
	}
	if len(node.ColNameList) > 0 {
		if len(node.ColNameList) != len(fieldList) {
			// The error content comes from MySQL.
			return db.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
		}
		for i := 0; i < len(fieldList); i++ {
			fieldList[i].name = node.ColNameList[i].O
		}
	}
	result := db.TableSchema{
		Name:       node.Name.O,
		ColumnList: []db.ColumnInfo{},
	}
	for _, field := range fieldList {
		result.ColumnList = append(result.ColumnList, db.ColumnInfo{
			Name:      field.name,
			Sensitive: field.sensitive,
		})
	}
	return result, nil
}

func (extractor *sensitiveFieldExtractor) extractCTE(node *tidbast.CommonTableExpression) (db.TableSchema, error) {
	if node.IsRecursive {
		return extractor.extractRecursiveCTE(node)
	}
	return extractor.extractNonRecursiveCTE(node)
}

func (extractor *sensitiveFieldExtractor) extractSelect(node *tidbast.SelectStmt) ([]fieldInfo, error) {
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

func (extractor *sensitiveFieldExtractor) checkFieldSensitive(databaseName string, tableName string, fieldName string) bool {
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
		sameDatabase := (databaseName == field.database || (databaseName == "" && field.database == extractor.currentDatabase))
		sameTable := (tableName == field.table || tableName == "")
		sameField := (fieldName == field.name)
		if sameDatabase && sameTable && sameField {
			return field.sensitive
		}
	}

	for _, field := range extractor.fromFieldList {
		sameDatabase := (databaseName == field.database || (databaseName == "" && field.database == extractor.currentDatabase))
		sameTable := (tableName == field.table || tableName == "")
		sameField := (fieldName == field.name)
		if sameDatabase && sameTable && sameField {
			return field.sensitive
		}
	}

	return false
}

func (extractor *sensitiveFieldExtractor) extractColumnFromExprNode(in tidbast.ExprNode) (sensitive bool, err error) {
	if in == nil {
		return false, nil
	}

	switch node := in.(type) {
	case *tidbast.ColumnNameExpr:
		return extractor.checkFieldSensitive(node.Name.Schema.O, node.Name.Table.O, node.Name.Name.O), nil
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
		subqueryExtractor := &sensitiveFieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
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

func (extractor *sensitiveFieldExtractor) findTableSchema(databaseName string, tableName string) (string, db.TableSchema, error) {
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
		if extractor.schemaInfo.IgnoreCaseSensitive {
			lowerDatabase := strings.ToLower(database.Name)
			lowerTable := strings.ToLower(tableName)
			if lowerDatabase == strings.ToLower(database.Name) || (databaseName == "" && lowerDatabase == strings.ToLower(extractor.currentDatabase)) {
				for _, table := range database.TableList {
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
			for _, table := range database.TableList {
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

	if database, schema, err := extractor.findViewSchema(databaseName, tableName); err == nil {
		// nolint:nilerr
		return database, schema, nil
	}
	return "", db.TableSchema{}, errors.Errorf("Table or view %q.%q not found", databaseName, tableName)
}

func (extractor *sensitiveFieldExtractor) buildTableSchemaForView(viewName string, definition string) (db.TableSchema, error) {
	newExtractor := &sensitiveFieldExtractor{
		currentDatabase: extractor.currentDatabase,
		schemaInfo:      extractor.schemaInfo,
	}
	fields, err := newExtractor.extractMySQLSensitiveField(definition)
	if err != nil {
		return db.TableSchema{}, err
	}

	result := db.TableSchema{
		Name:       viewName,
		ColumnList: []db.ColumnInfo{},
	}
	for _, field := range fields {
		// nolint:gosimple
		result.ColumnList = append(result.ColumnList, db.ColumnInfo{
			Name:      field.Name,
			Sensitive: field.Sensitive,
		})
	}
	return result, nil
}

func (extractor *sensitiveFieldExtractor) findViewSchema(databaseName string, viewName string) (string, db.TableSchema, error) {
	for _, database := range extractor.schemaInfo.DatabaseList {
		if extractor.schemaInfo.IgnoreCaseSensitive {
			lowerDatabase := strings.ToLower(database.Name)
			lowerView := strings.ToLower(viewName)
			if lowerDatabase == strings.ToLower(database.Name) || (databaseName == "" && lowerDatabase == strings.ToLower(extractor.currentDatabase)) {
				for _, view := range database.ViewList {
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
			for _, view := range database.ViewList {
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
	return "", db.TableSchema{}, errors.Errorf("View %q.%q not found", databaseName, viewName)
}

func (extractor *sensitiveFieldExtractor) extractTableName(node *tidbast.TableName) ([]fieldInfo, error) {
	databaseName, tableSchema, err := extractor.findTableSchema(node.Schema.O, node.Name.O)
	if err != nil {
		return nil, err
	}

	var res []fieldInfo
	for _, column := range tableSchema.ColumnList {
		res = append(res, fieldInfo{
			name:      column.Name,
			table:     tableSchema.Name,
			database:  databaseName,
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
		// Column name in MySQL is NOT case-sensitive.
		leftFieldMap[strings.ToLower(field.name)] = field
	}
	for _, field := range rightField {
		// Column name in MySQL is NOT case-sensitive.
		rightFieldMap[strings.ToLower(field.name)] = field
	}
	if node.NaturalJoin {
		// Natural Join will merge the same column name field.
		for _, field := range leftField {
			// Merge the sensitive attribute for the same column name field.
			if rField, exists := rightFieldMap[strings.ToLower(field.name)]; exists && rField.sensitive {
				field.sensitive = true
			}
			result = append(result, field)
		}

		for _, field := range rightField {
			if _, exists := leftFieldMap[strings.ToLower(field.name)]; !exists {
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
				_, existsInUsingMap := usingMap[strings.ToLower(field.name)]
				rField, existsInRightField := rightFieldMap[strings.ToLower(field.name)]
				// Merge the sensitive attribute for the column name field in USING.
				if existsInUsingMap && existsInRightField && rField.sensitive {
					field.sensitive = true
				}
				result = append(result, field)
			}

			for _, field := range rightField {
				_, existsInUsingMap := usingMap[strings.ToLower(field.name)]
				_, existsInLeftField := leftFieldMap[strings.ToLower(field.name)]
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
