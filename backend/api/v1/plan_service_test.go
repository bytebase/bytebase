package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func cdcSpec(id, sheet string, targets []string, priorBackup bool) *storepb.PlanConfig_Spec {
	return &storepb.PlanConfig_Spec{
		Id: id,
		Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
			ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
				SheetSha256:       sheet,
				Targets:           targets,
				EnablePriorBackup: priorBackup,
			},
		},
	}
}

func TestPlanSpecsEqualSet(t *testing.T) {
	cases := []struct {
		name string
		a, b []*storepb.PlanConfig_Spec
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "identical single spec",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			want: true,
		},
		{
			name: "same set reordered",
			a: []*storepb.PlanConfig_Spec{
				cdcSpec("s1", "sha1", []string{"db1"}, false),
				cdcSpec("s2", "sha2", []string{"db2"}, false),
			},
			b: []*storepb.PlanConfig_Spec{
				cdcSpec("s2", "sha2", []string{"db2"}, false),
				cdcSpec("s1", "sha1", []string{"db1"}, false),
			},
			want: true,
		},
		{
			name: "added spec",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b: []*storepb.PlanConfig_Spec{
				cdcSpec("s1", "sha1", []string{"db1"}, false),
				cdcSpec("s2", "sha2", []string{"db2"}, false),
			},
			want: false,
		},
		{
			name: "removed spec",
			a: []*storepb.PlanConfig_Spec{
				cdcSpec("s1", "sha1", []string{"db1"}, false),
				cdcSpec("s2", "sha2", []string{"db2"}, false),
			},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			want: false,
		},
		{
			name: "same id sheet differs",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha2", []string{"db1"}, false)},
			want: false,
		},
		{
			name: "same id targets differ",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1", "db2"}, false)},
			want: false,
		},
		{
			name: "same id prior_backup differs",
			a:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, false)},
			b:    []*storepb.PlanConfig_Spec{cdcSpec("s1", "sha1", []string{"db1"}, true)},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, planSpecsEqualSet(tc.a, tc.b))
		})
	}
}
