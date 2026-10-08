package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestIsBuiltinSemanticTypeID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{id: defaultSemanticTypeID, want: true},
		{id: defaultPartialSemanticTypeID, want: true},
		{id: "email", want: false},
		{id: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			if got := isBuiltinSemanticTypeID(test.id); got != test.want {
				t.Fatalf("isBuiltinSemanticTypeID(%q) = %v, want %v", test.id, got, test.want)
			}
		})
	}
}

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
