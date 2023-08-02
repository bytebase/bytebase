// Package parser is the parser for SQL statement.
package parser

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	tsqlparser "github.com/bytebase/tsql-parser"
)

// tsqlKeywordsMap is the map of all TSQL keywords.
// Crawled from https://learn.microsoft.com/en-us/sql/t-sql/language-elements/reserved-keywords-transact-sql?view=sql-server-ver16.
var tsqlKeywordsMap = map[string]bool{
	"OVER":                             true,
	"END-EXEC":                         true,
	"ESCAPE":                           true,
	"XMLATTRIBUTES":                    true,
	"OPTION":                           true,
	"SIMILAR":                          true,
	"XMLTEXT":                          true,
	"CHECK":                            true,
	"BETWEEN":                          true,
	"XMLTABLE":                         true,
	"SQLSTATE":                         true,
	"OLD":                              true,
	"ROWCOUNT":                         true,
	"VARCHAR":                          true,
	"PERCENT_RANK":                     true,
	"EVERY":                            true,
	"ALL":                              true,
	"RULE":                             true,
	"PIVOT":                            true,
	"CONNECTION":                       true,
	"CLASS":                            true,
	"PAD":                              true,
	"UESCAPE":                          true,
	"XMLPARSE":                         true,
	"BLOB":                             true,
	"EXEC":                             true,
	"EXCEPTION":                        true,
	"ROWGUIDCOL":                       true,
	"RESULT":                           true,
	"DYNAMIC":                          true,
	"TEMPORARY":                        true,
	"ASC":                              true,
	"XMLITERATE":                       true,
	"DETERMINISTIC":                    true,
	"START":                            true,
	"CURRENT_TRANSFORM_GROUP_FOR_TYPE": true,
	"MODIFY":                           true,
	"STATEMENT":                        true,
	"PLAN":                             true,
	"XMLFOREST":                        true,
	"SQLWARNING":                       true,
	"MATCH":                            true,
	"CALLED":                           true,
	"OPENXML":                          true,
	"ROW":                              true,
	"CURSOR":                           true,
	"INTERVAL":                         true,
	"ADA":                              true,
	"REPLICATION":                      true,
	"CUME_DIST":                        true,
	"WITHIN GROUP":                     true,
	"CLOSE":                            true,
	"UPPER":                            true,
	"REVERT":                           true,
	"USING":                            true,
	"DECIMAL":                          true,
	"FULL":                             true,
	"USER":                             true,
	"TRUNCATE":                         true,
	"IMMEDIATE":                        true,
	"REGR_SLOPE":                       true,
	"SEMANTICSIMILARITYTABLE":          true,
	"PASCAL":                           true,
	"SEMANTICSIMILARITYDETAILSTABLE":   true,
	"RETURN":                           true,
	"PRIOR":                            true,
	"NONCLUSTERED":                     true,
	"INTERSECT":                        true,
	"ON":                               true,
	"TRAILING":                         true,
	"DISTINCT":                         true,
	"FETCH":                            true,
	"PARTITION":                        true,
	"AT":                               true,
	"TSEQUAL":                          true,
	"BREADTH":                          true,
	"SECTION":                          true,
	"MODIFIES":                         true,
	"OPERATION":                        true,
	"CONSTRAINTS":                      true,
	"ITERATE":                          true,
	"AND":                              true,
	"XMLVALIDATE":                      true,
	"COUNT":                            true,
	"ASYMMETRIC":                       true,
	"DESCRIPTOR":                       true,
	"SCOPE":                            true,
	"EXTERNAL":                         true,
	"FOR":                              true,
	"SELECT":                           true,
	"STATE":                            true,
	"TERMINATE":                        true,
	"EQUALS":                           true,
	"XMLPI":                            true,
	"ORDINALITY":                       true,
	"PRESERVE":                         true,
	"PRIMARY":                          true,
	"TOP":                              true,
	"GOTO":                             true,
	"BEGIN":                            true,
	"SET":                              true,
	"ISOLATION":                        true,
	"USAGE":                            true,
	"ARRAY":                            true,
	"BACKUP":                           true,
	"REFERENCES":                       true,
	"DEC":                              true,
	"DEFERRABLE":                       true,
	"WRITE":                            true,
	"ROLLBACK":                         true,
	"WAITFOR":                          true,
	"ROWS":                             true,
	"LIKE":                             true,
	"ASENSITIVE":                       true,
	"XMLCONCAT":                        true,
	"USE":                              true,
	"DAY":                              true,
	"INCLUDE":                          true,
	"OCTET_LENGTH":                     true,
	"UNKNOWN":                          true,
	"OBJECT":                           true,
	"SPECIFIC":                         true,
	"WORK":                             true,
	"SEQUENCE":                         true,
	"GRANT":                            true,
	"OPENROWSET":                       true,
	"LESS":                             true,
	"REGR_AVGY":                        true,
	"DOUBLE":                           true,
	"TRY_CONVERT":                      true,
	"HAVING":                           true,
	"PROC":                             true,
	"CLUSTERED":                        true,
	"CHARACTER_LENGTH":                 true,
	"GLOBAL":                           true,
	"DROP":                             true,
	"ELEMENT":                          true,
	"IDENTITYCOL":                      true,
	"LOCALTIMESTAMP":                   true,
	"NOCHECK":                          true,
	"OPENQUERY":                        true,
	"LAST":                             true,
	"LOCATOR":                          true,
	"COVAR_POP":                        true,
	"DELETE":                           true,
	"SYSTEM_USER":                      true,
	"DESTRUCTOR":                       true,
	"UNDER":                            true,
	"WITHIN":                           true,
	"WITH":                             true,
	"NO":                               true,
	"NCLOB":                            true,
	"LATERAL":                          true,
	"LEVEL":                            true,
	"SQLEXCEPTION":                     true,
	"INNER":                            true,
	"FOREIGN":                          true,
	"REGR_INTERCEPT":                   true,
	"XMLCOMMENT":                       true,
	"FUSION":                           true,
	"INTO":                             true,
	"NOT":                              true,
	"LOWER":                            true,
	"INITIALIZE":                       true,
	"SUBMULTISET":                      true,
	"SCHEMA":                           true,
	"PREFIX":                           true,
	"TRANSLATION":                      true,
	"MEMBER":                           true,
	"ACTION":                           true,
	"METHOD":                           true,
	"EXECUTE":                          true,
	"NATIONAL":                         true,
	"RIGHT":                            true,
	"PERCENTILE_CONT":                  true,
	"OVERLAPS":                         true,
	"MIN":                              true,
	"VARIABLE":                         true,
	"PRIVILEGES":                       true,
	"DESCRIBE":                         true,
	"FILE":                             true,
	"KILL":                             true,
	"CYCLE":                            true,
	"SQLCODE":                          true,
	"READS":                            true,
	"REGR_SXY":                         true,
	"LINENO":                           true,
	"INTERSECTION":                     true,
	"REGR_SXX":                         true,
	"BREAK":                            true,
	"CALL":                             true,
	"RETURNS":                          true,
	"LN":                               true,
	"MODULE":                           true,
	"DISCONNECT":                       true,
	"DATABASE":                         true,
	"PERCENTILE_DISC":                  true,
	"CURRENT_DATE":                     true,
	"RELATIVE":                         true,
	"DESTROY":                          true,
	"LIMIT":                            true,
	"VAR_POP":                          true,
	"EACH":                             true,
	"END":                              true,
	"FALSE":                            true,
	"PARAMETERS":                       true,
	"IGNORE":                           true,
	"GET":                              true,
	"FREE":                             true,
	"XMLNAMESPACES":                    true,
	"ALIAS":                            true,
	"REGR_COUNT":                       true,
	"CONTAINSTABLE":                    true,
	"SCROLL":                           true,
	"RESTRICT":                         true,
	"ALLOCATE":                         true,
	"ONLY":                             true,
	"DOMAIN":                           true,
	"SMALLINT":                         true,
	"DATE":                             true,
	"BINARY":                           true,
	"VAR_SAMP":                         true,
	"TREAT":                            true,
	"OPENDATASOURCE":                   true,
	"THAN":                             true,
	"NUMERIC":                          true,
	"COMMIT":                           true,
	"EXCEPT":                           true,
	"FORTRAN":                          true,
	"OFF":                              true,
	"TIMEZONE_MINUTE":                  true,
	"PREPARE":                          true,
	"OUTER":                            true,
	"SQL":                              true,
	"FULLTEXTTABLE":                    true,
	"PRECISION":                        true,
	"CROSS":                            true,
	"SAVE":                             true,
	"LEFT":                             true,
	"FILTER":                           true,
	"SAVEPOINT":                        true,
	"TRANSACTION":                      true,
	"WHILE":                            true,
	"STRUCTURE":                        true,
	"READ":                             true,
	"ROLE":                             true,
	"VALUE":                            true,
	"REVOKE":                           true,
	"ZONE":                             true,
	"XMLEXISTS":                        true,
	"INTEGER":                          true,
	"WHERE":                            true,
	"SUBSTRING":                        true,
	"NAMES":                            true,
	"IDENTITY":                         true,
	"IN":                               true,
	"SEMANTICKEYPHRASETABLE":           true,
	"RELEASE":                          true,
	"DATA":                             true,
	"AGGREGATE":                        true,
	"DENY":                             true,
	"TRIM":                             true,
	"SESSION":                          true,
	"INDICATOR":                        true,
	"EXTRACT":                          true,
	"NATURAL":                          true,
	"DISTRIBUTED":                      true,
	"TEXTSIZE":                         true,
	"CAST":                             true,
	"CARDINALITY":                      true,
	"BROWSE":                           true,
	"NEW":                              true,
	"BOOLEAN":                          true,
	"OCCURRENCES_REGEX":                true,
	"XMLSERIALIZE":                     true,
	"CURRENT_DEFAULT_TRANSFORM_GROUP":  true,
	"INDEX":                            true,
	"INSENSITIVE":                      true,
	"CONVERT":                          true,
	"SOME":                             true,
	"GENERAL":                          true,
	"KEY":                              true,
	"VARYING":                          true,
	"CONTAINS":                         true,
	"OUT":                              true,
	"MAX":                              true,
	"REFERENCING":                      true,
	"PREORDER":                         true,
	"OR":                               true,
	"HOLDLOCK":                         true,
	"STDDEV_POP":                       true,
	"XMLCAST":                          true,
	"INPUT":                            true,
	"ASSERTION":                        true,
	"DEALLOCATE":                       true,
	"XMLDOCUMENT":                      true,
	"CHAR":                             true,
	"INOUT":                            true,
	"DEFERRED":                         true,
	"READTEXT":                         true,
	"COLLATION":                        true,
	"STATIC":                           true,
	"FREETEXTTABLE":                    true,
	"THEN":                             true,
	"GROUPING":                         true,
	"SQLERROR":                         true,
	"NORMALIZE":                        true,
	"BEFORE":                           true,
	"SECOND":                           true,
	"DBCC":                             true,
	"FIRST":                            true,
	"ROUTINE":                          true,
	"COLUMN":                           true,
	"ABSOLUTE":                         true,
	"WRITETEXT":                        true,
	"IS":                               true,
	"NCHAR":                            true,
	"CONSTRAINT":                       true,
	"ERRLVL":                           true,
	"DUMP":                             true,
	"TIMEZONE_HOUR":                    true,
	"YEAR":                             true,
	"OVERLAY":                          true,
	"ROLLUP":                           true,
	"POSITION":                         true,
	"BIT":                              true,
	"SYSTEM":                           true,
	"REF":                              true,
	"CHAR_LENGTH":                      true,
	"GROUP":                            true,
	"LOAD":                             true,
	"IDENTITY_INSERT":                  true,
	"TRAN":                             true,
	"XMLELEMENT":                       true,
	"XMLBINARY":                        true,
	"SUBSTRING_REGEX":                  true,
	"LABEL":                            true,
	"ARE":                              true,
	"REGR_R2":                          true,
	"SENSITIVE":                        true,
	"INITIALLY":                        true,
	"CHECKPOINT":                       true,
	"DICTIONARY":                       true,
	"GO":                               true,
	"REAL":                             true,
	"IF":                               true,
	"HOST":                             true,
	"CASCADE":                          true,
	"DEPTH":                            true,
	"ALTER":                            true,
	"TRUE":                             true,
	"CURRENT_ROLE":                     true,
	"TO":                               true,
	"DISK":                             true,
	"OUTPUT":                           true,
	"STATISTICS":                       true,
	"FLOAT":                            true,
	"CURRENT_PATH":                     true,
	"NONE":                             true,
	"VIEW":                             true,
	"LARGE":                            true,
	"SHUTDOWN":                         true,
	"SEARCH":                           true,
	"DECLARE":                          true,
	"CURRENT_CATALOG":                  true,
	"POSTFIX":                          true,
	"WITHOUT":                          true,
	"XMLQUERY":                         true,
	"WINDOW":                           true,
	"PERCENT":                          true,
	"COMPUTE":                          true,
	"STDDEV_SAMP":                      true,
	"SYMMETRIC":                        true,
	"CONSTRUCTOR":                      true,
	"COLLECT":                          true,
	"PARAMETER":                        true,
	"COALESCE":                         true,
	"SETS":                             true,
	"CREATE":                           true,
	"TABLE":                            true,
	"DIAGNOSTICS":                      true,
	"CURRENT_SCHEMA":                   true,
	"MERGE":                            true,
	"AUTHORIZATION":                    true,
	"SUM":                              true,
	"SPACE":                            true,
	"CLOB":                             true,
	"CORRESPONDING":                    true,
	"CASE":                             true,
	"FILLFACTOR":                       true,
	"LEADING":                          true,
	"COMPLETION":                       true,
	"CUBE":                             true,
	"UNION":                            true,
	"BULK":                             true,
	"LOCAL":                            true,
	"ADD":                              true,
	"DEREF":                            true,
	"DESC":                             true,
	"OPEN":                             true,
	"RESTORE":                          true,
	"JOIN":                             true,
	"MONTH":                            true,
	"POSITION_REGEX":                   true,
	"MAP":                              true,
	"REGR_SYY":                         true,
	"HOUR":                             true,
	"VALUES":                           true,
	"TRANSLATE_REGEX":                  true,
	"FOUND":                            true,
	"RANGE":                            true,
	"BOTH":                             true,
	"INSERT":                           true,
	"COVAR_SAMP":                       true,
	"CURRENT_USER":                     true,
	"AVG":                              true,
	"REGR_AVGX":                        true,
	"LANGUAGE":                         true,
	"SIZE":                             true,
	"PROCEDURE":                        true,
	"ORDER":                            true,
	"CORR":                             true,
	"NULL":                             true,
	"SPECIFICTYPE":                     true,
	"SESSION_USER":                     true,
	"RECURSIVE":                        true,
	"PUBLIC":                           true,
	"AS":                               true,
	"RECONFIGURE":                      true,
	"CONDITION":                        true,
	"TIMESTAMP":                        true,
	"OFFSETS":                          true,
	"SQLCA":                            true,
	"INT":                              true,
	"TIME":                             true,
	"LIKE_REGEX":                       true,
	"RAISERROR":                        true,
	"CURRENT_TIME":                     true,
	"PARTIAL":                          true,
	"ATOMIC":                           true,
	"FUNCTION":                         true,
	"UPDATE":                           true,
	"ADMIN":                            true,
	"MOD":                              true,
	"EXISTS":                           true,
	"SECURITYAUDIT":                    true,
	"CONTINUE":                         true,
	"UNIQUE":                           true,
	"CONNECT":                          true,
	"LOCALTIME":                        true,
	"COLLATE":                          true,
	"CASCADED":                         true,
	"UNNEST":                           true,
	"ELSE":                             true,
	"CURRENT":                          true,
	"EXIT":                             true,
	"FREETEXT":                         true,
	"UPDATETEXT":                       true,
	"PATH":                             true,
	"XMLAGG":                           true,
	"MULTISET":                         true,
	"CHARACTER":                        true,
	"WHEN":                             true,
	"TRIGGER":                          true,
	"WIDTH_BUCKET":                     true,
	"AFTER":                            true,
	"TABLESAMPLE":                      true,
	"TRANSLATE":                        true,
	"DEFAULT":                          true,
	"HOLD":                             true,
	"BY":                               true,
	"CURRENT_TIMESTAMP":                true,
	"PRINT":                            true,
	"BIT_LENGTH":                       true,
	"CATALOG":                          true,
	"UNPIVOT":                          true,
	"WHENEVER":                         true,
	"FROM":                             true,
	"SETUSER":                          true,
	"OF":                               true,
	"NULLIF":                           true,
	"MINUTE":                           true,
	"NEXT":                             true,
	"ANY":                              true,
}

// IsTSQLKeyword returns true if the given keyword is a TSQL keywords.
func IsTSQLKeyword(keyword string, caseSensitive bool) bool {
	if !caseSensitive {
		keyword = strings.ToUpper(keyword)
	}
	return tsqlKeywordsMap[keyword]
}

// ParseTSQL parses the given SQL statement by using antlr4. Returns the AST and token stream if no error.
func ParseTSQL(statement string) (antlr.Tree, error) {
	statement = strings.TrimRight(statement, " \t\n\r\f;") + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := tsqlparser.NewTSqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := tsqlparser.NewTSqlParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &ParseErrorListener{}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &ParseErrorListener{}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Tsql_file()

	if lexerErrorListener.err != nil {
		return nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, parserErrorListener.err
	}

	return tree, nil
}

// NormalizeTSQLTableName returns the normalized table name.
func NormalizeTSQLTableName(ctx tsqlparser.ITable_nameContext, fallbackDatabaseName, fallbackSchemaName string, _ bool) string {
	database := fallbackDatabaseName
	schema := fallbackSchemaName
	table := ""
	if d := ctx.GetDatabase(); d != nil {
		if id := NormalizeTSQLIdentifier(d); id != "" {
			database = id
		}
	}
	if s := ctx.GetSchema(); s != nil {
		if id := NormalizeTSQLIdentifier(s); id != "" {
			schema = id
		}
	}
	if t := ctx.GetTable(); t != nil {
		if id := NormalizeTSQLIdentifier(t); id != "" {
			table = id
		}
	}
	return fmt.Sprintf("%s.%s.%s", database, schema, table)
}

// NormalizeTSQLIdentifier returns the normalized identifier.
// https://learn.microsoft.com/zh-cn/sql/relational-databases/databases/database-identifiers?view=sql-server-ver15
// TODO(zp): currently, we returns the lower case of the part, we may need to get the CI/CS from the server/database.
func NormalizeTSQLIdentifier(part tsqlparser.IId_Context) string {
	if part == nil {
		return ""
	}
	text := part.GetText()
	if text == "" {
		return ""
	}
	if text[0] == '[' && text[len(text)-1] == ']' {
		text = text[1 : len(text)-1]
	}

	s := ""
	for _, r := range text {
		s += string(unicode.ToLower(r))
	}
	return s
}

// FlattenExecuteStatementArgExecuteStatementArgUnnamed returns the flattened unnamed execute statement arg.
func FlattenExecuteStatementArgExecuteStatementArgUnnamed(ctx tsqlparser.IExecute_statement_argContext) []tsqlparser.IExecute_statement_arg_unnamedContext {
	var queue []tsqlparser.IExecute_statement_arg_unnamedContext
	ele := ctx
	for {
		if ele.Execute_statement_arg_unnamed() == nil {
			break
		}
		queue = append(queue, ele.Execute_statement_arg_unnamed())
		if len(ele.AllExecute_statement_arg()) != 1 {
			break
		}
		ele = ele.AllExecute_statement_arg()[0]
	}
	return queue
}

// extractMSSQLNormalizedResourceListFromSelectStatement extracts the list of resources from the SELECT statement, and normalizes the object names with the NON-EMPTY currentNormalizedDatabase and currentNormalizedSchema.
func extractMSSQLNormalizedResourceListFromSelectStatement(currentNormalizedDatabase string, currentNormalizedSchema string, selectStatement string) ([]SchemaResource, error) {
	tree, err := ParseTSQL(selectStatement)
	if err != nil {
		return nil, err
	}

	l := &tsqlReasourceExtractListener{
		currentDatabase: currentNormalizedDatabase,
		currentSchema:   currentNormalizedSchema,
		resourceMap:     make(map[string]SchemaResource),
	}

	var result []SchemaResource
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result, nil
}

type tsqlReasourceExtractListener struct {
	*tsqlparser.BaseTSqlParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]SchemaResource
}

// EnterTable_source_item is called when the parser enters the table_source_item production.
func (l *tsqlReasourceExtractListener) EnterTable_source_item(ctx *tsqlparser.Table_source_itemContext) {
	if fullTableName := ctx.Full_table_name(); fullTableName != nil {
		var parts []string
		var linkedServer string
		if server := fullTableName.GetLinkedServer(); server != nil {
			linkedServer = NormalizeTSQLIdentifier(server)
		}
		parts = append(parts, linkedServer)

		database := l.currentDatabase
		if d := fullTableName.GetDatabase(); d != nil {
			normalizedD := NormalizeTSQLIdentifier(d)
			if normalizedD != "" {
				database = normalizedD
			}
		}
		parts = append(parts, database)

		schema := l.currentSchema
		if s := fullTableName.GetSchema(); s != nil {
			normalizedS := NormalizeTSQLIdentifier(s)
			if normalizedS != "" {
				schema = normalizedS
			}
		}
		parts = append(parts, schema)

		var table string
		if t := fullTableName.GetTable(); t != nil {
			normalizedT := NormalizeTSQLIdentifier(t)
			if normalizedT != "" {
				table = normalizedT
			}
		}
		parts = append(parts, table)
		normalizedObjectName := strings.Join(parts, ".")
		l.resourceMap[normalizedObjectName] = SchemaResource{
			LinkedServer: linkedServer,
			Database:     database,
			Schema:       schema,
			Table:        table,
		}
	}

	if rowsetFunction := ctx.Rowset_function(); rowsetFunction != nil {
		return
	}

	// https://simonlearningsqlserver.wordpress.com/tag/changetable/
	// It seems that the CHANGETABLE is only return some statistics, so we ignore it.
	if changeTable := ctx.Change_table(); changeTable != nil {
		return
	}

	// other...
}
