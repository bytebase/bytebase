package snowflake

import (
	"context"

	"github.com/bytebase/omni/snowflake/analysis"
	omniast "github.com/bytebase/omni/snowflake/ast"
	sfcompletion "github.com/bytebase/omni/snowflake/completion"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, sfcompletion.Engine)
	engine.QuerySpan = GetQuerySpan
	base.Register(engine, storepb.Engine_SNOWFLAKE)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (omniast.Node, bool) {
	node, ok := base.ExtractNode[omniast.Node](a)
	return node, ok && node != nil
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
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		var resolveErr *analysis.ResolveError
		if !errors.As(err, &resolveErr) {
			return nil, err
		}
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(resolveErr.Err, &resourceNotFound) {
			return &base.QuerySpan{
				SourceColumns: base.OmniColumnSet(resolveErr.SourceColumns),
				Results:       []base.QuerySpanResult{},
				NotFoundError: resourceNotFound,
			}, nil
		}
		return nil, resolveErr.Err
	}
	return base.OmniResolvedSpan(span), nil
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
