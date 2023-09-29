// Package parser is the parser for SQL statement.
package parser

import (
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"

	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

// FlattenExecuteStatementArgExecuteStatementArgUnnamed returns the flattened unnamed execute statement arg.
func FlattenExecuteStatementArgExecuteStatementArgUnnamed(ctx parser.IExecute_statement_argContext) []parser.IExecute_statement_arg_unnamedContext {
	var queue []parser.IExecute_statement_arg_unnamedContext
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
	tree, err := tsqlparser.ParseTSQL(selectStatement)
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
	*parser.BaseTSqlParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]SchemaResource
}

// EnterTable_source_item is called when the parser enters the table_source_item production.
func (l *tsqlReasourceExtractListener) EnterTable_source_item(ctx *parser.Table_source_itemContext) {
	if fullTableName := ctx.Full_table_name(); fullTableName != nil {
		var parts []string
		var linkedServer string
		if server := fullTableName.GetLinkedServer(); server != nil {
			linkedServer = tsqlparser.NormalizeTSQLIdentifier(server)
		}
		parts = append(parts, linkedServer)

		database := l.currentDatabase
		if d := fullTableName.GetDatabase(); d != nil {
			normalizedD := tsqlparser.NormalizeTSQLIdentifier(d)
			if normalizedD != "" {
				database = normalizedD
			}
		}
		parts = append(parts, database)

		schema := l.currentSchema
		if s := fullTableName.GetSchema(); s != nil {
			normalizedS := tsqlparser.NormalizeTSQLIdentifier(s)
			if normalizedS != "" {
				schema = normalizedS
			}
		}
		parts = append(parts, schema)

		var table string
		if t := fullTableName.GetTable(); t != nil {
			normalizedT := tsqlparser.NormalizeTSQLIdentifier(t)
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
