package snowflake

import (
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/snowsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractResourceListFunc(storepb.Engine_SNOWFLAKE, ExtractResourceList)
	base.RegisterExtractDatabaseListFunc(storepb.Engine_SNOWFLAKE, ExtractDatabaseList)
}

type snowsqlResourceExtractListener struct {
	*parser.BaseSnowflakeParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]base.SchemaResource
}

func (l *snowsqlResourceExtractListener) EnterObject_ref(ctx *parser.Object_refContext) {
	objectName := ctx.Object_name()
	if objectName == nil {
		return
	}

	var parts []string
	database := l.currentDatabase
	if d := objectName.GetD(); d != nil {
		normalizedD := NormalizeSnowSQLObjectNamePart(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	schema := l.currentSchema
	if s := objectName.GetS(); s != nil {
		normalizedS := NormalizeSnowSQLObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)

	var table string
	if o := objectName.GetO(); o != nil {
		normalizedO := NormalizeSnowSQLObjectNamePart(o)
		if normalizedO != "" {
			table = normalizedO
		}
	}
	parts = append(parts, table)

	normalizedObjectName := strings.Join(parts, ".")
	l.resourceMap[normalizedObjectName] = base.SchemaResource{
		Database: database,
		Schema:   schema,
		Table:    table,
	}
}

// ExtractDatabaseList extracts all databases from statement, and normalizes the database name.
// If the database name is not specified, it will fallback to the normalizedDatabaseName.
func ExtractDatabaseList(statement string, normalizedDatabaseName string) ([]string, error) {
	schemaPlaceholder := "schema_placeholder"
	schemaResource, err := ExtractResourceList(normalizedDatabaseName, schemaPlaceholder, statement)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, resource := range schemaResource {
		result = append(result, resource.Database)
	}
	return result, nil
}

// ExtractResourceList extracts the list of resources from the SELECT statement, and normalizes the object names with the NON-EMPTY currentNormalizedDatabase and currentNormalizedSchema.
func ExtractResourceList(currentDatabase string, currentSchema string, selectStatement string) ([]base.SchemaResource, error) {
	tree, err := ParseSnowSQL(selectStatement)
	if err != nil {
		return nil, err
	}

	l := &snowsqlResourceExtractListener{
		currentDatabase: currentDatabase,
		currentSchema:   currentSchema,
		resourceMap:     make(map[string]base.SchemaResource),
	}

	var result []base.SchemaResource
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result, nil
}
