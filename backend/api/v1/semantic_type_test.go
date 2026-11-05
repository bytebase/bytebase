package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestAppendBuiltinSemanticTypes(t *testing.T) {
	configured := &storepb.SemanticTypeSetting{
		Types: []*storepb.SemanticTypeSetting_SemanticType{
			{Id: "email", Title: "Email"},
			{Id: defaultSemanticTypeID, Title: "Stored default"},
		},
	}

	got := appendBuiltinSemanticTypes(configured)

	require.Equal(t, []string{"email", defaultSemanticTypeID, defaultPartialSemanticTypeID}, semanticTypeIDs(got))
	require.Equal(t, "Default", got.Types[1].Title)
	require.Equal(t, []string{"email", defaultSemanticTypeID}, semanticTypeIDs(configured))
}

func semanticTypeIDs(setting *storepb.SemanticTypeSetting) []string {
	ids := make([]string, 0, len(setting.GetTypes()))
	for _, semanticType := range setting.GetTypes() {
		ids = append(ids, semanticType.GetId())
	}
	return ids
}
