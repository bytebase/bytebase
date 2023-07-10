// Package parser is the parser for SQL statement.
package parser

import (
	"sort"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/snowsql-parser"
)

// ParseSnowSQL parses the given SQL statement by using antlr4. Returns the AST and token stream if no error.
func ParseSnowSQL(statement string) (antlr.Tree, error) {
	statement = strings.TrimRight(statement, " \t\n\r\f;") + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewSnowflakeLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewSnowflakeParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &ParseErrorListener{}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &ParseErrorListener{}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Snowflake_file()

	if lexerErrorListener.err != nil {
		return nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, parserErrorListener.err
	}

	return tree, nil
}

var snowflakeKeyword = map[string]bool{
	"ACCOUNT": true,
	"ALL":     true,
	"ALTER":   true,
	"AND":     true,
	"ANY":     true,
	"AS":      true,

	"BETWEEN": true,
	"BY":      true,

	"CASE":              true,
	"CAST":              true,
	"CHECK":             true,
	"COLUMN":            true,
	"CONNECT":           true,
	"CONNECTION":        true,
	"CONSTRAINT":        true,
	"CREATE":            true,
	"CROSS":             true,
	"CURRENT":           true,
	"CURRENT_DATE":      true,
	"CURRENT_TIME":      true,
	"CURRENT_TIMESTAMP": true,
	"CURRENT_USER":      true,

	"DATABASE": true,
	"DELETE":   true,
	"DISTINCT": true,
	"DROP":     true,

	"ELSE":   true,
	"EXISTS": true,

	"FALSE":     true,
	"FOLLOWING": true,
	"FOR":       true,
	"FROM":      true,
	"FULL":      true,

	"GRANT":     true,
	"GROUP":     true,
	"GSCLUSTER": true,

	"HAVING": true,

	"ILIKE":     true,
	"IN":        true,
	"INCREMENT": true,
	"INNER":     true,
	"INSERT":    true,
	"INTERSECT": true,
	"INTO":      true,
	"IS":        true,
	"ISSUE":     true,

	"JOIN": true,

	"LATERAL":        true,
	"LEFT":           true,
	"LIKE":           true,
	"LOCALTIME":      true,
	"LOCALTIMESTAMP": true,

	"MINUS": true,

	"NATURAL": true,
	"NOT":     true,
	"NULL":    true,

	"OF":           true,
	"ON":           true,
	"OR":           true,
	"ORDER":        true,
	"ORGANIZATION": true,

	"QUALIFY": true,

	"REGEXP": true,
	"REVOKE": true,
	"RIGHT":  true,
	"RLIKE":  true,
	"ROW":    true,
	"ROWS":   true,

	"SAMPLE": true,
	"SCHEMA": true,
	"SELECT": true,
	"SET":    true,
	"SOME":   true,
	"START":  true,

	"TABLE":       true,
	"TABLESAMPLE": true,
	"THEN":        true,
	"TO":          true,
	"TRIGGER":     true,
	"TRUE":        true,
	"TRY_CAST":    true,

	"UNION":  true,
	"UNIQUE": true,
	"UPDATE": true,
	"USING":  true,

	"VALUES": true,
	"VIEW":   true,

	"WHEN":     true,
	"WHENEVER": true,
	"WHERE":    true,
	"WITH":     true,
}

// IsSnowflakeKeyword returns true if the given string is a snowflake keyword.
// Follows https://docs.snowflake.com/en/sql-reference/reserved-keywords.
func IsSnowflakeKeyword(s string, caseSensitive bool) bool {
	if !caseSensitive {
		s = strings.ToUpper(s)
	}
	return snowflakeKeyword[s]
}

type snowsqlResourceExtractListener struct {
	*parser.BaseSnowflakeParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]SchemaResource
}

func (l *snowsqlResourceExtractListener) EnterObject_ref(ctx *parser.Object_refContext) {
	objectName := ctx.Object_name()
	if objectName == nil {
		return
	}

	var parts []string
	database := l.currentDatabase
	if d := objectName.GetD(); d != nil {
		normalizedD := NormalizeObjectNamePart(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	schema := l.currentSchema
	if s := objectName.GetS(); s != nil {
		normalizedS := NormalizeObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)

	var table string
	if o := objectName.GetO(); o != nil {
		normalizedO := NormalizeObjectNamePart(o)
		if normalizedO != "" {
			table = normalizedO
		}
	}
	parts = append(parts, table)

	normalizedObjectName := strings.Join(parts, ".")
	l.resourceMap[normalizedObjectName] = SchemaResource{
		Database: database,
		Schema:   schema,
		Table:    table,
	}
}

// extractSnowflakeNormalizeResourceListFromSelectStatement extracts the list of resources from the SELECT statement, and normalizes the object names with the NON-EMPTY currentNormalizedDatabase and currentNormalizedSchema.
func extractSnowflakeNormalizeResourceListFromSelectStatement(currentNormalizedDatabase string, currentNormalizedSchema string, selectStatement string) ([]SchemaResource, error) {
	tree, err := ParseSnowSQL(selectStatement)
	if err != nil {
		return nil, err
	}

	l := &snowsqlResourceExtractListener{
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

// NormalizeObjectName normalizes the given object name.
func NormalizeObjectName(objectName parser.IObject_nameContext, fallbackDatabaseName, fallbackSchemaName string) string {
	var parts []string

	database := fallbackDatabaseName
	if d := objectName.GetD(); d != nil {
		normalizedD := NormalizeObjectNamePart(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	schema := fallbackSchemaName
	if s := objectName.GetS(); s != nil {
		normalizedS := NormalizeObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)

	if o := objectName.GetO(); o != nil {
		normalizedO := NormalizeObjectNamePart(o)
		if normalizedO != "" {
			parts = append(parts, normalizedO)
		}
	}
	return strings.Join(parts, ".")
}

// NormalizeSchemaName normalizes the given schema name.
func NormalizeSchemaName(schemaName parser.ISchema_nameContext, fallbackDatabaseName string) string {
	ids := schemaName.AllId_()

	var parts []string
	database := fallbackDatabaseName
	if len(ids) == 2 {
		normalizedD := NormalizeObjectNamePart(ids[0])
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	var schema string
	if len(ids) == 2 {
		normalizedS := NormalizeObjectNamePart(ids[1])
		if normalizedS != "" {
			schema = normalizedS
		}
	} else {
		normalizedS := NormalizeObjectNamePart(ids[0])
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)
	return strings.Join(parts, ".")
}

// NormalizeObjectNamePart normalizes the object name part.
func NormalizeObjectNamePart(part parser.IId_Context) string {
	if part == nil {
		return ""
	}
	return ExtractOrdinaryIdentifier(part.GetText())
}

// ExtractOrdinaryIdentifier extracts the ordinary object name from a string. It follows the following rules:
//
// 1. If there are no double quotes on either side, it will be converted to uppercase.
//
// 2. If there are double quotes on both sides, the case will not change, the double quotes on both sides will be removed, and `""` in content will be converted to `"`.
//
// Caller MUST ensure the identifier is VALID.
func ExtractOrdinaryIdentifier(identifier string) string {
	quoted := strings.HasPrefix(identifier, `"`) && strings.HasSuffix(identifier, `"`)
	if quoted {
		identifier = identifier[1 : len(identifier)-1]
	}
	runeObjectName := []rune(identifier)
	var result []rune
	for i := 0; i < len(runeObjectName); i++ {
		newRune := runeObjectName[i]
		if i+1 < len(runeObjectName) && runeObjectName[i] == '"' && runeObjectName[i+1] == '"' && quoted {
			newRune = '"'
			i++
		} else if !quoted {
			newRune = unicode.ToUpper(newRune)
		}
		result = append(result, newRune)
	}
	return string(result)
}
