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
}

// ExtractResourceList extracts the list of resources from the SELECT statement, and normalizes the object names with the NON-EMPTY currentNormalizedDatabase and currentNormalizedSchema.
func ExtractResourceList(currentDatabase string, currentSchema string, selectStatement string) ([]base.SchemaResource, error) {
	parseResult, err := ParseSnowSQL(selectStatement)
	if err != nil {
		return nil, err
	}
	if parseResult == nil {
		return nil, nil
	}

	l := &resourceExtractListener{
		currentDatabase: currentDatabase,
		currentSchema:   currentSchema,
		resourceMap:     make(map[string]base.SchemaResource),
	}

	var result []base.SchemaResource
	antlr.ParseTreeWalkerDefault.Walk(l, parseResult.Tree)
	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result, nil
}

type resourceExtractListener struct {
	*parser.BaseSnowflakeParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]base.SchemaResource
}

func (l *resourceExtractListener) EnterObject_ref(ctx *parser.Object_refContext) {
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
