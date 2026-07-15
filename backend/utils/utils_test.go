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

func TestCheckApprovalApproved(t *testing.T) {
	tests := []struct {
		name     string
		approval *storepb.IssuePayloadApproval
		want     bool
		wantErr  string
	}{
		{
			name: "approval finding not done is not approved",
			approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: false,
			},
		},
		{
			name: "approval without template keeps existing approved behavior",
			approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
			},
			want: true,
		},
		{
			name: "empty approval flow skips approval",
			approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{},
				},
			},
			want: true,
		},
		{
			name: "pending approval role is not approved",
			approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
				},
			},
		},
		{
			name: "empty approval role is invalid",
			approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{Roles: []string{""}},
				},
			},
			wantErr: "approval template role at position 1 cannot be empty",
		},
		{
			name: "blank approval role is invalid",
			approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
				ApprovalTemplate: &storepb.ApprovalTemplate{
					Flow: &storepb.ApprovalFlow{Roles: []string{" "}},
				},
			},
			wantErr: "approval template role at position 1 cannot be empty",
		},
		{
			name: "nil approval flow is invalid",
			approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: true,
				ApprovalTemplate:    &storepb.ApprovalTemplate{},
			},
			wantErr: "approval template flow cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckApprovalApproved(tt.approval)
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
