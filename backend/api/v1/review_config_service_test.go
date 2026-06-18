package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
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
					Type:   v1pb.SQLReviewRule_TABLE_REQUIRE_PK,
					Level:  v1pb.SQLReviewRule_ERROR,
					Engine: v1pb.Engine_POSTGRES,
				},
			},
			wantErr: false,
		},
		{
			name: "valid WARNING level",
			rules: []*v1pb.SQLReviewRule{
				{
					Type:   v1pb.SQLReviewRule_TABLE_REQUIRE_PK,
					Level:  v1pb.SQLReviewRule_WARNING,
					Engine: v1pb.Engine_POSTGRES,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid LEVEL_UNSPECIFIED",
			rules: []*v1pb.SQLReviewRule{
				{
					Type:   v1pb.SQLReviewRule_TABLE_REQUIRE_PK,
					Level:  v1pb.SQLReviewRule_LEVEL_UNSPECIFIED,
					Engine: v1pb.Engine_POSTGRES,
				},
			},
			wantErr: true,
			errMsg:  "invalid rule level: LEVEL_UNSPECIFIED is not allowed for rule \"TABLE_REQUIRE_PK\"",
		},
		{
			name: "invalid TYPE_UNSPECIFIED",
			rules: []*v1pb.SQLReviewRule{
				{
					Type:   v1pb.SQLReviewRule_TYPE_UNSPECIFIED,
					Level:  v1pb.SQLReviewRule_ERROR,
					Engine: v1pb.Engine_POSTGRES,
				},
			},
			wantErr: true,
			errMsg:  "invalid rule type: TYPE_UNSPECIFIED is not allowed",
		},
		{
			name: "multiple rules with one invalid",
			rules: []*v1pb.SQLReviewRule{
				{
					Type:   v1pb.SQLReviewRule_TABLE_REQUIRE_PK,
					Level:  v1pb.SQLReviewRule_ERROR,
					Engine: v1pb.Engine_POSTGRES,
				},
				{
					Type:   v1pb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
					Level:  v1pb.SQLReviewRule_LEVEL_UNSPECIFIED,
					Engine: v1pb.Engine_POSTGRES,
				},
			},
			wantErr: true,
			errMsg:  "invalid rule level: LEVEL_UNSPECIFIED is not allowed for rule \"TABLE_NO_FOREIGN_KEY\"",
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

func TestConvertReviewConfigMessagesToV1(t *testing.T) {
	reviewConfigs, configByName := convertReviewConfigMessagesToV1([]*store.ReviewConfigMessage{
		{
			ID:      "basic",
			Name:    "Basic",
			Enforce: true,
			Payload: &storepb.ReviewConfigPayload{
				SqlReviewRules: []*storepb.SQLReviewRule{
					{
						Type:   storepb.SQLReviewRule_NAMING_TABLE,
						Level:  storepb.SQLReviewRule_ERROR,
						Engine: storepb.Engine_MYSQL,
					},
				},
			},
		},
		{
			ID:      "advanced",
			Name:    "Advanced",
			Enforce: false,
			Payload: &storepb.ReviewConfigPayload{
				SqlReviewRules: []*storepb.SQLReviewRule{
					{
						Type:   storepb.SQLReviewRule_TABLE_REQUIRE_PK,
						Level:  storepb.SQLReviewRule_WARNING,
						Engine: storepb.Engine_POSTGRES,
					},
				},
			},
		},
	})

	require.Len(t, reviewConfigs, 2)
	require.Equal(t, "reviewConfigs/basic", reviewConfigs[0].Name)
	require.Equal(t, "reviewConfigs/advanced", reviewConfigs[1].Name)
	require.Same(t, reviewConfigs[0], configByName["reviewConfigs/basic"])
	require.Same(t, reviewConfigs[1], configByName["reviewConfigs/advanced"])
	require.True(t, reviewConfigs[0].Enabled)
	require.False(t, reviewConfigs[1].Enabled)
	require.Len(t, reviewConfigs[0].Rules, 1)
	require.Equal(t, v1pb.SQLReviewRule_NAMING_TABLE, reviewConfigs[0].Rules[0].Type)
}
