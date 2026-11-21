package snowflake

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func buildMockDatabaseMetadataGetter(databaseMetadata []*metadatapb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			for _, metadata := range databaseMetadata {
				if metadata.Name == databaseName {
					return "", model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_SNOWFLAKE, true /* isObjectCaseSensitive */), nil
				}
			}
			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(_ context.Context, _ string) ([]string, error) {
			names := make([]string, 0, len(databaseMetadata))
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}

func TestGetQuerySpanSeam(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "DB1",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "PUBLIC",
			Tables: []*metadatapb.TableMetadata{{
				Name:    "T1",
				Columns: []*metadatapb.ColumnMetadata{{Name: "A"}, {Name: "B"}},
			}},
		}},
	}
	getter, lister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_SNOWFLAKE,
	}

	// Registry dispatch, resolveTable metadata lookup, and lineage conversion.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_SNOWFLAKE, []base.Statement{
		{Text: "SELECT A, A + B AS TOTAL FROM T1"},
	}, "DB1", "PUBLIC", false)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	require.Equal(t, base.Select, spans[0].Type)
	require.Equal(t, base.SourceColumnSet{{Database: "DB1", Schema: "PUBLIC", Table: "T1"}: true}, spans[0].SourceColumns)
	require.Equal(t, []base.QuerySpanResult{
		{
			Name:          "A",
			SourceColumns: base.SourceColumnSet{{Database: "DB1", Schema: "PUBLIC", Table: "T1", Column: "A"}: true},
			IsPlainField:  true,
		},
		{
			Name: "TOTAL",
			SourceColumns: base.SourceColumnSet{
				{Database: "DB1", Schema: "PUBLIC", Table: "T1", Column: "A"}: true,
				{Database: "DB1", Schema: "PUBLIC", Table: "T1", Column: "B"}: true,
			},
		},
	}, spans[0].Results)

	// ResourceNotFoundError conversion when callback returns *base.ResourceNotFoundError.
	missingDB := "DB1"
	nfCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, &base.ResourceNotFoundError{Database: &missingDB}
		},
		ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) { return []string{"DB1"}, nil },
		Engine:                storepb.Engine_SNOWFLAKE,
	}
	span, err := GetQuerySpan(context.Background(), nfCtx, base.Statement{Text: "SELECT A FROM T1"}, "DB1", "PUBLIC", false)
	require.NoError(t, err)
	var notFound *base.ResourceNotFoundError
	require.ErrorAs(t, span.NotFoundError, &notFound)

	// Unsynced database error.
	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, nil
		},
		ListDatabaseNamesFunc: func(context.Context, string) ([]string, error) {
			return []string{"DB1"}, nil
		},
	}, base.Statement{Text: "SELECT * FROM T1"}, "DB1", "PUBLIC", false)
	require.ErrorContains(t, err, `database metadata for database "DB1" not found (database not synced)`)

	// Fail-closed error propagation for unsupported constructs.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: `SHOW TABLES ->> SELECT * FROM T1;`}, "DB1", "PUBLIC", false)
	require.ErrorContains(t, err, "result-pipe")
}
