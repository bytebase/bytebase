package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestParsePlanCheckRunFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		want        *store.FindPlanCheckRunMessage
		wantErr     bool
		errContains string
	}{
		{
			name:   "empty filter",
			filter: "",
			want:   &store.FindPlanCheckRunMessage{},
		},
		{
			name:   "status equals DONE",
			filter: `status == "DONE"`,
			want: &store.FindPlanCheckRunMessage{
				Status: &[]store.PlanCheckRunStatus{store.PlanCheckRunStatusDone},
			},
		},
		{
			name:   "status equals RUNNING",
			filter: `status == "RUNNING"`,
			want: &store.FindPlanCheckRunMessage{
				Status: &[]store.PlanCheckRunStatus{store.PlanCheckRunStatusRunning},
			},
		},
		{
			name:   "status equals FAILED",
			filter: `status == "FAILED"`,
			want: &store.FindPlanCheckRunMessage{
				Status: &[]store.PlanCheckRunStatus{store.PlanCheckRunStatusFailed},
			},
		},
		{
			name:   "status equals CANCELED",
			filter: `status == "CANCELED"`,
			want: &store.FindPlanCheckRunMessage{
				Status: &[]store.PlanCheckRunStatus{store.PlanCheckRunStatusCanceled},
			},
		},
		{
			name:   "status in list",
			filter: `status in ["DONE", "FAILED"]`,
			want: &store.FindPlanCheckRunMessage{
				Status: &[]store.PlanCheckRunStatus{
					store.PlanCheckRunStatusDone,
					store.PlanCheckRunStatusFailed,
				},
			},
		},
		{
			name:   "result_status equals SUCCESS",
			filter: `result_status == "SUCCESS"`,
			want: &store.FindPlanCheckRunMessage{
				ResultStatus: &[]storepb.Advice_Status{
					storepb.Advice_SUCCESS,
				},
			},
		},
		{
			name:   "result_status equals ERROR",
			filter: `result_status == "ERROR"`,
			want: &store.FindPlanCheckRunMessage{
				ResultStatus: &[]storepb.Advice_Status{
					storepb.Advice_ERROR,
				},
			},
		},
		{
			name:   "result_status equals WARNING",
			filter: `result_status == "WARNING"`,
			want: &store.FindPlanCheckRunMessage{
				ResultStatus: &[]storepb.Advice_Status{
					storepb.Advice_WARNING,
				},
			},
		},
		{
			name:   "result_status in list",
			filter: `result_status in ["SUCCESS", "WARNING"]`,
			want: &store.FindPlanCheckRunMessage{
				ResultStatus: &[]storepb.Advice_Status{
					storepb.Advice_SUCCESS,
					storepb.Advice_WARNING,
				},
			},
		},
		{
			name:   "combined status and result_status",
			filter: `status == "DONE" && result_status == "SUCCESS"`,
			want: &store.FindPlanCheckRunMessage{
				Status: &[]store.PlanCheckRunStatus{store.PlanCheckRunStatusDone},
				ResultStatus: &[]storepb.Advice_Status{
					storepb.Advice_SUCCESS,
				},
			},
		},
		{
			name:   "combined status in list and result_status",
			filter: `status in ["DONE", "FAILED"] && result_status == "ERROR"`,
			want: &store.FindPlanCheckRunMessage{
				Status: &[]store.PlanCheckRunStatus{
					store.PlanCheckRunStatusDone,
					store.PlanCheckRunStatusFailed,
				},
				ResultStatus: &[]storepb.Advice_Status{
					storepb.Advice_ERROR,
				},
			},
		},
		{
			name:        "invalid status value",
			filter:      `status == "INVALID_STATUS"`,
			wantErr:     true,
			errContains: "invalid status value",
		},
		{
			name:        "invalid result_status value",
			filter:      `result_status == "INVALID_RESULT_STATUS"`,
			wantErr:     true,
			errContains: "invalid result_status value",
		},
		{
			name:        "non-string status value",
			filter:      `status == 123`,
			wantErr:     true,
			errContains: "status value must be a string",
		},
		{
			name:        "non-string result_status value",
			filter:      `result_status == 123`,
			wantErr:     true,
			errContains: "result_status value must be a string",
		},
		{
			name:        "empty status list",
			filter:      `status in []`,
			wantErr:     true,
			errContains: "empty list value for filter",
		},
		{
			name:        "empty result_status list",
			filter:      `result_status in []`,
			wantErr:     true,
			errContains: "empty list value for filter",
		},
		{
			name:        "unsupported variable",
			filter:      `unknown_field == "value"`,
			wantErr:     true,
			errContains: "unsupported filter variable",
		},
		{
			name:        "unsupported operator",
			filter:      `status > "DONE"`,
			wantErr:     true,
			errContains: "unsupported operator",
		},
		{
			name:        "invalid CEL syntax",
			filter:      `status == "DONE" &&`,
			wantErr:     true,
			errContains: "failed to parse filter",
		},
		{
			name:        "non-string in status list",
			filter:      `status in ["DONE", 123]`,
			wantErr:     true,
			errContains: "status value must be a string",
		},
		{
			name:        "non-string in result_status list",
			filter:      `result_status in ["SUCCESS", 123]`,
			wantErr:     true,
			errContains: "result_status value must be a string",
		},
	}

	service := &PlanService{}
	a := require.New(t)

	for _, test := range tests {
		t.Run(test.name, func(*testing.T) {
			find := &store.FindPlanCheckRunMessage{}
			err := service.parsePlanCheckRunFilter(test.filter, find)

			if test.wantErr {
				a.Error(err, "expected error for test case: %s", test.name)
				if test.errContains != "" {
					a.Contains(err.Error(), test.errContains, "error message should contain expected text")
				}
				return
			}

			a.NoError(err, "unexpected error for test case: %s", test.name)

			// Compare Status slices
			if test.want.Status == nil {
				a.Nil(find.Status, "Status should be nil")
			} else {
				a.NotNil(find.Status, "Status should not be nil")
				a.ElementsMatch(*test.want.Status, *find.Status, "Status slices should match")
			}

			// Compare ResultStatus slices
			if test.want.ResultStatus == nil {
				a.Nil(find.ResultStatus, "ResultStatus should be nil")
			} else {
				a.NotNil(find.ResultStatus, "ResultStatus should not be nil")
				a.ElementsMatch(*test.want.ResultStatus, *find.ResultStatus, "ResultStatus slices should match")
			}

			// Compare other fields that should remain unchanged
			a.Equal(test.want.PlanUID, find.PlanUID, "PlanUID should match")
			a.Equal(test.want.UIDs, find.UIDs, "UIDs should match")
		})
	}
}
