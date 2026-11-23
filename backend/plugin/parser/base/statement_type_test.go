package base

import (
	"testing"

	omnianalysis "github.com/bytebase/omni/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestOmniStatementTypeCoversAllStorepbValues(t *testing.T) {
	require.Equal(t, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED, OmniStatementType(omnianalysis.StatementTypeUnspecified))
	require.Equal(t, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED, OmniStatementType(omnianalysis.StatementType(999)))

	mapped := make(map[storepb.StatementType]bool, len(omniStatementTypeMap))
	for omniType, pbType := range omniStatementTypeMap {
		require.Equal(t, pbType, OmniStatementType(omniType))
		require.Equal(t, pbType.String(), omniType.String(), "omni %v and storepb %v names should match", omniType, pbType)
		mapped[pbType] = true
	}

	for num, name := range storepb.StatementType_name {
		pbType := storepb.StatementType(num)
		if pbType == storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED {
			continue
		}
		require.True(t, mapped[pbType], "storepb.StatementType_%s is not mapped in omniStatementTypeMap", name)
	}
}

func TestOmniStatementTypesFiltersUnspecified(t *testing.T) {
	got := OmniStatementTypes([]omnianalysis.StatementType{
		omnianalysis.CreateTable,
		omnianalysis.StatementTypeUnspecified,
		omnianalysis.StatementType(999),
		omnianalysis.Insert,
	})
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_INSERT,
	}, got)

	require.Nil(t, OmniStatementTypes([]omnianalysis.StatementType{omnianalysis.StatementTypeUnspecified}))
}

type stubStatementAST struct {
	kind omnianalysis.StatementType
}

func (*stubStatementAST) ASTStartPosition() *storepb.Position { return nil }

func TestCollectOmniStatementTypes(t *testing.T) {
	getNode := func(a AST) (omnianalysis.StatementType, bool) {
		s, ok := a.(*stubStatementAST)
		if !ok || s == nil {
			return 0, false
		}
		return s.kind, true
	}
	classify := func(k omnianalysis.StatementType) omnianalysis.StatementType { return k }

	got, err := CollectOmniStatementTypes(
		[]AST{
			&stubStatementAST{kind: omnianalysis.CreateTable},
			&stubStatementAST{kind: omnianalysis.Insert},
			&stubStatementAST{kind: omnianalysis.CreateTable},
		},
		getNode,
		"expected stub AST",
		classify,
	)
	require.NoError(t, err)
	require.ElementsMatch(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_INSERT,
	}, got)

	_, err = CollectOmniStatementTypes([]AST{nil}, getNode, "expected stub AST", classify)
	require.EqualError(t, err, "expected stub AST")
}
