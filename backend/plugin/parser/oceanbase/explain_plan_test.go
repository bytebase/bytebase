package oceanbase

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEstimateAffectedRowsFromExplainJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		plan string
		want int64
	}{
		{
			name: "root operator",
			plan: `{"ID":0,"OPERATOR":"UPDATE","NAME":"","EST.ROWS":1000,"EST.TIME(us)":7680,"output":"","CHILD_1":{"ID":1,"OPERATOR":"TABLE RANGE SCAN","NAME":"dba_test_1","EST.ROWS":1200,"EST.TIME(us)":91}}`,
			want: 1000,
		},
		{
			name: "fractional estimate",
			plan: `{"ID":0,"OPERATOR":"DELETE","NAME":"","EST.ROWS":2.6,"EST.TIME(us)":20}`,
			want: 3,
		},
		{
			name: "largest child operator without a root operator",
			plan: `{"CHILD_1":{"ID":1,"OPERATOR":"TABLE FULL SCAN","NAME":"t1","EST.ROWS":40},"CHILD_2":{"ID":2,"OPERATOR":"TABLE FULL SCAN","NAME":"t2","EST.ROWS":70}}`,
			want: 70,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EstimateAffectedRowsFromExplainJSON(tc.plan)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	for _, tc := range []struct {
		name    string
		plan    string
		wantErr string
	}{
		{
			name:    "root operator without an estimate",
			plan:    `{"ID":0,"OPERATOR":"UPDATE","NAME":""}`,
			wantErr: `operator "UPDATE" has no EST.ROWS`,
		},
		{
			name:    "child operator without an estimate",
			plan:    `{"CHILD_1":{"ID":1,"OPERATOR":"TABLE FULL SCAN","NAME":"t1"}}`,
			wantErr: `operator "TABLE FULL SCAN" has no EST.ROWS`,
		},
		{
			name:    "no operator",
			plan:    `{}`,
			wantErr: "the plan has no operator",
		},
		{
			name:    "empty output",
			plan:    "",
			wantErr: "failed to parse EXPLAIN FORMAT=JSON output",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EstimateAffectedRowsFromExplainJSON(tc.plan)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
