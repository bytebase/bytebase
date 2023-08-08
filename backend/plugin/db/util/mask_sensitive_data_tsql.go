package util

import (
	"fmt"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	tsqlparser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

func (extractor *sensitiveFieldExtractor) extractTSqlSensitiveFields(sql string) ([]db.SensitiveField, error) {
	tree, err := parser.ParseTSQL(sql)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse snowsql")
	}
	if tree == nil {
		return nil, nil
	}

	listener := &tsqlSensitiveFieldExtractorListener{
		extractor: extractor,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.result, listener.err
}

type tsqlSensitiveFieldExtractorListener struct {
	*tsqlparser.BaseTSqlParserListener

	extractor *sensitiveFieldExtractor
	result    []db.SensitiveField
	err       error
}

// EnterSelect_statement_standalone is called when production select_statement_standalone is entered.
func (l *tsqlSensitiveFieldExtractorListener) EnterDml_clause(ctx *tsqlparser.Dml_clauseContext) {
	if ctx.Select_statement_standalone() == nil {
		return
	}

	result, err := l.extractor.extractTSqlSensitiveFieldsFromSelectStatementStandalone(ctx.Select_statement_standalone())
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

// extractTSqlSensitiveFieldsFromSelectStatementStandalone extracts sensitive fields from select_statement_standalone.
func (l *sensitiveFieldExtractor) extractTSqlSensitiveFieldsFromSelectStatementStandalone(ctx tsqlparser.ISelect_statement_standaloneContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	// TODO(zp): handle the CTE
	if ctx.With_expression() != nil {
		return nil, nil
	}
	return l.extractTSqlSensitiveFieldsFromSelectStatement(ctx.Select_statement())
}

// extractTSqlSensitiveFieldsFromSelectStatement extracts sensitive fields from select_statement.
func (l *sensitiveFieldExtractor) extractTSqlSensitiveFieldsFromSelectStatement(ctx tsqlparser.ISelect_statementContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	queryResult, err := l.extractTSqlSensitiveFieldsFromQueryExpression(ctx.Query_expression())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_expression` in `select_statement`")
	}

	return queryResult, nil
}

func (l *sensitiveFieldExtractor) extractTSqlSensitiveFieldsFromQueryExpression(ctx tsqlparser.IQuery_expressionContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.Query_specification() != nil {
		return l.extractTSqlSensitiveFieldsFromQuerySpecification(ctx.Query_specification())
	}

	// TODO(zp): handle the query_expression.
	if len(ctx.AllQuery_expression()) != 0 {
		return nil, nil
	}

	panic("should not reach here")
}

func (l *sensitiveFieldExtractor) extractTSqlSensitiveFieldsFromQuerySpecification(ctx tsqlparser.IQuery_specificationContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	if from := ctx.GetFrom(); from != nil {
		fromFieldList, err := l.extractTSqlSensitiveFieldsFromTableSources(ctx.Table_sources())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_sources` in `query_specification`")
		}
		originalFromFieldList := len(l.fromFieldList)
		l.fromFieldList = append(l.fromFieldList, fromFieldList...)
		defer func() {
			l.fromFieldList = l.fromFieldList[:originalFromFieldList]
		}()
	}

	var result []fieldInfo

	selectList := ctx.Select_list()
	for _, selectListElem := range selectList.AllSelect_list_elem() {
		if asterisk := selectListElem.Asterisk(); asterisk != nil {
			var normalizedDatabaseName, normalizedSchemaName, normalizedTableName string
			if tableName := asterisk.Table_name(); tableName != nil {
				normalizedDatabaseName, normalizedSchemaName, normalizedTableName = l.splitTableNameIntoNormalizedParts(tableName)
			}
			left, err := l.tsqlGetAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get all fields of table %s.%s.%s", normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
			}
			result = append(result, left...)
		} else if udtElem := selectListElem.Udt_elem(); udtElem != nil {

		}

	}

	return result, nil
}

func (l *sensitiveFieldExtractor) extractTSqlSensitiveFieldsFromTableSources(ctx tsqlparser.ITable_sourcesContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var allTableSources []tsqlparser.ITable_sourceContext
	if v := ctx.Non_ansi_join(); v != nil {
		allTableSources = v.GetSource()
	} else if len(ctx.AllTable_source()) != 0 {
		allTableSources = ctx.GetSource()
	}

	var result []fieldInfo
	// If there are multiple table sources, the default join type is CROSS JOIN.
	for _, tableSource := range allTableSources {
		tableSourceResult, err := l.extractTSqlSensitiveFieldsFromTableSource(tableSource)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source` in `table_sources`")
		}
		result = append(result, tableSourceResult...)
	}
	return result, nil
}

func (l *sensitiveFieldExtractor) extractTSqlSensitiveFieldsFromTableSource(ctx tsqlparser.ITable_sourceContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	tableSourceItemResult, err := l.extractTSqlSensitiveFieldsFromTableSourceItem(ctx.Table_source_item())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source_item` in `table_source`")
	}

	// TODO(zp): handle join
	if ctx.GetJoins() != nil {
		return nil, nil
	}

	return tableSourceItemResult, nil
}

// extractTSqlSensitiveFieldsFromTableSourceItem extracts sensitive fields from table source item.
func (l *sensitiveFieldExtractor) extractTSqlSensitiveFieldsFromTableSourceItem(ctx tsqlparser.ITable_source_itemContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []fieldInfo
	// TODO(zp): handle other cases likes ROWSET_FUNCTION.
	if ctx.Full_table_name() != nil {
		normalizedDatabaseName, tableSchema, err := l.tsqlFindTableSchema(ctx.Full_table_name(), "", l.currentDatabase, "dbo")
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

	if ctx.Table_source() != nil {
		return l.extractTSqlSensitiveFieldsFromTableSource(ctx.Table_source())
	}

	return result, nil
}

func (l *sensitiveFieldExtractor) tsqlFindTableSchema(fullTableName tsqlparser.IFull_table_nameContext, normalizedFallbackLinkedServerName, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, db.TableSchema, error) {
	normalizedLinkedServer, normalizedDatabaseName, normalizedSchemaName, normalizedTableName := l.normalizeFullTableName(fullTableName, normalizedFallbackLinkedServerName, normalizedFallbackDatabaseName, normalizedFallbackSchemaName)
	if normalizedLinkedServer != "" {
		// TODO(zp): How do we handle the linked server?
		return "", db.TableSchema{}, errors.New(fmt.Sprintf("linked server is not supported yet, but found %q", fullTableName.GetText()))
	}
	for _, databaseSchema := range l.schemaInfo.DatabaseList {
		if normalizedDatabaseName != "" && !l.isIdentifierEqual(normalizedDatabaseName, databaseSchema.Name) {
			continue
		}
		for _, schemaSchema := range databaseSchema.SchemaList {
			if normalizedSchemaName != "" && !l.isIdentifierEqual(normalizedSchemaName, schemaSchema.Name) {
				continue
			}
			for _, tableSchema := range schemaSchema.TableList {
				if !l.isIdentifierEqual(normalizedTableName, tableSchema.Name) {
					continue
				}
				return normalizedDatabaseName, tableSchema, nil
			}
		}
	}
	return "", db.TableSchema{}, errors.New(fmt.Sprintf("table %s.%s.%s is not found", normalizedDatabaseName, normalizedSchemaName, normalizedTableName))
}

// splitTableNameIntoNormalizedParts splits the table name into normalized 3 parts: database, schema, table.
func (l *sensitiveFieldExtractor) splitTableNameIntoNormalizedParts(tableName tsqlparser.ITable_nameContext) (string, string, string) {
	var database string
	if d := tableName.GetDatabase(); d != nil {
		normalizedD := parser.NormalizeTSQLIdentifier(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}

	var schema string
	if s := tableName.GetSchema(); s != nil {
		normalizedS := parser.NormalizeTSQLIdentifier(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}

	var table string
	if t := tableName.GetTable(); t != nil {
		normalizedT := parser.NormalizeTSQLIdentifier(t)
		if normalizedT != "" {
			table = normalizedT
		}
	}
	return database, schema, table
}

// normalizeFullTableName normalizes the each part of the full table name, returns (linkedServer, database, schema, table).
func (l *sensitiveFieldExtractor) normalizeFullTableName(fullTableName tsqlparser.IFull_table_nameContext, normalizedFallbackLinkedServerName, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, string, string, string) {
	// TODO(zp): unify here and the related code in sql_service.go
	linkedServer := normalizedFallbackLinkedServerName
	if server := fullTableName.GetLinkedServer(); server != nil {
		linkedServer = parser.NormalizeTSQLIdentifier(server)
	}

	database := normalizedFallbackDatabaseName
	if d := fullTableName.GetDatabase(); d != nil {
		normalizedD := parser.NormalizeTSQLIdentifier(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}

	schema := normalizedFallbackSchemaName
	if s := fullTableName.GetSchema(); s != nil {
		normalizedS := parser.NormalizeTSQLIdentifier(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}

	var table string
	if t := fullTableName.GetTable(); t != nil {
		normalizedT := parser.NormalizeTSQLIdentifier(t)
		if normalizedT != "" {
			table = normalizedT
		}
	}

	return linkedServer, database, schema, table
}

func (l *sensitiveFieldExtractor) tsqlGetAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) ([]fieldInfo, error) {
	type maskType = uint8
	const (
		maskNone         maskType = 0
		maskDatabaseName maskType = 1 << iota
		maskSchemaName
		maskTableName
	)
	mask := maskNone
	if normalizedTableName != "" {
		mask |= maskTableName
	}
	if normalizedSchemaName != "" {
		if mask&maskTableName == 0 {
			return nil, errors.Errorf(`table name %s is specified without column name`, normalizedTableName)
		}
		mask |= maskSchemaName
	}
	if normalizedDatabaseName != "" {
		if mask&maskSchemaName == 0 {
			return nil, errors.Errorf(`database name %s is specified without schema name`, normalizedDatabaseName)
		}
		mask |= maskDatabaseName
	}

	var result []fieldInfo
	for _, field := range l.fromFieldList {
		if mask&maskDatabaseName != 0 && !l.isIdentifierEqual(normalizedDatabaseName, field.database) {
			continue
		}
		if mask&maskSchemaName != 0 && !l.isIdentifierEqual(normalizedSchemaName, field.schema) {
			continue
		}
		if mask&maskTableName != 0 && !l.isIdentifierEqual(normalizedTableName, field.table) {
			continue
		}
		result = append(result, field)
	}
	return result, nil
}

func (l *sensitiveFieldExtractor) tsqlIsFieldSensitive(normalizedDatabaseName string, normalizedSchemaName string, normalizedTableName string, normalizedColumnName string) (fieldInfo, error) {
	type maskType = uint8
	const (
		maskNone         maskType = 0
		maskDatabaseName maskType = 1 << iota
		maskSchemaName
		maskTableName
		maskColumnName
	)
	mask := maskNone
	if normalizedColumnName != "" {
		mask |= maskColumnName
	}
	if normalizedTableName != "" {
		if mask&maskColumnName == 0 {
			return fieldInfo{}, errors.Errorf(`table name %s is specified without column name`, normalizedTableName)
		}
		mask |= maskTableName
	}
	if normalizedSchemaName != "" {
		if mask&maskTableName == 0 {
			return fieldInfo{}, errors.Errorf(`schema name %s is specified without table name`, normalizedSchemaName)
		}
		mask |= maskSchemaName
	}
	if normalizedDatabaseName != "" {
		if mask&maskSchemaName == 0 {
			return fieldInfo{}, errors.Errorf(`database name %s is specified without schema name`, normalizedDatabaseName)
		}
		mask |= maskDatabaseName
	}

	if mask == maskNone {
		return fieldInfo{}, errors.Errorf(`no object name is specified`)
	}

	// We just need to iterate through the fromFieldList sequentially until we find the first matching object.

	// It is safe if there are two or more objects in the fromFieldList have the same column name, because the executor
	// will throw a compilation error if the column name is ambiguous.
	// For example, there are two tables T1 and T2, and both of them have a column named "C1". The following query will throw
	// a compilation error:
	// SELECT C1 FROM T1, T2;
	//
	// But users can specify the table name to avoid the compilation error:
	// SELECT T1.C1 FROM T1, T2;
	//
	// Further more, users can not use the original table name if they specify the alias name:
	// SELECT T1.C1 FROM T1 AS T3, T2; -- invalid identifier 'ADDRESS.ID'
	for _, field := range l.fromFieldList {
		if mask&maskDatabaseName != 0 && !l.isIdentifierEqual(normalizedDatabaseName, field.database) {
			continue
		}
		if mask&maskSchemaName != 0 && !l.isIdentifierEqual(normalizedSchemaName, field.schema) {
			continue
		}
		if mask&maskTableName != 0 && !l.isIdentifierEqual(normalizedTableName, field.table) {
			continue
		}
		if mask&maskColumnName != 0 && !l.isIdentifierEqual(normalizedColumnName, field.name) {
			continue
		}
		return field, nil
	}
	return fieldInfo{}, errors.Errorf(`no matching column %q.%q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
}

// isIdentifierEqual compares the identifier with the given normalized parts, returns true if they are equal.
// It will consider the case sensitivity based on the current database.
func (l *sensitiveFieldExtractor) isIdentifierEqual(a, b string) bool {
	if !l.schemaInfo.IgnoreCaseSensitive {
		return a == b
	}
	if len(a) != len(b) {
		return false
	}
	for i, c := range a {
		if unicode.ToLower(c) != unicode.ToLower(rune(b[i])) {
			return false
		}
	}
	return true
}
