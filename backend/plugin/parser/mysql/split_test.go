package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitSQLSeam(t *testing.T) {
	// Registry dispatch and 1-based line/column + byte-range conversion across newlines, UTF-8, and comments.
	sql := "\n  SELECT '中';\n/* empty */;\nSELECT 2;"
	list, err := base.SplitMultiSQL(storepb.Engine_MYSQL, sql)
	require.NoError(t, err)
	filtered := base.FilterEmptyStatements(list)
	require.Len(t, filtered, 2)

	require.Equal(t, "\n  SELECT '中';", filtered[0].Text)
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, filtered[0].Start)
	require.Equal(t, &storepb.Position{Line: 2, Column: 14}, filtered[0].End)
	require.Equal(t, 0, filtered[0].BaseLine())
	require.Equal(t, filtered[0].Text, sql[filtered[0].Range.Start:filtered[0].Range.End])

	require.Equal(t, "\nSELECT 2;", filtered[1].Text)
	require.Equal(t, &storepb.Position{Line: 3, Column: 13}, filtered[1].Start)
	require.Equal(t, &storepb.Position{Line: 4, Column: 10}, filtered[1].End)
	require.Equal(t, filtered[1].Text, sql[filtered[1].Range.Start:filtered[1].Range.End])

	// DELIMITER directive round-trip keeps exact Range slicing.
	delimSQL := "DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\nDELIMITER ;\nCALL p();"
	delimList, err := SplitSQL(delimSQL)
	require.NoError(t, err)
	delimList = base.FilterEmptyStatements(delimList)
	require.Len(t, delimList, 2)
	require.Equal(t, delimList[0].Text, delimSQL[delimList[0].Range.Start:delimList[0].Range.End])
	require.Equal(t, delimList[1].Text, delimSQL[delimList[1].Range.Start:delimList[1].Range.End])
}
