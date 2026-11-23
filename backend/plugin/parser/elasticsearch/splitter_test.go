package elasticsearch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitAndDiagnoseSeam(t *testing.T) {
	empty, err := base.SplitMultiSQL(storepb.Engine_ELASTICSEARCH, "   \n  ")
	require.NoError(t, err)
	require.Nil(t, empty)

	stmts, err := base.SplitMultiSQL(storepb.Engine_ELASTICSEARCH, "GET /users/_search\n{}\n\nGET /orders/_doc/1")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, "GET /users/_search\n{}", stmts[0].Text)
	require.Equal(t, int32(1), stmts[0].Start.Line)
	require.Equal(t, int32(1), stmts[0].Start.Column)
	require.Equal(t, "GET /orders/_doc/1", stmts[1].Text)
	require.Equal(t, int32(4), stmts[1].Start.Line)
	require.NotNil(t, stmts[1].Range)

	validDiags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_ELASTICSEARCH, "GET /users/_search\n{}")
	require.NoError(t, err)
	require.Empty(t, validDiags)

	invalidDiags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_ELASTICSEARCH, "db.users.find({})")
	require.NoError(t, err)
	require.NotEmpty(t, invalidDiags)
	require.Equal(t, "Syntax check", invalidDiags[0].Source)
}
