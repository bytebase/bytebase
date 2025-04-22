package trino

import (
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TRINO, validateQuery)
	base.RegisterExtractResourceListFunc(storepb.Engine_TRINO, ExtractResourceList)
}

// validateQuery validates if the given SQL statement is valid for SQL editor.
// We only allow read-only queries in the SQL editor.
func validateQuery(statement string) (bool, bool, error) {
	result, err := ParseTrino(statement)
	if err != nil {
		return false, false, err
	}

	queryType, isAnalyze := getQueryType(result.Tree, false)

	// If it's an EXPLAIN ANALYZE, the query will be executed
	if isAnalyze {
		// Only allow EXPLAIN ANALYZE for SELECT statements
		if queryType == base.Select {
			return true, true, nil
		}
		return false, false, nil
	}

	// Determine if the statement is read-only
	readOnly := queryType == base.Select ||
		queryType == base.Explain ||
		queryType == base.SelectInfoSchema

	// Determine if the statement returns data
	returnsData := queryType == base.Select ||
		queryType == base.Explain ||
		queryType == base.SelectInfoSchema

	return readOnly, returnsData, nil
}

// ExtractResourceList extracts the resource list from the given SQL statement.
func ExtractResourceList(currentDatabase, currentSchema, statement string) ([]base.SchemaResource, error) {
	result, err := ParseTrino(statement)
	if err != nil {
		return nil, err
	}

	listener := &resourceExtractListener{
		currentDatabase: currentDatabase,
		currentSchema:   currentSchema,
		resourceMap:     make(map[string]base.SchemaResource),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)

	var resources []base.SchemaResource
	for _, resource := range listener.resourceMap {
		resources = append(resources, resource)
	}

	// Sort the resources for consistent output
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].String() < resources[j].String()
	})

	return resources, nil
}

// resourceExtractListener implements the TrinoParserListener interface to extract table resources.
type resourceExtractListener struct {
	parser.BaseTrinoParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]base.SchemaResource
}

// EnterTableName is called when a tableName node is entered
func (l *resourceExtractListener) EnterTableName(ctx *parser.TableNameContext) {
	if ctx.QualifiedName() == nil {
		return
	}

	// Extract the database, schema, and table name
	catalog, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.currentDatabase,
		l.currentSchema,
	)

	resource := base.SchemaResource{
		Database: catalog,
		Schema:   schema,
		Table:    table,
	}

	l.resourceMap[resource.String()] = resource
}
