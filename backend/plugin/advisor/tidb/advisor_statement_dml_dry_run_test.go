package tidb

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestStatementDMLDryRun(t *testing.T) {
	sm := sheet.NewManager()
	rules := []*storepb.SQLReviewRule{{Type: storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, Level: storepb.SQLReviewRule_WARNING, Engine: storepb.Engine_TIDB}}

	tests := []struct {
		name       string
		statement  string
		failOn     string
		wantProbes []advisor.ProbeCall
		wantAdvice int
	}{
		{
			name:      "plain dml is explained in a transaction",
			statement: "DELETE FROM t WHERE id = 1;",
			wantProbes: []advisor.ProbeCall{
				{Statement: "EXPLAIN DELETE FROM t WHERE id = 1"},
			},
		},
		{
			name:      "explain failure is reported",
			statement: "DELETE FROM t WHERE missing_col = 1",
			failOn:    "missing_col",
			wantProbes: []advisor.ProbeCall{
				{Statement: "EXPLAIN DELETE FROM t WHERE missing_col = 1"},
			},
			wantAdvice: 1,
		},
		{
			name:      "batch dry runs in autocommit then explains the inner dml",
			statement: "BATCH ON id LIMIT 2 DELETE FROM t WHERE id > 0",
			wantProbes: []advisor.ProbeCall{
				{Autocommit: true, Statement: "BATCH ON id LIMIT 2 DRY RUN DELETE FROM t WHERE id > 0"},
				{Statement: "EXPLAIN DELETE FROM t WHERE id > 0"},
			},
		},
		{
			name:      "batch update",
			statement: "BATCH ON id LIMIT 2 UPDATE t SET name = 'y' WHERE id > 0",
			wantProbes: []advisor.ProbeCall{
				{Autocommit: true, Statement: "BATCH ON id LIMIT 2 DRY RUN UPDATE t SET name = 'y' WHERE id > 0"},
				{Statement: "EXPLAIN UPDATE t SET name = 'y' WHERE id > 0"},
			},
		},
		{
			name:      "batch insert select",
			statement: "BATCH ON id LIMIT 2 INSERT INTO t2 SELECT id, name FROM t",
			wantProbes: []advisor.ProbeCall{
				{Autocommit: true, Statement: "BATCH ON id LIMIT 2 DRY RUN INSERT INTO t2 SELECT id, name FROM t"},
				{Statement: "EXPLAIN INSERT INTO t2 SELECT id, name FROM t"},
			},
		},
		{
			name:      "batch without the optional ON clause",
			statement: "BATCH LIMIT 2 DELETE FROM t WHERE id > 0",
			wantProbes: []advisor.ProbeCall{
				{Autocommit: true, Statement: "BATCH LIMIT 2 DRY RUN DELETE FROM t WHERE id > 0"},
				{Statement: "EXPLAIN DELETE FROM t WHERE id > 0"},
			},
		},
		{
			name:      "batch dry run failure skips the explain",
			statement: "BATCH ON missing_col LIMIT 2 DELETE FROM t WHERE id > 0",
			failOn:    "missing_col",
			wantProbes: []advisor.ProbeCall{
				{Autocommit: true, Statement: "BATCH ON missing_col LIMIT 2 DRY RUN DELETE FROM t WHERE id > 0"},
			},
			wantAdvice: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prober := &advisor.ProbeRecorder{}
			if tc.failOn != "" {
				prober.ProbeFunc = func(_ db.ProbeContext, statement string) (db.ProbeResult, error) {
					if strings.Contains(statement, tc.failOn) {
						return db.ProbeResult{}, errors.New("rejected: " + statement)
					}
					return db.ProbeResult{}, nil
				}
			}

			advices, err := advisor.SQLReviewCheck(context.Background(), sm, tc.statement, rules, advisor.Context{
				DBType:          storepb.Engine_TIDB,
				Prober:          prober,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantProbes, prober.Probes)
			require.Len(t, advices, tc.wantAdvice)
			for _, a := range advices {
				require.Equal(t, code.StatementDMLDryRunFailed.Int32(), a.Code)
			}
		})
	}
}
