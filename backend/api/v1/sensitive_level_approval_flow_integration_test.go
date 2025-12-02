// Copyright 2024 Bytebase Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/bytebase/bytebase/backend/api/v1/gen/v1"
    "github.com/bytebase/bytebase/backend/store"
    "github.com/bytebase/bytebase/backend/testutil"
)

func TestSensitiveLevelAndApprovalFlowIntegration(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create services
    sensitiveLevelService := NewSensitiveLevelService(s)
    approvalFlowService := NewApprovalFlowService(s)
    
    // Create a test instance
    instance := &store.InstanceMessage{
        ID:          "test-instance",
        ResourceUID: "test-instance-uid",
        Name:        "Test Instance",
        Type:        store.MYSQL,
        Host:        "localhost",
        Port:        3306,
        Database:    "test",
        Status:      store.INSTANCE_STATUS_READY,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    err := s.CreateInstance(ctx, instance)
    require.NoError(t, err)
    
    // Step 1: Create sensitive levels for different sensitivity levels
    highLevelReq := &v1.CreateSensitiveLevelRequest{
        DisplayName: "High Sensitive Data",
        Description: "High sensitivity data (requires 2-level approval)",
        Level:       v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        TableName:   "users",
        SchemaName:  "public",
        InstanceId:  "test-instance",
        FieldRules: []*v1.FieldRule{
            {
                FieldName: "password",
                DataType:  "varchar",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_EXACT,
                Pattern:   "password",
            },
            {
                FieldName: "credit_card",
                DataType:  "varchar",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_REGEX,
                Pattern:   "^\\d{16}$",
            },
        },
    }
    
    highLevelResp, err := sensitiveLevelService.CreateSensitiveLevel(ctx, highLevelReq)
    require.NoError(t, err)
    
    mediumLevelReq := &v1.CreateSensitiveLevelRequest{
        DisplayName: "Medium Sensitive Data",
        Description: "Medium sensitivity data (requires 1-level approval)",
        Level:       v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
        TableName:   "users",
        SchemaName:  "public",
        InstanceId:  "test-instance",
        FieldRules: []*v1.FieldRule{
            {
                FieldName: "email",
                DataType:  "varchar",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_REGEX,
                Pattern:   ".*@.*",
            },
            {
                FieldName: "phone",
                DataType:  "varchar",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_REGEX,
                Pattern:   "^\\d{11}$",
            },
        },
    }
    
    mediumLevelResp, err := sensitiveLevelService.CreateSensitiveLevel(ctx, mediumLevelReq)
    require.NoError(t, err)
    
    lowLevelReq := &v1.CreateSensitiveLevelRequest{
        DisplayName: "Low Sensitive Data",
        Description: "Low sensitivity data (no approval required)",
        Level:       v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
        TableName:   "users",
        SchemaName:  "public",
        InstanceId:  "test-instance",
        FieldRules: []*v1.FieldRule{
            {
                FieldName: "name",
                DataType:  "varchar",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_EXACT,
                Pattern:   "name",
            },
            {
                FieldName: "address",
                DataType:  "text",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_EXACT,
                Pattern:   "address",
            },
        },
    }
    
    lowLevelResp, err := sensitiveLevelService.CreateSensitiveLevel(ctx, lowLevelReq)
    require.NoError(t, err)
    
    // Step 2: Create corresponding approval flows
    highApprovalFlowReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "High Sensitivity Approval Flow",
        Description:       "2-level approval for high sensitivity data",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Steps: []*v1.ApprovalStep{
            {
                StepNumber: 1,
                Approver:   "department-manager@example.com",
                Role:       "DEPARTMENT_MANAGER",
            },
            {
                StepNumber: 2,
                Approver:   "dba@example.com",
                Role:       "DBA",
            },
        },
    }
    
    highApprovalFlowResp, err := approvalFlowService.CreateApprovalFlow(ctx, highApprovalFlowReq)
    require.NoError(t, err)
    
    mediumApprovalFlowReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "Medium Sensitivity Approval Flow",
        Description:       "1-level approval for medium sensitivity data",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
        Steps: []*v1.ApprovalStep{
            {
                StepNumber: 1,
                Approver:   "dba@example.com",
                Role:       "DBA",
            },
        },
    }
    
    mediumApprovalFlowResp, err := approvalFlowService.CreateApprovalFlow(ctx, mediumApprovalFlowReq)
    require.NoError(t, err)
    
    lowApprovalFlowReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "Low Sensitivity Approval Flow",
        Description:       "No approval required for low sensitivity data",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
        Steps: []*v1.ApprovalStep{},
    }
    
    lowApprovalFlowResp, err := approvalFlowService.CreateApprovalFlow(ctx, lowApprovalFlowReq)
    require.NoError(t, err)
    
    // Step 3: Test high sensitivity approval flow
    highSubmitReq := &v1.SubmitApprovalRequest{
        Title:             "Update User Password",
        Description:       "Update user password in users table",
        IssueId:           "issue-12345",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Submitter:         "developer@example.com",
    }
    
    highSubmitResp, err := approvalFlowService.SubmitApproval(ctx, highSubmitReq)
    require.NoError(t, err)
    
    // Verify initial status
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_PENDING, highSubmitResp.ApprovalRequest.Status)
    
    // First approval step (department manager)
    firstApproveReq := &v1.ApproveRequestRequest{
        ApprovalRequestId: highSubmitResp.ApprovalRequest.Name,
        Comment:           "Approved by department manager",
        Approver:          "department-manager@example.com",
    }
    
    firstApproveResp, err := approvalFlowService.ApproveRequest(ctx, firstApproveReq)
    require.NoError(t, err)
    
    // Second approval step (DBA)
    secondApproveReq := &v1.ApproveRequestRequest{
        ApprovalRequestId: highSubmitResp.ApprovalRequest.Name,
        Comment:           "Approved by DBA",
        Approver:          "dba@example.com",
    }
    
    secondApproveResp, err := approvalFlowService.ApproveRequest(ctx, secondApproveReq)
    require.NoError(t, err)
    
    // Verify final status
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_APPROVED, secondApproveResp.ApprovalRequest.Status)
    assert.Len(t, secondApproveResp.ApprovalRequest.ApprovalHistory, 2)
    
    // Step 4: Test medium sensitivity approval flow
    mediumSubmitReq := &v1.SubmitApprovalRequest{
        Title:             "Update User Email",
        Description:       "Update user email in users table",
        IssueId:           "issue-67890",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
        Submitter:         "developer@example.com",
    }
    
    mediumSubmitResp, err := approvalFlowService.SubmitApproval(ctx, mediumSubmitReq)
    require.NoError(t, err)
    
    // Verify initial status
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_PENDING, mediumSubmitResp.ApprovalRequest.Status)
    
    // Approval step (DBA)
    mediumApproveReq := &v1.ApproveRequestRequest{
        ApprovalRequestId: mediumSubmitResp.ApprovalRequest.Name,
        Comment:           "Approved by DBA",
        Approver:          "dba@example.com",
    }
    
    mediumApproveResp, err := approvalFlowService.ApproveRequest(ctx, mediumApproveReq)
    require.NoError(t, err)
    
    // Verify final status
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_APPROVED, mediumApproveResp.ApprovalRequest.Status)
    assert.Len(t, mediumApproveResp.ApprovalRequest.ApprovalHistory, 1)
    
    // Step 5: Test low sensitivity approval flow (no approval required)
    lowSubmitReq := &v1.SubmitApprovalRequest{
        Title:             "Update User Name",
        Description:       "Update user name in users table",
        IssueId:           "issue-11223",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
        Submitter:         "developer@example.com",
    }
    
    lowSubmitResp, err := approvalFlowService.SubmitApproval(ctx, lowSubmitReq)
    require.NoError(t, err)
    
    // Verify status (should be approved automatically)
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_APPROVED, lowSubmitResp.ApprovalRequest.Status)
    assert.Empty(t, lowSubmitResp.ApprovalRequest.ApprovalHistory)
    
    // Step 6: Test approval rejection
    rejectSubmitReq := &v1.SubmitApprovalRequest{
        Title:             "Rejected Approval Request",
        Description:       "This request should be rejected",
        IssueId:           "issue-44556",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Submitter:         "developer@example.com",
    }
    
    rejectSubmitResp, err := approvalFlowService.SubmitApproval(ctx, rejectSubmitReq)
    require.NoError(t, err)
    
    // Reject the request
    rejectReq := &v1.RejectRequestRequest{
        ApprovalRequestId: rejectSubmitResp.ApprovalRequest.Name,
        Comment:           "Rejected due to security concerns",
        Approver:          "department-manager@example.com",
    }
    
    rejectResp, err := approvalFlowService.RejectRequest(ctx, rejectReq)
    require.NoError(t, err)
    
    // Verify rejection status
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_REJECTED, rejectResp.ApprovalRequest.Status)
    assert.Len(t, rejectResp.ApprovalRequest.ApprovalHistory, 1)
    assert.Equal(t, v1.ApprovalActionType_APPROVAL_ACTION_TYPE_REJECT, rejectResp.ApprovalRequest.ApprovalHistory[0].ActionType)
    
    // Step 7: Test listing approval requests
    listReq := &v1.ListApprovalRequestsRequest{}
    listResp, err := approvalFlowService.ListApprovalRequests(ctx, listReq)
    require.NoError(t, err)
    
    // Should have 4 approval requests
    assert.Len(t, listResp.ApprovalRequests, 4)
    
    // Verify each request's sensitivity level
    sensitivityLevels := []v1.SensitivityLevel{
        v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
        v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
        v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
    }
    
    for i, request := range listResp.ApprovalRequests {
        assert.Equal(t, sensitivityLevels[i], request.SensitivityLevel)
    }
}