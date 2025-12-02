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

func TestCreateApprovalFlow(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create approval flow service
    service := NewApprovalFlowService(s)
    
    // Test create approval flow
    req := &v1.CreateApprovalFlowRequest{
        DisplayName:       "Test Approval Flow",
        Description:       "Test description",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Steps: []*v1.ApprovalStep{
            {
                StepNumber: 1,
                Approver:   "manager@example.com",
                Role:       "DEPARTMENT_MANAGER",
            },
            {
                StepNumber: 2,
                Approver:   "dba@example.com",
                Role:       "DBA",
            },
        },
    }
    
    resp, err := service.CreateApprovalFlow(ctx, req)
    require.NoError(t, err)
    require.NotNil(t, resp)
    
    assert.Equal(t, "Test Approval Flow", resp.ApprovalFlow.DisplayName)
    assert.Equal(t, v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH, resp.ApprovalFlow.SensitivityLevel)
    assert.Len(t, resp.ApprovalFlow.Steps, 2)
    assert.Equal(t, "manager@example.com", resp.ApprovalFlow.Steps[0].Approver)
    assert.Equal(t, "dba@example.com", resp.ApprovalFlow.Steps[1].Approver)
}

func TestGetApprovalFlow(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create approval flow service
    service := NewApprovalFlowService(s)
    
    // Create an approval flow first
    createReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "Test Approval Flow",
        Description:       "Test description",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
        Steps: []*v1.ApprovalStep{
            {
                StepNumber: 1,
                Approver:   "dba@example.com",
                Role:       "DBA",
            },
        },
    }
    
    createResp, err := service.CreateApprovalFlow(ctx, createReq)
    require.NoError(t, err)
    
    // Test get approval flow
    getReq := &v1.GetApprovalFlowRequest{
        ApprovalFlowId: createResp.ApprovalFlow.Name,
    }
    
    getResp, err := service.GetApprovalFlow(ctx, getReq)
    require.NoError(t, err)
    require.NotNil(t, getResp)
    
    assert.Equal(t, "Test Approval Flow", getResp.ApprovalFlow.DisplayName)
    assert.Equal(t, v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM, getResp.ApprovalFlow.SensitivityLevel)
    assert.Len(t, getResp.ApprovalFlow.Steps, 1)
    assert.Equal(t, "dba@example.com", getResp.ApprovalFlow.Steps[0].Approver)
}

func TestListApprovalFlows(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create approval flow service
    service := NewApprovalFlowService(s)
    
    // Create multiple approval flows
    for i := 0; i < 3; i++ {
        req := &v1.CreateApprovalFlowRequest{
            DisplayName:       "Test Approval Flow " + string(rune(i+1)),
            Description:       "Test description " + string(rune(i+1)),
            SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
            Steps: []*v1.ApprovalStep{
                {
                    StepNumber: 1,
                    Approver:   "user" + string(rune(i+1)) + "@example.com",
                    Role:       "USER",
                },
            },
        }
        
        _, err := service.CreateApprovalFlow(ctx, req)
        require.NoError(t, err)
    }
    
    // Test list approval flows
    listReq := &v1.ListApprovalFlowsRequest{}
    
    listResp, err := service.ListApprovalFlows(ctx, listReq)
    require.NoError(t, err)
    require.NotNil(t, listResp)
    
    assert.Len(t, listResp.ApprovalFlows, 3)
    for i, flow := range listResp.ApprovalFlows {
        assert.Contains(t, flow.DisplayName, "Test Approval Flow")
        assert.Equal(t, v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW, flow.SensitivityLevel)
        assert.Len(t, flow.Steps, 1)
    }
}

func TestUpdateApprovalFlow(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create approval flow service
    service := NewApprovalFlowService(s)
    
    // Create an approval flow first
    createReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "Test Approval Flow",
        Description:       "Test description",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Steps: []*v1.ApprovalStep{
            {
                StepNumber: 1,
                Approver:   "manager@example.com",
                Role:       "DEPARTMENT_MANAGER",
            },
            {
                StepNumber: 2,
                Approver:   "dba@example.com",
                Role:       "DBA",
            },
        },
    }
    
    createResp, err := service.CreateApprovalFlow(ctx, createReq)
    require.NoError(t, err)
    
    // Test update approval flow
    updateReq := &v1.UpdateApprovalFlowRequest{
        ApprovalFlowId: createResp.ApprovalFlow.Name,
        ApprovalFlow: &v1.ApprovalFlow{
            DisplayName:       "Updated Approval Flow",
            Description:       "Updated description",
            SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
            Steps: []*v1.ApprovalStep{
                {
                    StepNumber: 1,
                    Approver:   "new-manager@example.com",
                    Role:       "DEPARTMENT_MANAGER",
                },
                {
                    StepNumber: 2,
                    Approver:   "new-dba@example.com",
                    Role:       "DBA",
                },
            },
        },
        UpdateMask: []string{"display_name", "description", "steps"},
    }
    
    updateResp, err := service.UpdateApprovalFlow(ctx, updateReq)
    require.NoError(t, err)
    require.NotNil(t, updateResp)
    
    assert.Equal(t, "Updated Approval Flow", updateResp.ApprovalFlow.DisplayName)
    assert.Equal(t, "Updated description", updateResp.ApprovalFlow.Description)
    assert.Len(t, updateResp.ApprovalFlow.Steps, 2)
    assert.Equal(t, "new-manager@example.com", updateResp.ApprovalFlow.Steps[0].Approver)
    assert.Equal(t, "new-dba@example.com", updateResp.ApprovalFlow.Steps[1].Approver)
}

func TestDeleteApprovalFlow(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create approval flow service
    service := NewApprovalFlowService(s)
    
    // Create an approval flow first
    createReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "Test Approval Flow",
        Description:       "Test description",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
        Steps: []*v1.ApprovalStep{
            {
                StepNumber: 1,
                Approver:   "user@example.com",
                Role:       "USER",
            },
        },
    }
    
    createResp, err := service.CreateApprovalFlow(ctx, createReq)
    require.NoError(t, err)
    
    // Test delete approval flow
    deleteReq := &v1.DeleteApprovalFlowRequest{
        ApprovalFlowId: createResp.ApprovalFlow.Name,
    }
    
    _, err = service.DeleteApprovalFlow(ctx, deleteReq)
    require.NoError(t, err)
    
    // Verify deletion
    getReq := &v1.GetApprovalFlowRequest{
        ApprovalFlowId: createResp.ApprovalFlow.Name,
    }
    
    _, err = service.GetApprovalFlow(ctx, getReq)
    require.Error(t, err)
}

func TestSubmitApproval(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create approval flow service
    service := NewApprovalFlowService(s)
    
    // Create an approval flow first
    createReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "Test Approval Flow",
        Description:       "Test description",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Steps: []*v1.ApprovalStep{
            {
                StepNumber: 1,
                Approver:   "manager@example.com",
                Role:       "DEPARTMENT_MANAGER",
            },
            {
                StepNumber: 2,
                Approver:   "dba@example.com",
                Role:       "DBA",
            },
        },
    }
    
    createResp, err := service.CreateApprovalFlow(ctx, createReq)
    require.NoError(t, err)
    
    // Test submit approval
    submitReq := &v1.SubmitApprovalRequest{
        Title:             "Test Approval Request",
        Description:       "Test description",
        IssueId:           "test-issue-123",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Submitter:         "user@example.com",
    }
    
    submitResp, err := service.SubmitApproval(ctx, submitReq)
    require.NoError(t, err)
    require.NotNil(t, submitResp)
    
    assert.Equal(t, "Test Approval Request", submitResp.ApprovalRequest.Title)
    assert.Equal(t, "test-issue-123", submitResp.ApprovalRequest.IssueId)
    assert.Equal(t, v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH, submitResp.ApprovalRequest.SensitivityLevel)
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_PENDING, submitResp.ApprovalRequest.Status)
    assert.Equal(t, "user@example.com", submitResp.ApprovalRequest.Submitter)
}

func TestApproveRequest(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create approval flow service
    service := NewApprovalFlowService(s)
    
    // Create an approval flow first
    createReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "Test Approval Flow",
        Description:       "Test description",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
        Steps: []*v1.ApprovalStep{
            {
                StepNumber: 1,
                Approver:   "dba@example.com",
                Role:       "DBA",
            },
        },
    }
    
    createResp, err := service.CreateApprovalFlow(ctx, createReq)
    require.NoError(t, err)
    
    // Submit an approval request
    submitReq := &v1.SubmitApprovalRequest{
        Title:             "Test Approval Request",
        Description:       "Test description",
        IssueId:           "test-issue-123",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
        Submitter:         "user@example.com",
    }
    
    submitResp, err := service.SubmitApproval(ctx, submitReq)
    require.NoError(t, err)
    
    // Test approve request
    approveReq := &v1.ApproveRequestRequest{
        ApprovalRequestId: submitResp.ApprovalRequest.Name,
        Comment:           "Approved by DBA",
        Approver:          "dba@example.com",
    }
    
    approveResp, err := service.ApproveRequest(ctx, approveReq)
    require.NoError(t, err)
    require.NotNil(t, approveResp)
    
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_APPROVED, approveResp.ApprovalRequest.Status)
    assert.Len(t, approveResp.ApprovalRequest.ApprovalHistory, 1)
    assert.Equal(t, "Approved by DBA", approveResp.ApprovalRequest.ApprovalHistory[0].Comment)
    assert.Equal(t, v1.ApprovalActionType_APPROVAL_ACTION_TYPE_APPROVE, approveResp.ApprovalRequest.ApprovalHistory[0].ActionType)
}

func TestRejectRequest(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create approval flow service
    service := NewApprovalFlowService(s)
    
    // Create an approval flow first
    createReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "Test Approval Flow",
        Description:       "Test description",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Steps: []*v1.ApprovalStep{
            {
                StepNumber: 1,
                Approver:   "manager@example.com",
                Role:       "DEPARTMENT_MANAGER",
            },
            {
                StepNumber: 2,
                Approver:   "dba@example.com",
                Role:       "DBA",
            },
        },
    }
    
    createResp, err := service.CreateApprovalFlow(ctx, createReq)
    require.NoError(t, err)
    
    // Submit an approval request
    submitReq := &v1.SubmitApprovalRequest{
        Title:             "Test Approval Request",
        Description:       "Test description",
        IssueId:           "test-issue-123",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Submitter:         "user@example.com",
    }
    
    submitResp, err := service.SubmitApproval(ctx, submitReq)
    require.NoError(t, err)
    
    // Test reject request
    rejectReq := &v1.RejectRequestRequest{
        ApprovalRequestId: submitResp.ApprovalRequest.Name,
        Comment:           "Rejected by Manager - security concerns",
        Approver:          "manager@example.com",
    }
    
    rejectResp, err := service.RejectRequest(ctx, rejectReq)
    require.NoError(t, err)
    require.NotNil(t, rejectResp)
    
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_REJECTED, rejectResp.ApprovalRequest.Status)
    assert.Len(t, rejectResp.ApprovalRequest.ApprovalHistory, 1)
    assert.Equal(t, "Rejected by Manager - security concerns", rejectResp.ApprovalRequest.ApprovalHistory[0].Comment)
    assert.Equal(t, v1.ApprovalActionType_APPROVAL_ACTION_TYPE_REJECT, rejectResp.ApprovalRequest.ApprovalHistory[0].ActionType)
}