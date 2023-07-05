// Package util implements the util functions.
package util

import (
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	snowparser "github.com/bytebase/snowsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFields(sql string) ([]db.SensitiveField, error) {
	tree, err := parser.ParseSnowSQL(sql)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse snowsql")
	}
	if tree == nil {
		return nil, nil
	}

	listener := &snowsqlSensitiveFieldExtractorListener{
		extractor: extractor,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.result, listener.err
}

type snowsqlSensitiveFieldExtractorListener struct {
	*snowparser.BaseSnowflakeParserListener

	extractor *sensitiveFieldExtractor
	result    []db.SensitiveField
	err       error
}

func (l *snowsqlSensitiveFieldExtractorListener) EnterDml_command(ctx *snowparser.Dml_commandContext) {
	if l.err != nil {
		return
	}

	result, err := l.extractor.extractSnowsqlSensitiveFieldsQuery_statement(ctx.Query_statement())
	if err != nil {
		l.err = err
		return
	}
	for _, field := range result {
		l.result = append(l.result, db.SensitiveField{
			Name:      field.name,
			Sensitive: field.sensitive,
		})
	}
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsQuery_statement(ctx snowparser.IQuery_statementContext) ([]fieldInfo, error) {
	if ctx.With_expression() != nil {
		// TODO(zp): handle recursive CTE
		allCommandTableExpression := ctx.With_expression().AllCommon_table_expression()
		originalDatabaseSchema := extractor.schemaInfo.DatabaseList
		defer func() {
			extractor.schemaInfo.DatabaseList = originalDatabaseSchema
		}()

		for _, commandTableExpression := range allCommandTableExpression {
			normalizedCTEName := parser.NormalizeObjectNamePart(commandTableExpression.Id_())
			result, err := extractor.extractSnowsqlSensitiveFieldsSelect_statement(commandTableExpression.Select_statement())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract sensitive fields of the CTE %q near line %d", normalizedCTEName, commandTableExpression.GetStart().GetLine())
			}
			// TODO(zp): handle column list
			allSetOperators := ctx.AllSet_operators()
			for i, setOperator := range allSetOperators {
				// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
				// So we only need to extract the sensitive fields of the right part.
				right, err := extractor.extractSnowsqlSensitiveFieldSet_operator(setOperator)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, setOperator.GetStart().GetLine())
				}
				if len(result) != len(right) {
					return nil, errors.Wrapf(err, "the number of columns in the select statement nearly line %d returns %d fields, but %d set operator near line %d returns %d fields", commandTableExpression.Select_statement().GetStart().GetLine(), len(result), i+1, setOperator.GetStart().GetLine(), len(right))
				}
				for i := range right {
					if !result[i].sensitive {
						result[i].sensitive = right[i].sensitive
					}
				}
			}
			// Append to the extractor.schemaInfo.DatabaseList
			columnList := make([]db.ColumnInfo, 0, len(result))
			for _, field := range result {
				columnList = append(columnList, db.ColumnInfo{
					Name:      field.name,
					Sensitive: field.sensitive,
				})
			}
			extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, db.TableSchema{
				Name:       normalizedCTEName,
				ColumnList: columnList,
			})
		}
	}

	selectStatement := ctx.Select_statement()
	result, err := extractor.extractSnowsqlSensitiveFieldsSelect_statement(selectStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields of the query statement near line %d", selectStatement.GetStart().GetLine())
	}

	allSetOperators := ctx.AllSet_operators()
	for i, setOperator := range allSetOperators {
		// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
		// So we only need to extract the sensitive fields of the right part.
		right, err := extractor.extractSnowsqlSensitiveFieldSet_operator(setOperator)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, setOperator.GetStart().GetLine())
		}
		if len(result) != len(right) {
			return nil, errors.Wrapf(err, "the number of columns in the query statement nearly line %d returns %d fields, but %d set operator near line %d returns %d fields", selectStatement.GetStart().GetLine(), len(result), i+1, setOperator.GetStart().GetLine(), len(right))
		}
		for i := range right {
			if !result[i].sensitive {
				result[i].sensitive = right[i].sensitive
			}
		}

	}
	return result, nil
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldSet_operator(ctx snowparser.ISet_operatorsContext) ([]fieldInfo, error) {
	return extractor.extractSnowsqlSensitiveFieldsSelect_statement(ctx.Select_statement())
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsSelect_statement(ctx snowparser.ISelect_statementContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var fromFieldList []fieldInfo
	var err error
	if ctx.Select_optional_clauses().From_clause() != nil {
		fromFieldList, err = extractor.extractSnowsqlSensitiveFieldsFrom_clause(ctx.Select_optional_clauses().From_clause())
		if err != nil {
			return nil, err
		}
		originalFromFields := extractor.fromFieldList
		extractor.fromFieldList = fromFieldList
		defer func() {
			extractor.fromFieldList = originalFromFields
		}()
	}

	var result []fieldInfo

	var selectList snowparser.ISelect_listContext
	if ctx.Select_clause() != nil {
		selectList = ctx.Select_clause().Select_list_no_top().Select_list()
	} else if ctx.Select_top_clause() != nil {
		selectList = ctx.Select_clause().Select_list_no_top().Select_list()
	}
	for _, iSelectListElem := range selectList.AllSelect_list_elem() {
		// TODO(zp): handle expression elem
		if columnElem := iSelectListElem.Column_elem(); columnElem != nil {
			// TODO(zp): handle object_name
			if columnElem.STAR() != nil {
				result = append(result, fromFieldList...)
			} else if columnElem.Column_name() != nil {
				columnName := parser.NormalizeObjectNamePart(columnElem.Column_name().Id_())
				for _, fromField := range fromFieldList {
					if fromField.name == columnName {
						result = append(result, fromField)
					}
				}
			} else if columnElem.DOLLAR() != nil {
				columnPosition, err := strconv.Atoi(columnElem.Column_position().Num().GetText())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse column position %q to integer near line %d", columnElem.Column_position().Num().GetText(), columnElem.Column_position().Num().GetStart().GetLine())
				}
				if columnPosition < 1 {
					return nil, errors.Wrapf(err, "column position %d is invalid because it is less than 1 near line %d", columnPosition, columnElem.Column_position().Num().GetStart().GetLine())
				}
				if columnPosition > len(fromFieldList) {
					return nil, errors.Wrapf(err, "column position is invalid because want to try get the %d column near line %d, but FROM clause only returns %d columns", columnPosition, columnElem.Column_position().Num().GetStart().GetLine(), len(fromFieldList))
				}
				result = append(result, fromFieldList[columnPosition-1])
			}
			if asAlias := columnElem.As_alias(); asAlias != nil {
				result[len(result)-1].name = parser.NormalizeObjectNamePart(asAlias.Alias().Id_())
			}
		} else if expressionElem := iSelectListElem.Expression_elem(); expressionElem != nil {

		}
	}

	return result, nil
}

// The closure of the IExprContext
func (extractor *sensitiveFieldExtractor) isExprSensitive(ctx antlr.RuleContext) (string, bool, error) {
	switch ctx := ctx.(type) {
	case *snowparser.Primitive_expressionContext:
		return ctx.GetText(), false, nil
	case *snowparser.Function_callContext:
		if v := ctx.Ranking_windowed_function(); v != nil {
			return extractor.isExprSensitive(v)
		}
		panic("never reach here")
	case *snowparser.Ranking_windowed_functionContext:
		if v := ctx.Expr(); v != nil {
			return extractor.isExprSensitive(v)
		}
		if v := ctx.Over_clause(); v != nil {
			return extractor.isExprSensitive(v)
		}
		panic("never reach here")
	case *snowparser.

	}
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsFrom_clause(ctx snowparser.IFrom_clauseContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	return extractor.extractSnowsqlSensitiveFieldsTable_sources(ctx.Table_sources())
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTable_sources(ctx snowparser.ITable_sourcesContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}
	allTableSources := ctx.AllTable_source()
	var result []fieldInfo
	// If there are multiple table sources, the default join type is CROSS JOIN.
	for _, tableSource := range allTableSources {
		tableSourceResult, err := extractor.extractSnowsqlSensitiveFieldsTable_source(tableSource)
		if err != nil {
			return nil, err
		}
		result = append(result, tableSourceResult...)
	}
	return result, nil
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTable_source(ctx snowparser.ITable_sourceContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}
	return extractor.extractSnowsqlSensitiveFieldsTable_source_item_joined(ctx.Table_source_item_joined())
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTable_source_item_joined(ctx snowparser.ITable_source_item_joinedContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var left []fieldInfo
	var err error
	if ctx.Object_ref() != nil {
		left, err = extractor.extractSnowsqlSensitiveFieldsObject_ref(ctx.Object_ref())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the object ref near line %d", ctx.Object_ref().GetStart().GetLine())
		}
	}

	if ctx.Table_source_item_joined() != nil {
		left, err = extractor.extractSnowsqlSensitiveFieldsTable_source_item_joined(ctx.Table_source_item_joined())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the table source item joined near line %d", ctx.Table_source_item_joined().GetStart().GetLine())
		}
	}

	for i, joinClause := range ctx.AllJoin_clause() {
		left, err = extractor.extractSnowsqlSensitiveFieldsJoin_clause(joinClause, left)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the #%d join clause near line %d", i+1, joinClause.GetStart().GetLine())
		}
	}

	return left, nil
}

// extractSnowsqlSensitiveFieldsJoin_clause extracts sensitive fields from join clause, and return the
func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsJoin_clause(ctx snowparser.IJoin_clauseContext, left []fieldInfo) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	// Snowflake has 6 types of join:
	// INNER JOIN, LEFT OUTER JOIN, RIGHT OUTER JOIN, FULL OUTER JOIN, CROSS JOIN, and NATURAL JOIN.
	// Only the result(column num) of NATURAL JOIN may be reduced.
	right, err := extractor.extractSnowsqlSensitiveFieldsObject_ref(ctx.Object_ref())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields of the right part of the JOIN near line %d", ctx.Object_ref().GetStart().GetLine())
	}
	if ctx.NATURAL() != nil {
		// We should remove all the duplicate columns in the result set.
		// For example, if the left part has columns [a, b, c], and the right part has columns [a, b, d],
		// then the result set of NATURAL JOIN should be [a, b, c, d].
		leftMap := make(map[string]int)
		for i, field := range left {
			leftMap[field.name] = i
		}

		var result []fieldInfo
		result = append(result, left...)
		for _, field := range right {
			if _, ok := leftMap[field.name]; !ok {
				result = append(result, field)
			} else if field.sensitive {
				// If the field is in the left part and the right part, we should keep the field in the left part,
				// and set the sensitive flag to true if the field in the right part is sensitive.
				result[leftMap[field.name]].sensitive = true
			}
		}
		return result, nil
	}

	// For other types of join, we should keep all the columns for the left part and the right part.
	var result []fieldInfo
	result = append(result, left...)
	result = append(result, right...)
	return result, nil
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsObject_ref(ctx snowparser.IObject_refContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []fieldInfo

	if objectName := ctx.Object_name(); objectName != nil {
		normalizedDatabaseName, tableSchema, err := extractor.snowsqlFindTableSchema(objectName, extractor.currentDatabase, "PUBLIC")
		if err != nil {
			return nil, err
		}
		for _, column := range tableSchema.ColumnList {
			result = append(result, fieldInfo{
				database:  normalizedDatabaseName,
				table:     tableSchema.Name,
				name:      column.Name,
				sensitive: column.Sensitive,
			})
		}
	}

	// TODO(zp): Handle the value clause.
	if ctx.Values() != nil {
		return nil, nil
	}

	// TODO(zp): In data-warehouse, define a function to return multiple rows is widespread, we should parse the
	// function definition to extract the sensitive fields.
	if ctx.TABLE() != nil {
		return nil, nil
	}

	if ctx.Subquery() != nil {
		// TODO(zp): handle recursive and multiple cte.
		subqueryResult, err := extractor.extractSnowsqlSensitiveFieldsSelect_statement(ctx.Subquery().Query_statement().Select_statement())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of subquery near line %d", ctx.Subquery().GetStart().GetLine())
		}
		result = append(result, subqueryResult...)
	}

	// TODO(zp): Handle the flatten table.
	if ctx.Flatten_table() != nil {
		return nil, nil
	}

	// If the as alias is not nil, we should use the alias name to replace the original table name.
	if ctx.As_alias() != nil {
		id := ctx.As_alias().Alias().Id_()
		for i := 0; i < len(result); i++ {
			aliasName := parser.NormalizeObjectNamePart(id)
			result[i].table = aliasName
		}
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) snowsqlFindTableSchema(objectName snowparser.IObject_nameContext, fallbackDatabaseName, fallbackSchemaName string) (string, db.TableSchema, error) {
	normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizedObjectName(objectName, "", "")
	// For snowflake, we should find the table schema in cteOuterSchemaInfo by ascending order.
	if normalizedDatabaseName == "" {
		for _, tableSchema := range extractor.cteOuterSchemaInfo {
			// TODO(zp): handle the public hack.
			if normalizedTableName == tableSchema.Name {
				return normalizedDatabaseName, tableSchema, nil
			}
		}
	}
	normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizedObjectName(objectName, fallbackDatabaseName, fallbackSchemaName)
	normalizedSchemaTableName := fmt.Sprintf(`%s.%s`, normalizedSchemaName, normalizedTableName)
	for _, databaseSchema := range extractor.schemaInfo.DatabaseList {
		if databaseSchema.Name != normalizedDatabaseName {
			continue
		}
		for _, tableSchema := range databaseSchema.TableList {
			if normalizedSchemaTableName != tableSchema.Name {
				continue
			}
			return normalizedDatabaseName, tableSchema, nil
		}
	}
	return "", db.TableSchema{}, errors.Errorf(`table %s not found in database %s`, normalizedSchemaTableName, normalizedDatabaseName)
}

func normalizedObjectName(objectName snowparser.IObject_nameContext, fallbackDatabaseName, fallbackSchemaName string) (string, string, string) {
	// TODO(zp): unify here with NormalizeObjectName in backend/plugin/parser/sql/snowsql.go
	var parts []string
	if objectName == nil {
		return "", "", ""
	}
	database := fallbackDatabaseName
	if d := objectName.GetD(); d != nil {
		normalizedD := parser.NormalizeObjectNamePart(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	schema := fallbackSchemaName
	if s := objectName.GetS(); s != nil {
		normalizedS := parser.NormalizeObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)

	normalizedO := parser.NormalizeObjectNamePart(objectName.GetO())
	parts = append(parts, normalizedO)

	return parts[0], parts[1], parts[2]
}
