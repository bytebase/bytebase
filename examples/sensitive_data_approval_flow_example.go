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

package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/bytebase/bytebase/backend/api/v1/gen/v1"
    "github.com/bytebase/bytebase/backend/store"
    "github.com/bytebase/bytebase/backend/testutil"
)

func main() {
    // 创建上下文
    ctx := context.Background()
    
    // 创建存储实例
    s := testutil.NewStore(nil)
    
    // 创建服务
    sensitiveLevelService := v1.NewSensitiveLevelService(s)
    approvalFlowService := v1.NewApprovalFlowService(s)
    
    // 创建测试实例
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
    if err != nil {
        log.Fatalf("Failed to create instance: %v", err)
    }
    
    fmt.Println("=== 敏感数据分级与审批流功能示例 ===")
    fmt.Println()
    
    // 1. 创建敏感数据分级
    fmt.Println("1. 创建敏感数据分级配置")
    
    highLevelReq := &v1.CreateSensitiveLevelRequest{
        DisplayName: "高敏感数据",
        Description: "包含密码和银行卡号的高敏感数据",
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
    if err != nil {
        log.Fatalf("Failed to create high sensitive level: %v", err)
    }
    fmt.Printf("✓ 高敏感数据分级创建成功: %s\n", highLevelResp.SensitiveLevel.Name)
    
    mediumLevelReq := &v1.CreateSensitiveLevelRequest{
        DisplayName: "中敏感数据",
        Description: "包含邮箱和手机号的中敏感数据",
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
    if err != nil {
        log.Fatalf("Failed to create medium sensitive level: %v", err)
    }
    fmt.Printf("✓ 中敏感数据分级创建成功: %s\n", mediumLevelResp.SensitiveLevel.Name)
    
    lowLevelReq := &v1.CreateSensitiveLevelRequest{
        DisplayName: "低敏感数据",
        Description: "包含姓名和地址的低敏感数据",
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
    if err != nil {
        log.Fatalf("Failed to create low sensitive level: %v", err)
    }
    fmt.Printf("✓ 低敏感数据分级创建成功: %s\n", lowLevelResp.SensitiveLevel.Name)
    
    fmt.Println()
    
    // 2. 创建审批流
    fmt.Println("2. 创建审批流配置")
    
    highApprovalFlowReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "高敏感数据审批流程",
        Description:       "2 级审批流程：部门负责人 → DBA",
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
    if err != nil {
        log.Fatalf("Failed to create high approval flow: %v", err)
    }
    fmt.Printf("✓ 高敏感数据审批流程创建成功: %s\n", highApprovalFlowResp.ApprovalFlow.Name)
    
    mediumApprovalFlowReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "中敏感数据审批流程",
        Description:       "1 级审批流程：DBA",
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
    if err != nil {
        log.Fatalf("Failed to create medium approval flow: %v", err)
    }
    fmt.Printf("✓ 中敏感数据审批流程创建成功: %s\n", mediumApprovalFlowResp.ApprovalFlow.Name)
    
    lowApprovalFlowReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "低敏感数据审批流程",
        Description:       "无需审批，自动通过",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
        Steps: []*v1.ApprovalStep{},
    }
    
    lowApprovalFlowResp, err := approvalFlowService.CreateApprovalFlow(ctx, lowApprovalFlowReq)
    if err != nil {
        log.Fatalf("Failed to create low approval flow: %v", err)
    }
    fmt.Printf("✓ 低敏感数据审批流程创建成功: %s\n", lowApprovalFlowResp.ApprovalFlow.Name)
    
    fmt.Println()
    
    // 3. 测试高敏感数据审批流程
    fmt.Println("3. 测试高敏感数据审批流程")
    
    highSubmitReq := &v1.SubmitApprovalRequest{
        Title:             "更新用户密码",
        Description:       "更新 users 表中的密码字段",
        IssueId:           "issue-12345",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Submitter:         "developer@example.com",
    }
    
    highSubmitResp, err := approvalFlowService.SubmitApproval(ctx, highSubmitReq)
    if err != nil {
        log.Fatalf("Failed to submit high approval request: %v", err)
    }
    fmt.Printf("✓ 高敏感数据审批请求提交成功: %s\n", highSubmitResp.ApprovalRequest.Name)
    fmt.Printf("  当前状态: %v\n", highSubmitResp.ApprovalRequest.Status)
    
    // 部门负责人批准
    firstApproveReq := &v1.ApproveRequestRequest{
        ApprovalRequestId: highSubmitResp.ApprovalRequest.Name,
        Comment:           "批准此变更请求",
        Approver:          "department-manager@example.com",
    }
    
    firstApproveResp, err := approvalFlowService.ApproveRequest(ctx, firstApproveReq)
    if err != nil {
        log.Fatalf("Failed to approve first step: %v", err)
    }
    fmt.Printf("✓ 部门负责人已批准，当前状态: %v\n", firstApproveResp.ApprovalRequest.Status)
    
    // DBA 批准
    secondApproveReq := &v1.ApproveRequestRequest{
        ApprovalRequestId: highSubmitResp.ApprovalRequest.Name,
        Comment:           "批准此变更请求",
        Approver:          "dba@example.com",
    }
    
    secondApproveResp, err := approvalFlowService.ApproveRequest(ctx, secondApproveReq)
    if err != nil {
        log.Fatalf("Failed to approve second step: %v", err)
    }
    fmt.Printf("✓ DBA 已批准，当前状态: %v\n", secondApproveResp.ApprovalRequest.Status)
    
    fmt.Println()
    
    // 4. 测试中敏感数据审批流程
    fmt.Println("4. 测试中敏感数据审批流程")
    
    mediumSubmitReq := &v1.SubmitApprovalRequest{
        Title:             "更新用户邮箱",
        Description:       "更新 users 表中的邮箱字段",
        IssueId:           "issue-67890",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
        Submitter:         "developer@example.com",
    }
    
    mediumSubmitResp, err := approvalFlowService.SubmitApproval(ctx, mediumSubmitReq)
    if err != nil {
        log.Fatalf("Failed to submit medium approval request: %v", err)
    }
    fmt.Printf("✓ 中敏感数据审批请求提交成功: %s\n", mediumSubmitResp.ApprovalRequest.Name)
    fmt.Printf("  当前状态: %v\n", mediumSubmitResp.ApprovalRequest.Status)
    
    // DBA 批准
    mediumApproveReq := &v1.ApproveRequestRequest{
        ApprovalRequestId: mediumSubmitResp.ApprovalRequest.Name,
        Comment:           "批准此变更请求",
        Approver:          "dba@example.com",
    }
    
    mediumApproveResp, err := approvalFlowService.ApproveRequest(ctx, mediumApproveReq)
    if err != nil {
        log.Fatalf("Failed to approve medium request: %v", err)
    }
    fmt.Printf("✓ DBA 已批准，当前状态: %v\n", mediumApproveResp.ApprovalRequest.Status)
    
    fmt.Println()
    
    // 5. 测试低敏感数据审批流程
    fmt.Println("5. 测试低敏感数据审批流程")
    
    lowSubmitReq := &v1.SubmitApprovalRequest{
        Title:             "更新用户姓名",
        Description:       "更新 users 表中的姓名字段",
        IssueId:           "issue-11223",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
        Submitter:         "developer@example.com",
    }
    
    lowSubmitResp, err := approvalFlowService.SubmitApproval(ctx, lowSubmitReq)
    if err != nil {
        log.Fatalf("Failed to submit low approval request: %v", err)
    }
    fmt.Printf("✓ 低敏感数据审批请求提交成功: %s\n", lowSubmitResp.ApprovalRequest.Name)
    fmt.Printf("  当前状态: %v\n", lowSubmitResp.ApprovalRequest.Status)
    fmt.Println("  （低敏感数据无需审批，自动通过）")
    
    fmt.Println()
    
    // 6. 测试审批拒绝流程
    fmt.Println("6. 测试审批拒绝流程")
    
    rejectSubmitReq := &v1.SubmitApprovalRequest{
        Title:             "测试拒绝流程",
        Description:       "此请求将被部门负责人拒绝",
        IssueId:           "issue-44556",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Submitter:         "developer@example.com",
    }
    
    rejectSubmitResp, err := approvalFlowService.SubmitApproval(ctx, rejectSubmitReq)
    if err != nil {
        log.Fatalf("Failed to submit reject request: %v", err)
    }
    fmt.Printf("✓ 审批请求提交成功: %s\n", rejectSubmitResp.ApprovalRequest.Name)
    fmt.Printf("  当前状态: %v\n", rejectSubmitResp.ApprovalRequest.Status)
    
    // 部门负责人拒绝
    rejectReq := &v1.RejectRequestRequest{
        ApprovalRequestId: rejectSubmitResp.ApprovalRequest.Name,
        Comment:           "拒绝此变更请求，存在安全风险",
        Approver:          "department-manager@example.com",
    }
    
    rejectResp, err := approvalFlowService.RejectRequest(ctx, rejectReq)
    if err != nil {
        log.Fatalf("Failed to reject request: %v", err)
    }
    fmt.Printf("✓ 部门负责人已拒绝，当前状态: %v\n", rejectResp.ApprovalRequest.Status)
    fmt.Printf("  拒绝原因: %s\n", rejectResp.ApprovalRequest.ApprovalHistory[0].Comment)
    
    fmt.Println()
    
    // 7. 列出所有审批请求
    fmt.Println("7. 列出所有审批请求")
    
    listReq := &v1.ListApprovalRequestsRequest{}
    listResp, err := approvalFlowService.ListApprovalRequests(ctx, listReq)
    if err != nil {
        log.Fatalf("Failed to list approval requests: %v", err)
    }
    
    fmt.Printf("✓ 共找到 %d 个审批请求:\n", len(listResp.ApprovalRequests))
    for i, request := range listResp.ApprovalRequests {
        fmt.Printf("  %d. %s - %v - %v\n", i+1, request.Title, request.SensitivityLevel, request.Status)
    }
    
    fmt.Println()
    fmt.Println("=== 示例演示完成 ===")
    fmt.Println()
    fmt.Println("功能特点:")
    fmt.Println("1. 支持三级敏感数据分级（高、中、低）")
    fmt.Println("2. 自动根据数据敏感度触发不同审批流程")
    fmt.Println("3. 高敏感数据需要 2 级审批（部门负责人 → DBA）")
    fmt.Println("4. 中敏感数据需要 1 级审批（DBA）")
    fmt.Println("5. 低敏感数据无需审批，自动通过")
    fmt.Println("6. 完整的审批历史记录和状态跟踪")
    fmt.Println("7. 支持审批拒绝和原因记录")
}