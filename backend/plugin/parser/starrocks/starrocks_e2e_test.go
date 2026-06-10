package starrocks

import (
	"context"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestStarRocksDiagnoseE2E is the PR1 keystone (BYT-9140): it proves the
// Engine_STARROCKS parse path is re-routed from the doris parser to the
// omni/starrocks fork end-to-end, through the Bytebase diagnose dispatcher.
//
//   - value: a StarRocks generated column `AS (expr)` (which doris rejects as a
//     false syntax error) now diagnoses clean.
//   - zero-regression: shared-core statements still diagnose clean.
//   - diagnose still works: a genuinely-invalid statement still reports an error.
func TestStarRocksDiagnoseE2E(t *testing.T) {
	cases := []struct {
		name     string
		sql      string
		wantDiag bool
	}{
		{
			"generated_column_as", // value: doris rejected this; the fork accepts it
			"CREATE TABLE t (a INT, b INT AS (a + 1)) DUPLICATE KEY(a) DISTRIBUTED BY HASH(a)",
			false,
		},
		{
			"shared_core_create_table", // zero-regression
			"CREATE TABLE t (k INT, v VARCHAR(20)) DUPLICATE KEY(k) DISTRIBUTED BY HASH(k) BUCKETS 1",
			false,
		},
		{
			"shared_core_select", // zero-regression
			"SELECT a, b FROM t WHERE a > 0",
			false,
		},
		{
			"invalid_statement", // diagnose still catches real syntax errors
			"SELECT FROM WHERE",
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_STARROCKS, tc.sql)
			if err != nil {
				t.Fatalf("Diagnose error: %v", err)
			}
			if got := len(diags) > 0; got != tc.wantDiag {
				t.Fatalf("Diagnose(%q): hasDiagnostic=%v, want %v (diags=%+v)", tc.sql, got, tc.wantDiag, diags)
			}
		})
	}
}
