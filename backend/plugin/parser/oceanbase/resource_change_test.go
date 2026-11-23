package oceanbase

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResourcesSeam(t *testing.T) {
	statement := "CREATE TABLE t1 (c1 INT);\nALTER TABLE t1 ADD COLUMN c2 INT;\nINSERT INTO t1 (c1) VALUES (1), (2);\nUPDATE t1 SET c1 = 3;\nCREATE VIEW other_db.v AS SELECT 1;"

	stmts, err := base.ParseStatements(storepb.Engine_OCEANBASE, statement)
	require.NoError(t, err)

	got, err := base.ExtractChangedResources(storepb.Engine_OCEANBASE, "db", "", nil, base.ExtractASTs(stmts), statement)
	require.NoError(t, err)

	wantResources := model.NewChangedResources(nil)
	wantResources.AddTable("db", "", &storepb.ChangedResourceTable{Name: "t1"}, true)
	wantResources.AddDatabase("other_db")

	require.Equal(t, wantResources, got.ChangedResources)
	require.Equal(t, []string{"UPDATE t1 SET c1 = 3;"}, got.DMLStatements)
	require.Equal(t, 1, got.DMLCount)
	require.Equal(t, 2, got.InsertCount)
}
