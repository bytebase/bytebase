package starrocks

import (
	"context"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestStarRocksPR3DiagnoseE2E is the value-chain keystone for the PR3 DML
// features (omni #309): the new StarRocks DML forms now parse through the
// omni/starrocks fork, so Bytebase's diagnose path stops reporting false
// syntax errors on them (the deliverable is parse-accept, not lineage). Before
// the omni dep bump these forms do not parse and produce a diagnostic; after,
// they diagnose clean. A genuinely-invalid statement still reports an error.
func TestStarRocksPR3DiagnoseE2E(t *testing.T) {
	cases := []struct {
		name     string
		sql      string
		wantDiag bool
	}{
		{
			"insert_overwrite_no_table", // PR3: INSERT OVERWRITE t (no TABLE)
			"INSERT OVERWRITE t SELECT a FROM src",
			false,
		},
		{
			"cte_delete", // PR3: WITH … DELETE
			"WITH c AS (SELECT id FROM s) DELETE FROM t WHERE id IN (SELECT id FROM c)",
			false,
		},
		{
			"async_mv_refresh_every", // PR3: REFRESH ASYNC EVERY (INTERVAL …)
			"CREATE MATERIALIZED VIEW mv REFRESH ASYNC EVERY (INTERVAL 1 DAY) AS SELECT a FROM t",
			false,
		},
		{
			"invalid_statement", // diagnose still catches real syntax errors
			"INSERT OVERWRITE t BY NAME (a) SELECT a FROM s", // BY NAME + col list — rejected
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
