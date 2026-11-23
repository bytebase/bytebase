package tsql

import (
	"context"
	"errors"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mssql/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func strPtr(s string) *string { return &s }

func TestGetQuerySpanSeam(t *testing.T) {
	dbMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "name", Type: "nvarchar"},
						},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{Name: "v_users", Definition: "SELECT id FROM dbo.users"},
				},
			},
		},
	}, nil, nil, storepb.Engine_MSSQL, false)

	gCtx := base.GetQuerySpanContext{
		InstanceID: "inst-1",
		TempTables: make(map[string]*base.PhysicalTable),
		PlanOutput: make(base.PlanOutput),
		ListDatabaseNamesFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"app"}, nil
		},
		GetDatabaseMetadataFunc: func(_ context.Context, _, dbName string) (string, *model.DatabaseMetadata, error) {
			switch dbName {
			case "app":
				return dbName, dbMeta, nil
			case "missing":
				return "", nil, &base.ResourceNotFoundError{Err: errors.New("missing db"), Database: strPtr("missing")}
			default:
				return "", nil, nil
			}
		},
	}

	// Table and predicate column mapping via registry.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_MSSQL, []base.Statement{{Text: "SELECT name FROM dbo.users WHERE id = 1;"}}, "app", "dbo", true)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, base.Select, span.Type)
	require.Len(t, span.Results, 1)
	require.True(t, span.Results[0].SourceColumns[base.ColumnResource{Database: "app", Schema: "dbo", Table: "users", Column: "name"}])
	require.True(t, span.PredicateColumns[base.ColumnResource{Database: "app", Schema: "dbo", Table: "users", Column: "id"}])

	// TempTables round-trip across statements in the same batch.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "CREATE TABLE #tmp (c1 INT);"}, "app", "dbo", true)
	require.NoError(t, err)
	require.Contains(t, gCtx.TempTables, "#tmp")

	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT c1 FROM #tmp;"}, "app", "dbo", true)
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.Equal(t, "c1", span.Results[0].Name)

	// ResourceNotFoundError returned by metadata callback maps to span.NotFoundError.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT id FROM missing.dbo.users;"}, "app", "dbo", true)
	require.NoError(t, err)
	var notFoundErr *base.ResourceNotFoundError
	require.ErrorAs(t, span.NotFoundError, &notFoundErr)
	require.Equal(t, strPtr("missing"), notFoundErr.Database)

	// PlanOutput state persists across statements.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SET SHOWPLAN_ALL ON;"}, "app", "dbo", true)
	require.NoError(t, err)
	require.NotEmpty(t, gCtx.PlanOutput)
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT id FROM dbo.users;"}, "app", "dbo", true)
	require.NoError(t, err)
	require.Equal(t, base.Explain, span.Type)
	clear(gCtx.PlanOutput)

	// Mixing user and system tables maps to base.MixUserSystemTablesError.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT u.id, t.name FROM dbo.users u JOIN sys.tables t ON 1=1;"}, "app", "dbo", true)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// Error wrapper conversions for TypeNotSupportedError and ResourceNotFoundError.
	var typeErr *base.TypeNotSupportedError
	require.ErrorAs(t, convertQuerySpanError(&analysis.TypeNotSupportedError{Type: "xml", Name: "x"}), &typeErr)
	require.Equal(t, "xml", typeErr.Type)

	var nfErr *base.ResourceNotFoundError
	require.ErrorAs(t, convertQuerySpanError(&analysis.NotFoundError{Database: strPtr("app"), Table: strPtr("ghost")}), &nfErr)
	require.Equal(t, strPtr("ghost"), nfErr.Table)

	require.Equal(t, base.SelectInfoSchema, base.OmniQueryType(analysis.SelectInfoSchema))
	require.Equal(t, base.DDL, base.OmniQueryType(analysis.DDL))
	require.Equal(t, base.DML, base.OmniQueryType(analysis.DML))
	require.Equal(t, base.QueryTypeUnknown, base.OmniQueryType(analysis.QueryType(999)))
}
