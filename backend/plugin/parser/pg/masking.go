package pg

import (
	"fmt"
	"regexp"

	pgquery "github.com/pganalyze/pg_query_go/v4"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/pkg/errors"
)

const (
	pgUnknownFieldName = "?column?"
)

func init() {
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_POSTGRES, GetMaskedFields)
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_REDSHIFT, GetMaskedFields)
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_RISINGWAVE, GetMaskedFields)
}

func GetMaskedFields(statement, _ string, schemaInfo *base.SensitiveSchemaInfo) ([]base.SensitiveField, error) {
	extractor := &fieldExtractor{
		schemaInfo: schemaInfo,
	}
	result, err := extractor.extractSensitiveFields(statement)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type fieldExtractor struct {
	// For Oracle, we need to know the current database to determine if the table is in the current schema.
	schemaInfo         *base.SensitiveSchemaInfo
	outerSchemaInfo    []base.FieldInfo
	cteOuterSchemaInfo []base.TableSchema

	// SELECT statement specific field.
	fromFieldList []base.FieldInfo
}

func (extractor *fieldExtractor) extractSensitiveFields(statement string) ([]base.SensitiveField, error) {
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
		tableNotFound := regexp.MustCompile("^Table \"(.*)\\.(.*)\" not found$")
		content := tableNotFound.FindStringSubmatch(err.Error())
		if len(content) == 3 && IsSystemSchema(content[1]) {
			// skip for system schema
			return nil, nil
		}
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

func (extractor *fieldExtractor) pgExtractNode(in *pgquery.Node) ([]base.FieldInfo, error) {
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

func (extractor *fieldExtractor) pgExtractJoin(in *pgquery.Node_JoinExpr) ([]base.FieldInfo, error) {
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

func pgMergeJoinField(node *pgquery.Node_JoinExpr, leftField []base.FieldInfo, rightField []base.FieldInfo) ([]base.FieldInfo, error) {
	leftFieldMap := make(map[string]base.FieldInfo)
	rightFieldMap := make(map[string]base.FieldInfo)
	var result []base.FieldInfo
	for _, field := range leftField {
		leftFieldMap[field.Name] = field
	}
	for _, field := range rightField {
		rightFieldMap[field.Name] = field
	}
	if node.JoinExpr.IsNatural {
		// Natural Join will merge the same column name field.
		for _, field := range leftField {
			// Merge the sensitive attribute for the same column name field.
			if rField, exists := rightFieldMap[field.Name]; exists {
				field.MaskingAttributes.TransmittedBy(rField.MaskingAttributes)
			}
			result = append(result, field)
		}

		for _, field := range rightField {
			if _, exists := leftFieldMap[field.Name]; !exists {
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
				_, existsInUsingMap := usingMap[field.Name]
				rField, existsInRightField := rightFieldMap[field.Name]
				// Merge the sensitive attribute for the column name field in USING.
				if existsInUsingMap && existsInRightField {
					field.MaskingAttributes.TransmittedBy(rField.MaskingAttributes)
				}
				result = append(result, field)
			}

			for _, field := range rightField {
				_, existsInUsingMap := usingMap[field.Name]
				_, existsInLeftField := leftFieldMap[field.Name]
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

func (extractor *fieldExtractor) pgExtractRangeSubselect(node *pgquery.Node_RangeSubselect) ([]base.FieldInfo, error) {
	fieldList, err := extractor.pgExtractNode(node.RangeSubselect.Subquery)
	if err != nil {
		return nil, err
	}
	if node.RangeSubselect.Alias != nil {
		var result []base.FieldInfo
		aliasName, columnNameList, err := pgExtractAlias(node.RangeSubselect.Alias)
		if err != nil {
			return nil, err
		}
		if len(columnNameList) != 0 && len(columnNameList) != len(fieldList) {
			return nil, errors.Errorf("expect equal length but found %d and %d", len(columnNameList), len(fieldList))
		}
		for i, item := range fieldList {
			columnName := item.Name
			if len(columnNameList) > 0 {
				columnName = columnNameList[i]
			}
			result = append(result, base.FieldInfo{
				Schema:            "public",
				Table:             aliasName,
				Name:              columnName,
				MaskingAttributes: item.MaskingAttributes,
			})
		}
		return result, nil
	}
	return fieldList, nil
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

func (extractor *fieldExtractor) pgExtractRangeVar(node *pgquery.Node_RangeVar) ([]base.FieldInfo, error) {
	tableSchema, err := extractor.pgFindTableSchema(node.RangeVar.Schemaname, node.RangeVar.Relname)
	if err != nil {
		return nil, err
	}

	var res []base.FieldInfo
	if node.RangeVar.Alias == nil {
		for _, column := range tableSchema.ColumnList {
			res = append(res, base.FieldInfo{
				Name:              column.Name,
				Table:             tableSchema.Name,
				MaskingAttributes: column.MaskingAttributes,
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
			columnName := column.Name
			if len(columnNameList) > 0 {
				columnName = columnNameList[i]
			}
			res = append(res, base.FieldInfo{
				Schema:            "public",
				Name:              columnName,
				Table:             aliasName,
				MaskingAttributes: column.MaskingAttributes,
			})
		}
	}

	return res, nil
}

func (extractor *fieldExtractor) pgFindTableSchema(schemaName string, tableName string) (base.TableSchema, error) {
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
		for _, schema := range database.SchemaList {
			if schemaName == "" && schema.Name == "public" || schemaName == schema.Name {
				for _, table := range schema.TableList {
					if tableName == table.Name {
						return table, nil
					}
				}
			}
		}
	}
	return base.TableSchema{}, errors.Errorf("Table %q not found", tableName)
}

func (extractor *fieldExtractor) pgExtractRecursiveCTE(node *pgquery.Node_CommonTableExpr) (base.TableSchema, error) {
	switch selectNode := node.CommonTableExpr.Ctequery.Node.(type) {
	case *pgquery.Node_SelectStmt:
		if selectNode.SelectStmt.Op != pgquery.SetOperation_SETOP_UNION {
			return extractor.pgExtractNonRecursiveCTE(node)
		}
		// For PostgreSQL, recursive CTE will be an UNION statement, and the left node is the initial part,
		// the right node is the recursive part.
		initialField, err := extractor.pgExtractSelect(&pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Larg})
		if err != nil {
			return base.TableSchema{}, err
		}
		if len(node.CommonTableExpr.Aliascolnames) > 0 {
			if len(node.CommonTableExpr.Aliascolnames) != len(initialField) {
				return base.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
			}
			for i, nameNode := range node.CommonTableExpr.Aliascolnames {
				stringNode, yes := nameNode.Node.(*pgquery.Node_String_)
				if !yes {
					return base.TableSchema{}, errors.Errorf("expect Node_String_ but found %T", nameNode.Node)
				}
				initialField[i].Name = stringNode.String_.Sval
			}
		}

		cteInfo := base.TableSchema{Name: node.CommonTableExpr.Ctename}
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
			fieldList, err := extractor.pgExtractSelect(&pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Rarg})
			if err != nil {
				return base.TableSchema{}, err
			}
			if len(fieldList) != len(cteInfo.ColumnList) {
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
		return extractor.pgExtractNonRecursiveCTE(node)
	}
}

func (extractor *fieldExtractor) pgExtractNonRecursiveCTE(node *pgquery.Node_CommonTableExpr) (base.TableSchema, error) {
	fieldList, err := extractor.pgExtractNode(node.CommonTableExpr.Ctequery)
	if err != nil {
		return base.TableSchema{}, err
	}
	if len(node.CommonTableExpr.Aliascolnames) > 0 {
		if len(node.CommonTableExpr.Aliascolnames) != len(fieldList) {
			return base.TableSchema{}, errors.Errorf("The common table expression and column names list have different column counts")
		}
		var nameList []string
		for _, nameNode := range node.CommonTableExpr.Aliascolnames {
			stringNode, yes := nameNode.Node.(*pgquery.Node_String_)
			if !yes {
				return base.TableSchema{}, errors.Errorf("expect Node_String_ but found %T", nameNode.Node)
			}
			nameList = append(nameList, stringNode.String_.Sval)
		}
		for i := 0; i < len(fieldList); i++ {
			fieldList[i].Name = nameList[i]
		}
	}
	result := base.TableSchema{
		Name:       node.CommonTableExpr.Ctename,
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

func (extractor *fieldExtractor) pgExtractSelect(node *pgquery.Node_SelectStmt) ([]base.FieldInfo, error) {
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
			var cteTable base.TableSchema
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
		var result []base.FieldInfo
		for _, row := range node.SelectStmt.ValuesLists {
			var maskingAttributesList []base.MaskingAttributes
			list, yes := row.Node.(*pgquery.Node_List)
			if !yes {
				return nil, errors.Errorf("expect Node_List but found %T", row.Node)
			}
			for _, item := range list.List.Items {
				maskingLevel, err := extractor.pgExtractColumnRefFromExpressionNode(item)
				if err != nil {
					return nil, err
				}
				maskingAttributesList = append(maskingAttributesList, maskingLevel)
			}
			if len(result) == 0 {
				for i, item := range maskingAttributesList {
					result = append(result, base.FieldInfo{
						Name:              fmt.Sprintf("column%d", i+1),
						MaskingAttributes: item,
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
		var result []base.FieldInfo
		for i, field := range leftField {
			finalAttributes := base.NewDefaultMaskingAttributes()
			finalAttributes.TransmittedBy(field.MaskingAttributes)
			finalAttributes.TransmittedBy(rightField[i].MaskingAttributes)
			result = append(result, base.FieldInfo{
				Name:              field.Name,
				Table:             field.Table,
				MaskingAttributes: finalAttributes,
			})
		}
		return result, nil
	case pgquery.SetOperation_SETOP_NONE:
	default:
		return nil, errors.Errorf("unknown select op %v", node.SelectStmt.Op)
	}

	// SetOperation_SETOP_NONE case
	var fromFieldList []base.FieldInfo
	var err error
	// Extract From field list.
	for _, item := range node.SelectStmt.FromClause {
		fromFieldList, err = extractor.pgExtractNode(item)
		if err != nil {
			return nil, err
		}
		// FIXME: append
		extractor.fromFieldList = fromFieldList
	}
	defer func() {
		extractor.fromFieldList = nil
	}()

	var result []base.FieldInfo

	// Extract Target field list.
	for _, field := range node.SelectStmt.TargetList {
		resTarget, yes := field.Node.(*pgquery.Node_ResTarget)
		if !yes {
			return nil, errors.Errorf("expect Node_ResTarget but found %T", field.Node)
		}
		switch fieldNode := resTarget.ResTarget.Val.Node.(type) {
		case *pgquery.Node_ColumnRef:
			columnRef, err := pgrawparser.ConvertNodeListToColumnNameDef(fieldNode.ColumnRef.Fields)
			if err != nil {
				return nil, err
			}
			if columnRef.ColumnName == "*" {
				// SELECT * FROM ... case.
				if columnRef.Table.Name == "" {
					result = append(result, fromFieldList...)
				} else {
					schemaName, tableName, _ := extractSchemaTableColumnName(columnRef)
					for _, fromField := range fromFieldList {
						if fromField.Schema == schemaName && fromField.Table == tableName {
							result = append(result, fromField)
						}
					}
				}
			} else {
				maskingLevel, err := extractor.pgExtractColumnRefFromExpressionNode(resTarget.ResTarget.Val)
				if err != nil {
					return nil, err
				}
				columnName := columnRef.ColumnName
				if resTarget.ResTarget.Name != "" {
					columnName = resTarget.ResTarget.Name
				}
				result = append(result, base.FieldInfo{
					Name:              columnName,
					MaskingAttributes: maskingLevel,
				})
			}
		default:
			maskingLevel, err := extractor.pgExtractColumnRefFromExpressionNode(resTarget.ResTarget.Val)
			if err != nil {
				return nil, err
			}
			fieldName := resTarget.ResTarget.Name
			if fieldName == "" {
				if fieldName, err = pgExtractFieldName(resTarget.ResTarget.Val); err != nil {
					return nil, err
				}
			}
			result = append(result, base.FieldInfo{
				Name:              fieldName,
				MaskingAttributes: maskingLevel,
			})
		}
	}

	return result, nil
}

func pgExtractFieldName(node *pgquery.Node) (string, error) {
	if node == nil || node.Node == nil {
		return pgUnknownFieldName, nil
	}
	switch node := node.Node.(type) {
	case *pgquery.Node_ResTarget:
		if node.ResTarget.Name != "" {
			return node.ResTarget.Name, nil
		}
		return pgExtractFieldName(node.ResTarget.Val)
	case *pgquery.Node_ColumnRef:
		columnRef, err := pgrawparser.ConvertNodeListToColumnNameDef(node.ColumnRef.Fields)
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

func extractSchemaTableColumnName(columnName *ast.ColumnNameDef) (string, string, string) {
	return columnName.Table.Schema, columnName.Table.Name, columnName.ColumnName
}

func (extractor *fieldExtractor) pgCheckFieldMaskingLevel(schemaName string, tableName string, fieldName string) base.MaskingAttributes {
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
		if (schemaName == "" && field.Schema == "public") || schemaName == field.Schema {
			sameTable := (tableName == field.Table || tableName == "")
			sameField := (fieldName == field.Name)
			if sameTable && sameField {
				return field.MaskingAttributes
			}
		}
	}

	for _, field := range extractor.fromFieldList {
		sameTable := (tableName == field.Table || tableName == "")
		sameField := (fieldName == field.Name)
		if sameTable && sameField {
			return field.MaskingAttributes
		}
	}

	return base.NewDefaultMaskingAttributes()
}

func (extractor *fieldExtractor) pgExtractColumnRefFromExpressionNode(in *pgquery.Node) (base.MaskingAttributes, error) {
	if in == nil {
		return base.NewDefaultMaskingAttributes(), nil
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
		return base.NewDefaultMaskingAttributes(), nil
	case *pgquery.Node_ColumnRef:
		columnNameDef, err := pgrawparser.ConvertNodeListToColumnNameDef(node.ColumnRef.Fields)
		if err != nil {
			return base.NewDefaultMaskingAttributes(), err
		}
		return extractor.pgCheckFieldMaskingLevel(extractSchemaTableColumnName(columnNameDef)), nil
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
		return base.NewDefaultMaskingAttributes(), nil
	case *pgquery.Node_BoolExpr:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.BoolExpr.Args)
	case *pgquery.Node_SubLink:
		maskingAttributes, err := extractor.pgExtractColumnRefFromExpressionNode(node.SubLink.Testexpr)
		if err != nil {
			return base.NewDefaultMaskingAttributes(), err
		}
		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &fieldExtractor{
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.pgExtractNode(node.SubLink.Subselect)
		if err != nil {
			return base.NewDefaultMaskingAttributes(), err
		}

		for _, field := range fieldList {
			maskingAttributes.TransmittedByInExpression(field.MaskingAttributes)
			if maskingAttributes.IsNeverChangeInTransmission() {
				return maskingAttributes, nil
			}
		}
		return maskingAttributes, nil
	case *pgquery.Node_RowExpr:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.RowExpr.Args)
	case *pgquery.Node_CoalesceExpr:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.CoalesceExpr.Args)
	case *pgquery.Node_SetToDefault:
		return base.NewDefaultMaskingAttributes(), nil
	case *pgquery.Node_AIndirection:
		return extractor.pgExtractColumnRefFromExpressionNode(node.AIndirection.Arg)
	case *pgquery.Node_CollateClause:
		return extractor.pgExtractColumnRefFromExpressionNode(node.CollateClause.Arg)
	case *pgquery.Node_CurrentOfExpr:
		return base.NewDefaultMaskingAttributes(), nil
	case *pgquery.Node_SqlvalueFunction:
		return base.NewDefaultMaskingAttributes(), nil
	case *pgquery.Node_MinMaxExpr:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.MinMaxExpr.Args)
	case *pgquery.Node_BooleanTest:
		return extractor.pgExtractColumnRefFromExpressionNode(node.BooleanTest.Arg)
	case *pgquery.Node_GroupingFunc:
		return extractor.pgExtractColumnRefFromExpressionNodeList(node.GroupingFunc.Args)
	}
	return base.NewDefaultMaskingAttributes(), nil
}

func (extractor *fieldExtractor) pgExtractColumnRefFromExpressionNodeList(list []*pgquery.Node) (base.MaskingAttributes, error) {
	finalAttributes := base.NewDefaultMaskingAttributes()
	for _, node := range list {
		maskingAttributes, err := extractor.pgExtractColumnRefFromExpressionNode(node)
		if err != nil {
			return base.NewDefaultMaskingAttributes(), err
		}
		finalAttributes.TransmittedByInExpression(maskingAttributes)
		if finalAttributes.IsNeverChangeInTransmission() {
			return finalAttributes, nil
		}
	}
	return finalAttributes, nil
}
