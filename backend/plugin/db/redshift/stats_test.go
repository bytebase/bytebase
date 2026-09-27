package redshift

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The plans are EXPLAIN output published in the Amazon Redshift docs:
// https://docs.aws.amazon.com/redshift/latest/dg/c-the-query-plan.html and
// https://docs.aws.amazon.com/redshift/latest/dg/r_EXPLAIN.html.
func TestGetAffectedRowsFromPlan(t *testing.T) {
	for _, tc := range []struct {
		name    string
		plan    string
		want    int64
		wantErr bool
	}{
		{
			name: "hash_join_estimates_joined_rows_not_its_inner_table",
			plan: ` XN Hash Join DS_BCAST_INNER  (cost=0.14..6600286.07 rows=8798 width=84)
   Hash Cond: ("outer".catid = "inner".catid)
   ->  XN Seq Scan on event  (cost=0.00..87.98 rows=8798 width=35)
   ->  XN Hash  (cost=0.11..0.11 rows=11 width=49)
         ->  XN Seq Scan on category  (cost=0.00..0.11 rows=11 width=49)`,
			want: 8798,
		},
		{
			name: "three_way_join_estimates_the_outermost_join",
			plan: `XN Hash Join DS_BCAST_INNER  (cost=109.98..3871130276.17 rows=172456 width=132)
  Hash Cond: ("outer".eventid = "inner".eventid)
  ->  XN Merge Join DS_DIST_NONE  (cost=0.00..6285.93 rows=172456 width=97)
        Merge Cond: ("outer".listid = "inner".listid)
        ->  XN Seq Scan on listing  (cost=0.00..1924.97 rows=192497 width=44)
        ->  XN Seq Scan on sales  (cost=0.00..1724.56 rows=172456 width=53)
  ->  XN Hash  (cost=87.98..87.98 rows=8798 width=35)
        ->  XN Seq Scan on event  (cost=0.00..87.98 rows=8798 width=35)`,
			want: 172456,
		},
		{
			name: "aggregate_estimates_groups_not_scanned_rows",
			plan: `XN HashAggregate  (cost=131.97..133.41 rows=576 width=17)
  ->  XN Seq Scan on event  (cost=0.00..87.98 rows=8798 width=17)`,
			want: 576,
		},
		{
			name: "scan_estimates_rows_after_its_filter",
			plan: `XN Seq Scan on venue  (cost=0.00..2.02 rows=187 width=45)
Filter: (venueseats IS NOT NULL)`,
			want: 187,
		},
		{
			name:    "plan_without_rows_estimate",
			plan:    `Filter: (venueseats IS NOT NULL)`,
			wantErr: true,
		},
		{
			name:    "empty_plan",
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var plan []string
			if tc.plan != "" {
				plan = strings.Split(tc.plan, "\n")
			}
			got, err := getAffectedRowsFromPlan(plan)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
