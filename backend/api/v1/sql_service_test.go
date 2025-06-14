package v1

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/tests/mockstore" // Assuming mock store is here
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	// Import enterprise mock if it's in a different package like mockstore
	// For example: "github.com/bytebase/bytebase/backend/tests/mockenterprise"
	// If enterprise.MockLicenseService is generated in the enterprise package itself, this import is not needed.
	// We will use enterprise.NewMockLicenseService(ctrl) assuming it's available.
)

func TestCheckDataSourceQueryPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockstore.NewMockStore(ctrl)
	// Assuming mock for LicenseService is typically generated in the enterprise package or a sub-package.
	// If enterprise.NewMockLicenseService doesn't exist, it means mockgen needs to be run for that interface.
	// For this example, we'll proceed as if it's available.
	// If it's in a separate mock package like "mockenterprise", it would be:
	// mockLicense := mockenterprise.NewMockLicenseService(ctrl)
	mockLicense := enterprise.NewMockLicenseService(ctrl) // Assuming mock is in enterprise package

	ctx := context.Background()
	dummyEnvID := "test-env"
	dummyDBName := "test-db"
	database := &store.DatabaseMessage{
		EffectiveEnvironmentID: dummyEnvID,
		DatabaseName:           dummyDBName,
	}
	environment := &store.EnvironmentMessage{
		ResourceID: dummyEnvID,
		Title:      "Test Environment",
	}

	tests := []struct {
		name                 string
		statementType        base.QueryType
		featureEnabled       bool
		policy               *v1pb.DataSourceQueryPolicy
		getPolicyErr         error
		getEnvironmentErr    error
		expectError          bool
		expectedErrorMessage string
	}{
		{
			name:           "Feature Disabled - DDL",
			statementType:  base.DDL,
			featureEnabled: false,
			expectError:    false,
		},
		{
			name:           "Feature Disabled - DML",
			statementType:  base.DML,
			featureEnabled: false,
			expectError:    false,
		},
		{
			name:           "Feature Enabled - No Policy - DDL",
			statementType:  base.DDL,
			featureEnabled: true,
			policy:         nil, // No policy found
			expectError:    false,
		},
		{
			name:           "Feature Enabled - No Policy - DML",
			statementType:  base.DML,
			featureEnabled: true,
			policy:         nil, // No policy found
			expectError:    false,
		},
		{
			name:           "Feature Enabled - Policy Allows DDL",
			statementType:  base.DDL,
			featureEnabled: true,
			policy:         &v1pb.DataSourceQueryPolicy{DisallowDdl: false, DisallowDml: false},
			expectError:    false,
		},
		{
			name:           "Feature Enabled - Policy Allows DML",
			statementType:  base.DML,
			featureEnabled: true,
			policy:         &v1pb.DataSourceQueryPolicy{DisallowDdl: false, DisallowDml: false},
			expectError:    false,
		},
		{
			name:                 "Feature Enabled - Policy Disallows DDL",
			statementType:        base.DDL,
			featureEnabled:       true,
			policy:               &v1pb.DataSourceQueryPolicy{DisallowDdl: true},
			expectError:          true,
			expectedErrorMessage: fmt.Sprintf("disallow execute DDL statement in environment %q", environment.Title),
		},
		{
			name:                 "Feature Enabled - Policy Disallows DML",
			statementType:        base.DML,
			featureEnabled:       true,
			policy:               &v1pb.DataSourceQueryPolicy{DisallowDml: true},
			expectError:          true,
			expectedErrorMessage: fmt.Sprintf("disallow execute DML statement in environment %q", environment.Title),
		},
		{
			name:              "Feature Enabled - GetEnvironment Error",
			statementType:     base.DDL,
			featureEnabled:    true,
			getEnvironmentErr: errors.New("failed to get environment"),
			expectError:       true,
			expectedErrorMessage: "failed to get environment",
		},
		{
			name:           "Feature Enabled - GetPolicy Error",
			statementType:  base.DDL,
			featureEnabled: true,
			getPolicyErr:   errors.New("failed to get policy"),
			expectError:    true,
			expectedErrorMessage: "failed to get policy",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.featureEnabled {
				mockLicense.EXPECT().IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY).Return(nil) // nil means enabled
				mockStore.EXPECT().GetEnvironmentByID(ctx, dummyEnvID).Return(environment, tc.getEnvironmentErr)
				if tc.getEnvironmentErr == nil {
					resourceType := storepb.Policy_ENVIRONMENT
					environmentResource := common.FormatEnvironment(environment.ResourceID)
					policyType := storepb.Policy_DATA_SOURCE_QUERY

					var policyMsg *store.PolicyMessage
					if tc.policy != nil {
						payload, err := common.ProtojsonMarshaler.MarshalToString(tc.policy)
						require.NoError(t, err)
						policyMsg = &store.PolicyMessage{Payload: payload}
					}
					mockStore.EXPECT().GetPolicyV2(ctx, &store.FindPolicyMessage{
						ResourceType: &resourceType,
						Resource:     &environmentResource,
						Type:         &policyType,
					}).Return(policyMsg, tc.getPolicyErr)
				}
			} else {
				mockLicense.EXPECT().IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY).Return(errors.New("feature not enabled"))
			}

			err := checkDataSourceQueryPolicy(ctx, mockStore, mockLicense, database, tc.statementType)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectedErrorMessage != "" {
					require.Contains(t, err.Error(), tc.expectedErrorMessage)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
