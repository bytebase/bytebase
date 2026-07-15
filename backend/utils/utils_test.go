//nolint:revive
package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCheckDatabaseGroupMatch(t *testing.T) {
	tests := []struct {
		expression string
		database   *store.DatabaseMessage
		match      bool
	}{
		{
			expression: `resource.database_labels.unit == "gcp"`,
			database: &store.DatabaseMessage{
				Metadata: &storepb.DatabaseMetadata{
					Labels: map[string]string{
						"unit": "gcp",
					},
				},
			},
			match: true,
		},
		{
			expression: `resource.database_labels.unit == "aws"`,
			database: &store.DatabaseMessage{
				Metadata: &storepb.DatabaseMetadata{
					Labels: map[string]string{
						"unit": "gcp",
					},
				},
			},
			match: false,
		},
		{
			expression: `has(resource.database_labels.unit) && resource.database_labels.unit == "aws"`,
			database: &store.DatabaseMessage{
				Metadata: &storepb.DatabaseMetadata{},
			},
			match: false,
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		match, err := CheckDatabaseGroupMatch(ctx, test.expression, test.database)
		assert.NoError(t, err)
		assert.Equal(t, test.match, match)
	}
}

func TestCheckIssueApprovedForPlan(t *testing.T) {
	tests := []struct {
		name      string
		issueType storepb.Issue_Type
		issue     *storepb.Issue
		plan      *storepb.PlanConfig
		want      bool
		wantErr   string
	}{
		{
			name:      "database change approved for matching plan version",
			issueType: storepb.Issue_DATABASE_CHANGE,
			issue: &storepb.Issue{
				Approval: &storepb.IssuePayloadApproval{
					ApprovalFindingDone:  true,
					ApprovalInputVersion: 2,
				},
			},
			plan: &storepb.PlanConfig{ApprovalInputVersion: 2},
			want: true,
		},
		{
			name:      "database change stale approval is not approved",
			issueType: storepb.Issue_DATABASE_CHANGE,
			issue: &storepb.Issue{
				Approval: &storepb.IssuePayloadApproval{
					ApprovalFindingDone:  true,
					ApprovalInputVersion: 1,
				},
			},
			plan: &storepb.PlanConfig{ApprovalInputVersion: 2},
		},
		{
			name:      "database change matching version still requires completed approval",
			issueType: storepb.Issue_DATABASE_CHANGE,
			issue: &storepb.Issue{
				Approval: &storepb.IssuePayloadApproval{
					ApprovalInputVersion: 2,
				},
			},
			plan: &storepb.PlanConfig{ApprovalInputVersion: 2},
		},
		{
			name:      "non database change keeps existing approval behavior",
			issueType: storepb.Issue_ROLE_GRANT,
			issue: &storepb.Issue{
				Approval: &storepb.IssuePayloadApproval{
					ApprovalFindingDone:  true,
					ApprovalInputVersion: 1,
				},
			},
			plan: &storepb.PlanConfig{ApprovalInputVersion: 2},
			want: true,
		},
		{
			name:      "empty approval flow skips approval",
			issueType: storepb.Issue_DATABASE_CHANGE,
			issue: &storepb.Issue{
				Approval: &storepb.IssuePayloadApproval{
					ApprovalFindingDone:  true,
					ApprovalInputVersion: 2,
					ApprovalTemplate: &storepb.ApprovalTemplate{
						Flow: &storepb.ApprovalFlow{},
					},
				},
			},
			plan: &storepb.PlanConfig{ApprovalInputVersion: 2},
			want: true,
		},
		{
			name:      "empty approval role is invalid",
			issueType: storepb.Issue_DATABASE_CHANGE,
			issue: &storepb.Issue{
				Approval: &storepb.IssuePayloadApproval{
					ApprovalFindingDone:  true,
					ApprovalInputVersion: 2,
					ApprovalTemplate: &storepb.ApprovalTemplate{
						Flow: &storepb.ApprovalFlow{Roles: []string{""}},
					},
				},
			},
			plan:    &storepb.PlanConfig{ApprovalInputVersion: 2},
			wantErr: "approval template role at position 1 cannot be empty",
		},
		{
			name:      "blank approval role is invalid",
			issueType: storepb.Issue_DATABASE_CHANGE,
			issue: &storepb.Issue{
				Approval: &storepb.IssuePayloadApproval{
					ApprovalFindingDone:  true,
					ApprovalInputVersion: 2,
					ApprovalTemplate: &storepb.ApprovalTemplate{
						Flow: &storepb.ApprovalFlow{Roles: []string{" "}},
					},
				},
			},
			plan:    &storepb.PlanConfig{ApprovalInputVersion: 2},
			wantErr: "approval template role at position 1 cannot be empty",
		},
		{
			name:      "nil approval flow is invalid",
			issueType: storepb.Issue_DATABASE_CHANGE,
			issue: &storepb.Issue{
				Approval: &storepb.IssuePayloadApproval{
					ApprovalFindingDone:  true,
					ApprovalInputVersion: 2,
					ApprovalTemplate:     &storepb.ApprovalTemplate{},
				},
			},
			plan:    &storepb.PlanConfig{ApprovalInputVersion: 2},
			wantErr: "approval template flow cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckIssueApprovedForPlan(&store.IssueMessage{
				Type:    tt.issueType,
				Payload: tt.issue,
			}, &store.PlanMessage{
				Config: tt.plan,
			})
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.False(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
