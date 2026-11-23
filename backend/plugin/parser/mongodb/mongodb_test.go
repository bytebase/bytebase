package mongodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseAndDiagnoseSeam(t *testing.T) {
	// Registry dispatch for ParseStatements + GetNode.
	stmts, err := base.ParseStatements(storepb.Engine_MONGODB, "db.users.find({});\ndb.products.insertOne({name: \"test\"})")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	for _, ps := range stmts {
		require.NotNil(t, ps.Start)
		require.NotNil(t, ps.End)
		require.NotNil(t, ps.Range)
		node, ok := GetNode(ps.AST)
		require.True(t, ok)
		require.NotNil(t, node)
	}
	_, ok := GetNode(nil)
	require.False(t, ok)

	// Empty input and strict syntax error.
	empty, err := base.ParseStatements(storepb.Engine_MONGODB, "   \n\t  ")
	require.NoError(t, err)
	require.Empty(t, empty)

	_, err = base.ParseStatements(storepb.Engine_MONGODB, `db.collection.find({`)
	require.Error(t, err)

	// Registry dispatch for Diagnose.
	diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_MONGODB, `db.users.find()`)
	require.NoError(t, err)
	require.Empty(t, diags)

	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_MONGODB, `db.collection.find({`)
	require.NoError(t, err)
	require.NotEmpty(t, diags)
}

func TestGetStatementRanges(t *testing.T) {
	// Registry dispatch for Bytebase-owned statement_ranges.go (UTF-16 LSP ranges).
	ranges, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_MONGODB, "db.users.find({});\ndb.products.find({})")
	require.NoError(t, err)
	require.Len(t, ranges, 2)

	empty, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_MONGODB, "")
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestGetStatementRangesWithHindiCharacters(t *testing.T) {
	// The Hindi collection name "मिलन-भेंट" has 9 UTF-16 code units (9 runes, 25 UTF-8 bytes).
	statement := "db[\"मिलन-भेंट\"].find()\ndb.coll.find()"

	ranges, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_MONGODB, statement)
	require.NoError(t, err)
	require.Len(t, ranges, 2)

	require.Equal(t, uint32(0), ranges[0].Start.Line)
	require.Equal(t, uint32(0), ranges[0].Start.Character)
	require.Equal(t, uint32(0), ranges[0].End.Line)
	require.Equal(t, uint32(22), ranges[0].End.Character)

	require.Equal(t, uint32(1), ranges[1].Start.Line)
	require.Equal(t, uint32(0), ranges[1].Start.Character)
	require.Equal(t, uint32(1), ranges[1].End.Line)
	require.Equal(t, uint32(14), ranges[1].End.Character)
}

func TestGetStatementRangesWithSurrogatePair(t *testing.T) {
	// An emoji outside the BMP counts as 2 UTF-16 code units in LSP coordinates.
	statement := "db[\"😀\"].find()"

	ranges, err := base.GetStatementRanges(context.Background(), base.StatementRangeContext{}, storepb.Engine_MONGODB, statement)
	require.NoError(t, err)
	require.Len(t, ranges, 1)
	require.Equal(t, uint32(0), ranges[0].Start.Line)
	require.Equal(t, uint32(0), ranges[0].Start.Character)
	require.Equal(t, uint32(0), ranges[0].End.Line)
	require.Equal(t, uint32(15), ranges[0].End.Character)
}
