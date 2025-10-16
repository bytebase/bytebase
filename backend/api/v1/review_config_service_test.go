package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestValidateSQLReviewRules(t *testing.T) {
	tests := []struct {
		name    string
		rules   []*v1pb.SQLReviewRule
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty rule list",
			rules:   []*v1pb.SQLReviewRule{},
			wantErr: true,
			errMsg:  "invalid payload, rule list cannot be empty",
		},
		{
			name: "valid ERROR level",
			rules: []*v1pb.SQLReviewRule{
				{
					Type:    "table.require-pk",
					Level:   v1pb.SQLReviewRuleLevel_ERROR,
					Engine:  v1pb.Engine_POSTGRES,
					Payload: "",
				},
			},
			wantErr: false,
		},
		{
			name: "valid WARNING level",
			rules: []*v1pb.SQLReviewRule{
				{
					Type:    "table.require-pk",
					Level:   v1pb.SQLReviewRuleLevel_WARNING,
					Engine:  v1pb.Engine_POSTGRES,
					Payload: "",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid LEVEL_UNSPECIFIED",
			rules: []*v1pb.SQLReviewRule{
				{
					Type:    "table.require-pk",
					Level:   v1pb.SQLReviewRuleLevel_LEVEL_UNSPECIFIED,
					Engine:  v1pb.Engine_POSTGRES,
					Payload: "",
				},
			},
			wantErr: true,
			errMsg:  "invalid rule level: LEVEL_UNSPECIFIED is not allowed for rule \"table.require-pk\"",
		},
		{
			name: "multiple rules with one invalid",
			rules: []*v1pb.SQLReviewRule{
				{
					Type:    "table.require-pk",
					Level:   v1pb.SQLReviewRuleLevel_ERROR,
					Engine:  v1pb.Engine_POSTGRES,
					Payload: "",
				},
				{
					Type:    "table.no-foreign-key",
					Level:   v1pb.SQLReviewRuleLevel_LEVEL_UNSPECIFIED,
					Engine:  v1pb.Engine_POSTGRES,
					Payload: "",
				},
			},
			wantErr: true,
			errMsg:  "invalid rule level: LEVEL_UNSPECIFIED is not allowed for rule \"table.no-foreign-key\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSQLReviewRules(tt.rules)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
