package snowflake

import (
	"context"

	"github.com/bytebase/omni/snowflake/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_SNOWFLAKE, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, schema string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	span, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		ResolveTable: func(database, schema, table string) (*analysis.Table, error) {
			return resolveTable(ctx, gCtx, database, schema, table)
		},
	})
	if err != nil {
		if errors.Is(err, analysis.ErrMixUserSystemTables) {
			return nil, base.MixUserSystemTablesError
		}
		var resolveErr *analysis.ResolveError
		if !errors.As(err, &resolveErr) {
			return nil, err
		}
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(resolveErr.Err, &resourceNotFound) {
			return &base.QuerySpan{
				SourceColumns: convertColumnSet(resolveErr.SourceColumns),
				Results:       []base.QuerySpanResult{},
				NotFoundError: resourceNotFound,
			}, nil
		}
		return nil, resolveErr.Err
	}
	if span == nil {
		return nil, nil
	}
	return &base.QuerySpan{
		Type:          convertQueryType(span.Type),
		SourceColumns: convertColumnSet(span.SourceColumns),
		Results:       convertResults(span.Results),
	}, nil
}

// resolveTable finds a table in the synced metadata of the instance's
// databases. An empty database name searches every database.
func resolveTable(ctx context.Context, gCtx base.GetQuerySpanContext, normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) (*analysis.Table, error) {
	allDatabases, err := gCtx.ListDatabaseNamesFunc(ctx, gCtx.InstanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get all databases")
	}

	for _, databaseName := range allDatabases {
		if normalizedDatabaseName != "" && normalizedDatabaseName != databaseName {
			continue
		}
		_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, normalizedDatabaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %s", normalizedDatabaseName)
		}
		if database == nil {
			return nil, errors.Errorf("database metadata for database %q not found (database not synced)", normalizedDatabaseName)
		}
		allSchemaNames := database.ListSchemaNames()
		for _, schemaSchema := range allSchemaNames {
			if normalizedSchemaName != "" && normalizedSchemaName != schemaSchema {
				continue
			}
			schema := database.GetSchemaMetadata(normalizedSchemaName)
			if schema == nil {
				return nil, errors.Errorf(`schema %s.%s is not found`, normalizedDatabaseName, normalizedSchemaName)
			}
			allTableNames := schema.ListTableNames()
			for _, tableName := range allTableNames {
				if normalizedTableName != tableName {
					continue
				}
				tableSchema := schema.GetTable(normalizedTableName)
				if tableSchema == nil {
					return nil, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				}
				var columns []string
				for _, column := range tableSchema.GetProto().GetColumns() {
					columns = append(columns, column.Name)
				}
				return &analysis.Table{
					Name:    tableSchema.GetProto().Name,
					Columns: columns,
				}, nil
			}
		}
	}
	return nil, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

func convertQueryType(t analysis.QueryType) base.QueryType {
	return base.ConvertQueryType(t, analysis.Select, analysis.Explain, analysis.SelectInfoSchema, analysis.DDL, analysis.DML)
}

func convertColumnSet(set analysis.ColumnSet) base.SourceColumnSet {
	return base.ConvertColumnSet(set, func(c analysis.ColumnResource) base.ColumnResource {
		return base.ColumnResource{Database: c.Database, Schema: c.Schema, Table: c.Table, Column: c.Column}
	})
}

func convertResults(results []analysis.QuerySpanResult) []base.QuerySpanResult {
	return base.ConvertQuerySpanResults(results, func(r analysis.QuerySpanResult) base.QuerySpanResult {
		return base.QuerySpanResult{
			Name:          r.Name,
			SourceColumns: convertColumnSet(r.SourceColumns),
			IsPlainField:  r.IsPlainField,
		}
	})
}
