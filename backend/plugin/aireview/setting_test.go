package aireview

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestCheckSetting(t *testing.T) {
	t.Parallel()

	on, off := true, false
	tests := []struct {
		name    string
		setting *storepb.AISetting
		project *bool
		want    string
	}{
		{name: "no setting", setting: nil, want: "AI is not enabled"},
		{name: "a new workspace", setting: &storepb.AISetting{}, want: "AI is not enabled"},
		{name: "AI on for the SQL editor only", setting: &storepb.AISetting{Enabled: true}, want: "AI SQL review is not enabled"},
		{name: "the switch on while AI is off", setting: &storepb.AISetting{SqlReviewEnabled: true}, want: "AI is not enabled"},
		{name: "both on", setting: &storepb.AISetting{Enabled: true, SqlReviewEnabled: true}},
		{name: "a project on while the workspace is off", setting: &storepb.AISetting{Enabled: true}, project: &on},
		{name: "a project off while the workspace is on", setting: &storepb.AISetting{Enabled: true, SqlReviewEnabled: true}, project: &off, want: "turned off for the project"},
		{name: "a project on while AI is off", setting: &storepb.AISetting{SqlReviewEnabled: true}, project: &on, want: "AI is not enabled"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := CheckSetting(test.setting, test.project)
			if test.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.want)
		})
	}
}
