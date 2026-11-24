package redshift

import (
	"context"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func redshiftMockDatabaseMetadataGetter(databaseMetadata []*metadatapb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			for _, metadata := range databaseMetadata {
				if metadata.GetName() == databaseName {
					return "", model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_REDSHIFT, true /* isObjectCaseSensitive */), nil
				}
			}
			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(_ context.Context, _ string) ([]string, error) {
			names := make([]string, 0, len(databaseMetadata))
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.GetName())
			}
			return names, nil
		}
}

func TestGetQuerySpanSeam(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name:       "db",
		SearchPath: "analytics, public",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{{
					Name:    "orders",
					Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}, {Name: "status", Type: "varchar"}},
				}},
				Functions: []*metadatapb.FunctionMetadata{{
					Name:       "unsupported_fn",
					Definition: "CREATE FUNCTION public.unsupported_fn() RETURNS int STABLE AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql;",
					Signature:  "unsupported_fn()",
				}},
			},
			{
				Name: "analytics",
				Tables: []*metadatapb.TableMetadata{{
					Name:    "events",
					Columns: []*metadatapb.ColumnMetadata{{Name: "event_id", Type: "int"}},
				}},
			},
		},
	}
	getter, lister := redshiftMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_REDSHIFT,
	}

	// Registry dispatch, SearchPath from metadata, result lineage, and predicate columns.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_REDSHIFT, []base.Statement{
		{Text: "SELECT event_id FROM events"},
	}, "db", "", false)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	require.Equal(t, base.Select, spans[0].Type)
	require.Equal(t, []base.QuerySpanResult{{
		Name:          "event_id",
		SourceColumns: base.SourceColumnSet{{Database: "db", Schema: "analytics", Table: "events", Column: "event_id"}: true},
		IsPlainField:  true,
	}}, spans[0].Results)

	// Explicit schema argument overrides metadata SearchPath + PredicateColumns conversion.
	span, err := GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT id FROM orders WHERE status = 'open'"}, "db", "public", false)
	require.NoError(t, err)
	require.Equal(t, base.SourceColumnSet{{Database: "db", Schema: "public", Table: "orders", Column: "status"}: true}, span.PredicateColumns)

	// ResourceNotFoundError conversion.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT id FROM missing_orders"}, "db", "public", false)
	require.NoError(t, err)
	var notFound *base.ResourceNotFoundError
	require.ErrorAs(t, span.NotFoundError, &notFound)

	// FunctionNotSupportedError conversion.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT unsupported_fn()"}, "db", "public", false)
	require.NoError(t, err)
	var unsupported *base.FunctionNotSupportedError
	require.ErrorAs(t, span.FunctionNotSupportedError, &unsupported)

	// MixUserSystemTablesError conversion.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT orders.id FROM orders, pg_class"}, "db", "public", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// SyntaxError position offset using Statement.Start.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{
		Text:  "SELECT * FORM orders;",
		Start: &storepb.Position{Line: 7, Column: 1},
	}, "db", "public", false)
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, int32(7), syntaxErr.Position.Line)

	// Unsynced database & nil callback guards.
	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: "SELECT 1"}, "db", "", false)
	require.ErrorContains(t, err, "GetDatabaseMetadataFunc is not set")

	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, nil
		},
	}, base.Statement{Text: "SELECT id FROM orders"}, "db", "", false)
	require.ErrorContains(t, err, `database metadata for database "db" not found (database not synced)`)
}

func TestRedshiftQuerySpanEntrypointDoesNotDependOnANTLR(t *testing.T) {
	content, err := os.ReadFile("redshift.go")
	require.NoError(t, err)
	source := string(content)
	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/redshift")
}
