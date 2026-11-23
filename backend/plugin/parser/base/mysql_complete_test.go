package base

import (
	"context"
	"errors"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestMySQLFamilyDatabaseLoaders(t *testing.T) {
	loadNil, listNil := MySQLFamilyDatabaseLoaders(CompletionContext{})
	require.Nil(t, loadNil)
	require.Nil(t, listNil)

	wantProto := &metadatapb.DatabaseSchemaMetadata{Name: "appdb"}
	var gotInstID, gotDB string
	var gotListInstID string
	cCtx := CompletionContext{
		InstanceID: "inst-1",
		Metadata: func(_ context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
			gotInstID = instanceID
			gotDB = databaseName
			switch databaseName {
			case "appdb":
				return "AppDB", model.NewDatabaseMetadata(wantProto, nil, nil, storepb.Engine_MYSQL, false), nil
			case "errdb":
				return "", nil, errors.New("boom")
			default:
				return "", nil, nil
			}
		},
		ListDatabaseNames: func(_ context.Context, instanceID string) ([]string, error) {
			gotListInstID = instanceID
			return []string{"appdb", "otherdb"}, nil
		},
	}

	loadDB, listDBs := MySQLFamilyDatabaseLoaders(cCtx)
	require.NotNil(t, loadDB)
	require.NotNil(t, listDBs)

	resolved, proto, caseSensitive, ok := loadDB(context.Background(), "appdb")
	require.True(t, ok)
	require.Equal(t, "inst-1", gotInstID)
	require.Equal(t, "appdb", gotDB)
	require.Equal(t, "AppDB", resolved)
	require.Equal(t, wantProto, proto)
	require.False(t, caseSensitive)

	_, _, _, ok = loadDB(context.Background(), "errdb")
	require.False(t, ok)

	_, _, _, ok = loadDB(context.Background(), "missing")
	require.False(t, ok)

	names, err := listDBs(context.Background())
	require.NoError(t, err)
	require.Equal(t, "inst-1", gotListInstID)
	require.Equal(t, []string{"appdb", "otherdb"}, names)
}
