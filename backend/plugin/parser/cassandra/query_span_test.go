package cassandra

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetQuerySpanSeam(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "test_keyspace",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "",
			Tables: []*metadatapb.TableMetadata{{
				Name:    "users",
				Columns: []*metadatapb.ColumnMetadata{{Name: "id"}, {Name: "email"}},
			}},
		}},
	}
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
			if db == "test_keyspace" {
				return "", model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_CASSANDRA, true), nil
			}
			return "", nil, nil
		},
		Engine: storepb.Engine_CASSANDRA,
	}

	// Registry dispatch, star expansion via GetDatabaseMetadataFunc, table SourceColumns, and PredicateColumns.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_CASSANDRA, []base.Statement{
		{Text: "SELECT * FROM users WHERE id = 123"},
	}, "test_keyspace", "", false)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	require.Equal(t, base.Select, spans[0].Type)
	require.Equal(t, base.SourceColumnSet{{Database: "test_keyspace", Table: "users"}: true}, spans[0].SourceColumns)
	require.Equal(t, base.SourceColumnSet{{Database: "test_keyspace", Table: "users", Column: "id"}: true}, spans[0].PredicateColumns)
	require.Len(t, spans[0].Results, 2)
	require.Equal(t, "id", spans[0].Results[0].Name)
	require.Equal(t, "email", spans[0].Results[1].Name)

	// Without metadata, SELECT * preserves SelectAsterisk.
	span, err := GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: "SELECT * FROM users"}, "test_keyspace", "", false)
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.True(t, span.Results[0].SelectAsterisk)

	// QueryType conversion for DML and DDL.
	span, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: "INSERT INTO users (id) VALUES (1)"}, "test_keyspace", "", false)
	require.NoError(t, err)
	require.Equal(t, base.DML, span.Type)

	span, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: "CREATE TABLE users (id int PRIMARY KEY)"}, "test_keyspace", "", false)
	require.NoError(t, err)
	require.Equal(t, base.DDL, span.Type)

	// Syntax error conversion.
	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: "SELECT FROM WHERE"}, "test_keyspace", "", false)
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
}
