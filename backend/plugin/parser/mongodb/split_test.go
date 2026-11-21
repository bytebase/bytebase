package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQLSeam(t *testing.T) {
	empty, err := base.SplitMultiSQL(storepb.Engine_MONGODB, "   \n\t  ")
	require.NoError(t, err)
	require.Nil(t, empty)

	stmts, err := base.SplitMultiSQL(storepb.Engine_MONGODB, "show dbs;\ndb.users.find({})\ndb.products.drop()")
	require.NoError(t, err)
	require.Len(t, stmts, 3)
	require.Equal(t, "show dbs;", stmts[0].Text)
	require.Equal(t, "db.users.find({})", stmts[1].Text)
	require.Equal(t, "db.products.drop()", stmts[2].Text)
	for _, s := range stmts {
		require.NotNil(t, s.Start)
		require.NotNil(t, s.End)
		require.NotNil(t, s.Range)
		require.Greater(t, s.Range.End, s.Range.Start)
	}

	// Strict splitting fails the whole batch on any unparseable statement (BYT-9950).
	_, err = base.SplitMultiSQL(storepb.Engine_MONGODB, "db.users.find();\nthis is not mongosh")
	require.Error(t, err)
}
