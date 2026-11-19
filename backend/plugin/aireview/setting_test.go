package aireview

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestCheckSetting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setting *storepb.AISetting
		want    string
	}{
		{name: "no setting", setting: nil, want: "AI is not enabled"},
		{name: "a new workspace", setting: &storepb.AISetting{}, want: "AI is not enabled"},
		{name: "AI on for the SQL editor only", setting: &storepb.AISetting{Enabled: true}, want: "AI SQL review is not enabled"},
		{name: "the switch on while AI is off", setting: &storepb.AISetting{SqlReviewEnabled: true}, want: "AI is not enabled"},
		{name: "both on", setting: &storepb.AISetting{Enabled: true, SqlReviewEnabled: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := CheckSetting(test.setting)
			if test.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.want)
		})
	}
}
