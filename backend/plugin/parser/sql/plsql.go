package parser

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/plsql-parser"
)

// https://docs.oracle.com/en/database/oracle/oracle-database/21/lnpls/plsql-reserved-words-keywords.html#GUID-9BAA3A99-41B1-45CB-A91E-1E482BC1F927
var oracleReservedWords = map[string]bool{
	"ALL":        true,
	"ALTER":      true,
	"AND":        true,
	"ANY":        true,
	"AS":         true,
	"ASC":        true,
	"AT":         true,
	"BEGIN":      true,
	"BETWEEN":    true,
	"BY":         true,
	"CASE":       true,
	"CHECK":      true,
	"CLUSTERS":   true,
	"CLUSTER":    true,
	"COLAUTH":    true,
	"COLUMNS":    true,
	"COMPRESS":   true,
	"CONNECT":    true,
	"CRASH":      true,
	"CREATE":     true,
	"CURSOR":     true,
	"DECLARE":    true,
	"DEFAULT":    true,
	"DESC":       true,
	"DISTINCT":   true,
	"DROP":       true,
	"ELSE":       true,
	"END":        true,
	"EXCEPTION":  true,
	"EXCLUSIVE":  true,
	"FETCH":      true,
	"FOR":        true,
	"FROM":       true,
	"FUNCTION":   true,
	"GOTO":       true,
	"GRANT":      true,
	"GROUP":      true,
	"HAVING":     true,
	"IDENTIFIED": true,
	"IF":         true,
	"IN":         true,
	"INDEX":      true,
	"INDEXES":    true,
	"INSERT":     true,
	"INTERSECT":  true,
	"INTO":       true,
	"IS":         true,
	"LIKE":       true,
	"LOCK":       true,
	"MINUS":      true,
	"MODE":       true,
	"NOCOMPRESS": true,
	"NOT":        true,
	"NOWAIT":     true,
	"NULL":       true,
	"OF":         true,
	"ON":         true,
	"OPTION":     true,
	"OR":         true,
	"ORDER":      true,
	"OVERLAPS":   true,
	"PROCEDURE":  true,
	"PUBLIC":     true,
	"RESOURCE":   true,
	"REVOKE":     true,
	"SELECT":     true,
	"SHARE":      true,
	"SIZE":       true,
	"SQL":        true,
	"START":      true,
	"SUBTYPE":    true,
	"TABAUTH":    true,
	"TABLE":      true,
	"THEN":       true,
	"TO":         true,
	"TYPE":       true,
	"UNION":      true,
	"UNIQUE":     true,
	"UPDATE":     true,
	"VALUES":     true,
	"VIEW":       true,
	"VIEWS":      true,
	"WHEN":       true,
	"WHERE":      true,
	"WITH":       true,
}

var oracleKeywords = map[string]bool{
	"A":               true,
	"ADD":             true,
	"ACCESSIBLE":      true,
	"AGENT":           true,
	"AGGREGATE":       true,
	"ARRAY":           true,
	"ATTRIBUTE":       true,
	"AUTHID":          true,
	"AVG":             true,
	"BFILE_BASE":      true,
	"BINARY":          true,
	"BLOB_BASE":       true,
	"BLOCK":           true,
	"BODY":            true,
	"BOTH":            true,
	"BOUND":           true,
	"BULK":            true,
	"BYTE":            true,
	"C":               true,
	"CALL":            true,
	"CALLING":         true,
	"CASCADE":         true,
	"CHAR":            true,
	"CHAR_BASE":       true,
	"CHARACTER":       true,
	"CHARSET":         true,
	"CHARSETFORM":     true,
	"CHARSETID":       true,
	"CLOB_BASE":       true,
	"CLONE":           true,
	"CLOSE":           true,
	"COLLECT":         true,
	"COMMENT":         true,
	"COMMIT":          true,
	"COMMITTED":       true,
	"COMPILED":        true,
	"CONSTANT":        true,
	"CONSTRUCTOR":     true,
	"CONTEXT":         true,
	"CONTINUE":        true,
	"CONVERT":         true,
	"COUNT":           true,
	"CREDENTIAL":      true,
	"CURRENT":         true,
	"CUSTOMDATUM":     true,
	"DANGLING":        true,
	"DATA":            true,
	"DATE":            true,
	"DATE_BASE":       true,
	"DAY":             true,
	"DEFINE":          true,
	"DELETE":          true,
	"DETERMINISTIC":   true,
	"DIRECTORY":       true,
	"DOUBLE":          true,
	"DURATION":        true,
	"ELEMENT":         true,
	"ELSIF":           true,
	"EMPTY":           true,
	"ESCAPE":          true,
	"EXCEPT":          true,
	"EXCEPTIONS":      true,
	"EXECUTE":         true,
	"EXISTS":          true,
	"EXIT":            true,
	"EXTERNAL":        true,
	"FINAL":           true,
	"FIRST":           true,
	"FIXED":           true,
	"FLOAT":           true,
	"FORALL":          true,
	"FORCE":           true,
	"GENERAL":         true,
	"HASH":            true,
	"HEAP":            true,
	"HIDDEN":          true,
	"HOUR":            true,
	"IMMEDIATE":       true,
	"IMMUTABLE":       true,
	"INCLUDING":       true,
	"INDICATOR":       true,
	"INDICES":         true,
	"INFINITE":        true,
	"INSTANTIABLE":    true,
	"INT":             true,
	"INTERFACE":       true,
	"INTERVAL":        true,
	"INVALIDATE":      true,
	"ISOLATION":       true,
	"JAVA":            true,
	"LANGUAGE":        true,
	"LARGE":           true,
	"LEADING":         true,
	"LENGTH":          true,
	"LEVEL":           true,
	"LIBRARY":         true,
	"LIKE2":           true,
	"LIKE4":           true,
	"LIKEC":           true,
	"LIMIT":           true,
	"LIMITED":         true,
	"LOCAL":           true,
	"LONG":            true,
	"LOOP":            true,
	"MAP":             true,
	"MAX":             true,
	"MAXLEN":          true,
	"MEMBER":          true,
	"MERGE":           true,
	"MIN":             true,
	"MINUTE":          true,
	"MOD":             true,
	"MODIFY":          true,
	"MONTH":           true,
	"MULTISET":        true,
	"MUTABLE":         true,
	"NAME":            true,
	"NAN":             true,
	"NATIONAL":        true,
	"NATIVE":          true,
	"NCHAR":           true,
	"NEW":             true,
	"NOCOPY":          true,
	"NUMBER_BASE":     true,
	"OBJECT":          true,
	"OCICOLL":         true,
	"OCIDATE":         true,
	"OCIDATETIME":     true,
	"OCIDURATION":     true,
	"OCIINTERVAL":     true,
	"OCILOBLOCATOR":   true,
	"OCINUMBER":       true,
	"OCIRAW":          true,
	"OCIREF":          true,
	"OCIREFCURSOR":    true,
	"OCIROWID":        true,
	"OCISTRING":       true,
	"OCITYPE":         true,
	"OLD":             true,
	"ONLY":            true,
	"OPAQUE":          true,
	"OPEN":            true,
	"OPERATOR":        true,
	"ORACLE":          true,
	"ORADATA":         true,
	"ORGANIZATION":    true,
	"ORLANY":          true,
	"ORLVARY":         true,
	"OTHERS":          true,
	"OUT":             true,
	"OVERRIDING":      true,
	"PACKAGE":         true,
	"PARALLEL_ENABLE": true,
	"PARAMETER":       true,
	"PARAMETERS":      true,
	"PARENT":          true,
	"PARTITION":       true,
	"PASCAL":          true,
	"PERSISTABLE":     true,
	"PIPE":            true,
	"PIPELINED":       true,
	"PLUGGABLE":       true,
	"POLYMORPHIC":     true,
	"PRAGMA":          true,
	"PRECISION":       true,
	"PRIOR":           true,
	"PRIVATE":         true,
	"RAISE":           true,
	"RANGE":           true,
	"RAW":             true,
	"READ":            true,
	"RECORD":          true,
	"REF":             true,
	"REFERENCE":       true,
	"RELIES_ON":       true,
	"REM":             true,
	"REMAINDER":       true,
	"RENAME":          true,
	"RESULT":          true,
	"RESULT_CACHE":    true,
	"RETURN":          true,
	"RETURNING":       true,
	"REVERSE":         true,
	"ROLLBACK":        true,
	"ROW":             true,
	"SAMPLE":          true,
	"SAVE":            true,
	"SAVEPOINT":       true,
	"SB1":             true,
	"SB2":             true,
	"SB4":             true,
	"SECOND":          true,
	"SEGMENT":         true,
	"SELF":            true,
	"SEPARATE":        true,
	"SEQUENCE":        true,
	"SERIALIZABLE":    true,
	"SET":             true,
	"SHORT":           true,
	"SIZE_T":          true,
	"SOME":            true,
	"SPARSE":          true,
	"SQLCODE":         true,
	"SQLDATA":         true,
	"SQLNAME":         true,
	"SQLSTATE":        true,
	"STANDARD":        true,
	"STATIC":          true,
	"STDDEV":          true,
	"STORED":          true,
	"STRING":          true,
	"STRUCT":          true,
	"STYLE":           true,
	"SUBMULTISET":     true,
	"SUBPARTITION":    true,
	"SUBSTITUTABLE":   true,
	"SUM":             true,
	"SYNONYM":         true,
	"TDO":             true,
	"THE":             true,
	"TIME":            true,
	"TIMESTAMP":       true,
	"TIMEZONE_ABBR":   true,
	"TIMEZONE_HOUR":   true,
	"TIMEZONE_MINUTE": true,
	"TIMEZONE_REGION": true,
	"TRAILING":        true,
	"TRANSACTION":     true,
	"TRANSACTIONAL":   true,
	"TRUSTED":         true,
	"UB1":             true,
	"UB2":             true,
	"UB4":             true,
	"UNDER":           true,
	"UNPLUG":          true,
	"UNSIGNED":        true,
	"UNTRUSTED":       true,
	"USE":             true,
	"USING":           true,
	"VALIST":          true,
	"VALUE":           true,
	"VARIABLE":        true,
	"VARIANCE":        true,
	"VARRAY":          true,
	"VARYING":         true,
	"VOID":            true,
	"WHILE":           true,
	"WORK":            true,
	"WRAPPED":         true,
	"WRITE":           true,
	"YEAR":            true,
	"ZONE":            true,
}

// SyntaxError is a syntax error.
type SyntaxError struct {
	Line    int
	Column  int
	Message string
}

// Error returns the error message.
func (e *SyntaxError) Error() string {
	return e.Message
}

// ParseErrorListener is a custom error listener for PLSQL parser.
type ParseErrorListener struct {
	err *SyntaxError
}

// NewPLSQLErrorListener creates a new PLSQLErrorListener.
func NewPLSQLErrorListener() *ParseErrorListener {
	return &ParseErrorListener{}
}

// SyntaxError returns the errors.
func (l *ParseErrorListener) SyntaxError(_ antlr.Recognizer, token any, line, column int, _ string, _ antlr.RecognitionException) {
	if l.err == nil {
		errMessage := ""
		if token, ok := token.(*antlr.CommonToken); ok {
			stream := token.GetInputStream()
			start := token.GetStart() - 40
			if start < 0 {
				start = 0
			}
			stop := token.GetStop()
			if stop >= stream.Size() {
				stop = stream.Size() - 1
			}
			errMessage = fmt.Sprintf("related text: %s", stream.GetTextFromInterval(antlr.NewInterval(start, stop)))
		}
		l.err = &SyntaxError{
			Line:    line,
			Column:  column,
			Message: fmt.Sprintf("Syntax error at line %d:%d \n%s", line, column, errMessage),
		}
	}
}

// ReportAmbiguity reports an ambiguity.
func (*ParseErrorListener) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	antlr.ConsoleErrorListenerINSTANCE.ReportAmbiguity(recognizer, dfa, startIndex, stopIndex, exact, ambigAlts, configs)
}

// ReportAttemptingFullContext reports an attempting full context.
func (*ParseErrorListener) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	antlr.ConsoleErrorListenerINSTANCE.ReportAttemptingFullContext(recognizer, dfa, startIndex, stopIndex, conflictingAlts, configs)
}

// ReportContextSensitivity reports a context sensitivity.
func (*ParseErrorListener) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs *antlr.ATNConfigSet) {
	antlr.ConsoleErrorListenerINSTANCE.ReportContextSensitivity(recognizer, dfa, startIndex, stopIndex, prediction, configs)
}

func addSemicolonIfNeeded(sql string) string {
	lexer := parser.NewPlSqlLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	stream.Fill()
	tokens := stream.GetAllTokens()
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].GetChannel() != antlr.TokenDefaultChannel || tokens[i].GetTokenType() == parser.PlSqlParserEOF {
			continue
		}

		// The last default channel token is a semicolon.
		if tokens[i].GetTokenType() == parser.PlSqlParserSEMICOLON {
			return sql
		}

		return stream.GetTextFromInterval(antlr.NewInterval(0, tokens[i].GetTokenIndex())) + ";"
	}
	return sql
}

// ParsePLSQL parses the given PLSQL.
func ParsePLSQL(sql string) (antlr.Tree, *antlr.CommonTokenStream, error) {
	sql = addSemicolonIfNeeded(sql)
	lexer := parser.NewPlSqlLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewPlSqlParser(stream)
	p.SetVersion12(true)

	lexerErrorListener := &ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &ParseErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true
	tree := p.Sql_script()

	if lexerErrorListener.err != nil {
		return nil, nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, nil, parserErrorListener.err
	}

	return tree, stream, nil
}

// PLSQLValidateForEditor validates the given PLSQL for editor.
func PLSQLValidateForEditor(tree antlr.Tree) error {
	l := &plsqlValidateForEditorListener{
		validate: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if !l.validate {
		return errors.New("only support SELECT sql statement")
	}
	return nil
}

type plsqlValidateForEditorListener struct {
	*parser.BasePlSqlParserListener

	validate bool
}

// EnterSql_script is called when production sql_script is entered.
func (l *plsqlValidateForEditorListener) EnterSql_script(ctx *parser.Sql_scriptContext) {
	if len(ctx.AllSql_plus_command()) > 0 {
		l.validate = false
	}
}

// EnterUnit_statement is called when production unit_statement is entered.
func (l *plsqlValidateForEditorListener) EnterUnit_statement(ctx *parser.Unit_statementContext) {
	if ctx.Data_manipulation_language_statements() == nil {
		l.validate = false
	}
}

// EnterData_manipulation_language_statements is called when production data_manipulation_language_statements is entered.
func (l *plsqlValidateForEditorListener) EnterData_manipulation_language_statements(ctx *parser.Data_manipulation_language_statementsContext) {
	if ctx.Select_statement() == nil && ctx.Explain_statement() == nil {
		l.validate = false
	}
}

// PLSQLEquivalentType returns true if the given type is equivalent to the given text.
func PLSQLEquivalentType(tp parser.IDatatypeContext, text string) (bool, error) {
	tree, _, err := ParsePLSQL(fmt.Sprintf(`CREATE TABLE t(a %s);`, text))
	if err != nil {
		return false, err
	}

	listener := &typeEquivalentListener{tp: tp, equivalent: false}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.equivalent, nil
}

type typeEquivalentListener struct {
	*parser.BasePlSqlParserListener

	tp         parser.IDatatypeContext
	equivalent bool
}

// EnterColumn_definition is called when production column_definition is entered.
func (l *typeEquivalentListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if ctx.Datatype() != nil {
		l.equivalent = equivalentType(l.tp, ctx.Datatype())
	}
}

func equivalentType(lType parser.IDatatypeContext, rType parser.IDatatypeContext) bool {
	if lType == nil || rType == nil {
		return false
	}
	lNative := lType.Native_datatype_element()
	rNative := rType.Native_datatype_element()

	if lNative != nil && rNative != nil {
		switch {
		case lNative.BINARY_INTEGER() != nil:
			return rNative.BINARY_INTEGER() != nil
		case lNative.PLS_INTEGER() != nil:
			return rNative.PLS_INTEGER() != nil
		case lNative.NATURAL() != nil:
			return rNative.NATURAL() != nil
		case lNative.BINARY_FLOAT() != nil:
			return rNative.BINARY_FLOAT() != nil
		case lNative.BINARY_DOUBLE() != nil:
			return rNative.BINARY_DOUBLE() != nil
		case lNative.NATURALN() != nil:
			return rNative.NATURALN() != nil
		case lNative.POSITIVE() != nil:
			return rNative.POSITIVE() != nil
		case lNative.POSITIVEN() != nil:
			return rNative.POSITIVEN() != nil
		case lNative.SIGNTYPE() != nil:
			return rNative.SIGNTYPE() != nil
		case lNative.SIMPLE_INTEGER() != nil:
			return rNative.SIMPLE_INTEGER() != nil
		case lNative.NVARCHAR2() != nil:
			return rNative.NVARCHAR2() != nil
		case lNative.DEC() != nil:
			return rNative.DEC() != nil
		case lNative.INTEGER() != nil:
			return rNative.INTEGER() != nil
		case lNative.INT() != nil:
			return rNative.INT() != nil
		case lNative.NUMERIC() != nil:
			return rNative.NUMERIC() != nil
		case lNative.SMALLINT() != nil:
			return rNative.SMALLINT() != nil
		case lNative.NUMBER() != nil:
			return rNative.NUMBER() != nil
		case lNative.DECIMAL() != nil:
			return rNative.DECIMAL() != nil
		case lNative.DOUBLE() != nil:
			return rNative.DOUBLE() != nil
		case lNative.FLOAT() != nil:
			return rNative.FLOAT() != nil
		case lNative.REAL() != nil:
			return rNative.REAL() != nil
		case lNative.NCHAR() != nil:
			return rNative.NCHAR() != nil
		case lNative.LONG() != nil:
			return rNative.LONG() != nil
		case lNative.CHAR() != nil:
			return rNative.CHAR() != nil
		case lNative.CHARACTER() != nil:
			return rNative.CHARACTER() != nil
		case lNative.VARCHAR2() != nil:
			return rNative.VARCHAR2() != nil
		case lNative.VARCHAR() != nil:
			return rNative.VARCHAR() != nil
		case lNative.STRING() != nil:
			return rNative.STRING() != nil
		case lNative.RAW() != nil:
			return rNative.RAW() != nil
		case lNative.BOOLEAN() != nil:
			return rNative.BOOLEAN() != nil
		case lNative.DATE() != nil:
			return rNative.DATE() != nil
		case lNative.ROWID() != nil:
			return rNative.ROWID() != nil
		case lNative.UROWID() != nil:
			return rNative.UROWID() != nil
		case lNative.YEAR() != nil:
			return rNative.YEAR() != nil
		case lNative.MONTH() != nil:
			return rNative.MONTH() != nil
		case lNative.DAY() != nil:
			return rNative.DAY() != nil
		case lNative.HOUR() != nil:
			return rNative.HOUR() != nil
		case lNative.MINUTE() != nil:
			return rNative.MINUTE() != nil
		case lNative.SECOND() != nil:
			return rNative.SECOND() != nil
		case lNative.TIMEZONE_HOUR() != nil:
			return rNative.TIMEZONE_HOUR() != nil
		case lNative.TIMEZONE_MINUTE() != nil:
			return rNative.TIMEZONE_MINUTE() != nil
		case lNative.TIMEZONE_REGION() != nil:
			return rNative.TIMEZONE_REGION() != nil
		case lNative.TIMEZONE_ABBR() != nil:
			return rNative.TIMEZONE_ABBR() != nil
		case lNative.TIMESTAMP() != nil:
			return rNative.TIMESTAMP() != nil
		case lNative.TIMESTAMP_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_UNCONSTRAINED() != nil
		case lNative.TIMESTAMP_TZ_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_TZ_UNCONSTRAINED() != nil
		case lNative.TIMESTAMP_LTZ_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_LTZ_UNCONSTRAINED() != nil
		case lNative.YMINTERVAL_UNCONSTRAINED() != nil:
			return rNative.YMINTERVAL_UNCONSTRAINED() != nil
		case lNative.DSINTERVAL_UNCONSTRAINED() != nil:
			return rNative.DSINTERVAL_UNCONSTRAINED() != nil
		case lNative.BFILE() != nil:
			return rNative.BFILE() != nil
		case lNative.BLOB() != nil:
			return rNative.BLOB() != nil
		case lNative.CLOB() != nil:
			return rNative.CLOB() != nil
		case lNative.NCLOB() != nil:
			return rNative.NCLOB() != nil
		case lNative.MLSLABEL() != nil:
			return rNative.MLSLABEL() != nil
		default:
			return false
		}
	}

	if lNative != nil || rNative != nil {
		return false
	}

	return lType.GetText() == rType.GetText()
}

// IsOracleKeyword returns true if the given text is an Oracle keyword.
func IsOracleKeyword(text string) bool {
	if len(text) == 0 {
		return false
	}

	return oracleKeywords[strings.ToUpper(text)] || oracleReservedWords[strings.ToUpper(text)]
}

func extractOracleChangedResources(currentDatabase string, currentSchema string, statement string) ([]SchemaResource, error) {
	tree, _, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	l := &plsqlChangedResourceExtractListener{
		currentDatabase: currentDatabase,
		currentSchema:   currentSchema,
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

type plsqlChangedResourceExtractListener struct {
	*parser.BasePlSqlParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]SchemaResource
}

// EnterCreate_table is called when production create_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	resource := SchemaResource{
		Database: l.currentDatabase,
		Schema:   l.currentSchema,
		Table:    PLSQLNormalizeIdentifierContext(ctx.Table_name().Identifier()),
	}

	if ctx.Schema_name() != nil {
		resource.Schema = PLSQLNormalizeIdentifierContext(ctx.Schema_name().Identifier())
	}
	l.resourceMap[resource.String()] = resource
}

// EnterDrop_table is called when production drop_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterDrop_table(ctx *parser.Drop_tableContext) {
	result := []string{PLSQLNormalizeIdentifierContext(ctx.Tableview_name().Identifier())}
	if ctx.Tableview_name().Id_expression() != nil {
		result = append(result, PLSQLNormalizeIDExpression(ctx.Tableview_name().Id_expression()))
	}
	if len(result) == 1 {
		result = []string{l.currentSchema, result[0]}
	}

	resource := SchemaResource{
		Database: l.currentDatabase,
		Schema:   result[0],
		Table:    result[1],
	}
	l.resourceMap[resource.String()] = resource
}

// EnterAlter_table is called when production alter_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	result := []string{PLSQLNormalizeIdentifierContext(ctx.Tableview_name().Identifier())}
	if ctx.Tableview_name().Id_expression() != nil {
		result = append(result, PLSQLNormalizeIDExpression(ctx.Tableview_name().Id_expression()))
	}
	if len(result) == 1 {
		result = []string{l.currentSchema, result[0]}
	}

	resource := SchemaResource{
		Database: l.currentDatabase,
		Schema:   result[0],
		Table:    result[1],
	}
	l.resourceMap[resource.String()] = resource
}

// EnterAlter_table_properties is called when production alter_table_properties is entered.
func (l *plsqlChangedResourceExtractListener) EnterAlter_table_properties(ctx *parser.Alter_table_propertiesContext) {
	if ctx.RENAME() == nil {
		return
	}
	result := []string{PLSQLNormalizeIdentifierContext(ctx.Tableview_name().Identifier())}
	if ctx.Tableview_name().Id_expression() != nil {
		result = append(result, PLSQLNormalizeIDExpression(ctx.Tableview_name().Id_expression()))
	}
	if len(result) == 1 {
		result = []string{l.currentSchema, result[0]}
	}

	resource := SchemaResource{
		Database: l.currentDatabase,
		Schema:   result[0],
		Table:    result[1],
	}
	l.resourceMap[resource.String()] = resource
}

func extractOracleResourceList(currentDatabase string, currentSchema string, statement string) ([]SchemaResource, error) {
	tree, _, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	l := &plsqlResourceExtractListener{
		currentDatabase: currentDatabase,
		currentSchema:   currentSchema,
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

type plsqlResourceExtractListener struct {
	*parser.BasePlSqlParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]SchemaResource
}

func (l *plsqlResourceExtractListener) EnterTableview_name(ctx *parser.Tableview_nameContext) {
	if ctx.Identifier() == nil {
		return
	}

	result := []string{PLSQLNormalizeIdentifierContext(ctx.Identifier())}
	if ctx.Id_expression() != nil {
		result = append(result, PLSQLNormalizeIDExpression(ctx.Id_expression()))
	}
	if len(result) == 1 {
		result = []string{l.currentSchema, result[0]}
	}

	resource := SchemaResource{
		Database: l.currentDatabase,
		Schema:   result[0],
		Table:    result[1],
	}
	l.resourceMap[resource.String()] = resource
}

// PLSQLNormalizeIdentifierContext returns the normalized identifier from the given context.
func PLSQLNormalizeIdentifierContext(identifier parser.IIdentifierContext) string {
	if identifier == nil {
		return ""
	}

	return PLSQLNormalizeIDExpression(identifier.Id_expression())
}

// PLSQLNormalizeIDExpression returns the normalized identifier from the given context.
func PLSQLNormalizeIDExpression(idExpression parser.IId_expressionContext) string {
	if idExpression == nil {
		return ""
	}

	regularID := idExpression.Regular_id()
	if regularID != nil {
		return strings.ToUpper(regularID.GetText())
	}

	delimitedID := idExpression.DELIMITED_ID()
	if delimitedID != nil {
		return strings.Trim(delimitedID.GetText(), "\"")
	}

	return ""
}
