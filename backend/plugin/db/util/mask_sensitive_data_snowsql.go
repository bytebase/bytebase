// Package util implements the util functions.
package util

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/internal/status"

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

	listener := &selectStatementListener{
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

func (l *snowsqlSensitiveFieldExtractorListener) EnterQuery_statement(ctx *snowparser.Query_statementContext) {
	if l.err != nil {
		return
	}

	// TODO(zp): handle CTE.
	// if ctx.With_expression() != nil {}

	selectStatement := ctx.Select_statement()
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsSelect_statement(ctx snowparser.ISelect_statementContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.Select_optional_clauses().From_clause() != nil {
		fromFields, err := extractor.extractSnowsqlSensitiveFieldsFrom_clause(ctx.Select_optional_clauses().From_clause())
		if err != nil {
			return nil, err
		}
		originalFromFields := extractor.fromFieldList
		extractor.fromFieldList = fromFields
		defer func() {
			extractor.fromFieldList = originalFromFields
		}()
	}

}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsFrom_clause(ctx snowparser.IFrom_clauseContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	return extractor.extractSnowsqlSensitiveFieldsTable_sources(ctx.Table_sources())
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTable_sources(ctx snowparser.ITable_sourcesContext) ([]fieldInfo, error) {
	allTableSources := ctx.AllTable_source()
	if len(allTableSources) > 1 {
		// TODO(zp): handle select from multiple table sources.
		return nil, nil
	}
	return extractor.extractSnowsqlSensitiveFieldsTable_source(allTableSources[0])
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

	if ctx.Object_ref() != nil {
		if len(ctx.AllJoin_clause()) != 0 {
			// TODO(zp): handle join in table source item.
			return nil, nil
		}
		return extractor.extractSnowsqlSensitiveFieldsObject_ref(ctx.Object_ref())
	}

	if ctx.Table_source_item_joined() != nil {
		return extractor.extractSnowsqlSensitiveFieldsTable_source_item_joined(ctx.Table_source_item_joined())
	}

	// Never reach here.
	panic("never reach here")
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsObject_ref(ctx snowparser.IObject_refContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	if objectName := ctx.Object_name(); objectName != nil {
		database, schema, table := normalizedObjectName(objectName, extractor.currentDatabase, "PUBLIC")

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

	// TODO(zp): Handle the subquery.
	if ctx.Subquery() != nil {
		return nil, nil
	}

	// TODO(zp): Handle the flatten table.
	if ctx.Flatten_table() != nil {
		return nil, nil
	}

	return nil, status.Errorf(codes.Internal, "Should be unreachable")
}

func normalizedObjectName(objectName snowparser.IObject_nameContext, fallbackDatabaseName, fallbackTableName string) (string, string, string) {
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

	schema := "PUBLIC"
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

func (extractor *sensitiveFieldExtractor) snowsqlFindTableSchema(normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) (db.TableSchema, error) {
	for _, databaseSchema := range extractor.schemaInfo.DatabaseList {
		if databaseSchema.Name != normalizedDatabaseName {
			continue
		}
		for _, tableName := range databaseSchema.TableList {

		}
	}
}
