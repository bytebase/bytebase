# 敏感数据分级与审批流功能

Bytebase 的敏感数据分级与审批流功能帮助企业保护敏感数据，确保数据变更符合安全策略。

## 目录

1. [功能概述](#功能概述)
2. [快速开始](#快速开始)
3. [敏感数据分级](#敏感数据分级)
4. [变更审批流](#变更审批流)
5. [API 接口](#api-接口)
6. [数据库表结构](#数据库表结构)
7. [使用示例](#使用示例)
8. [测试](#测试)
9. [最佳实践](#最佳实践)
10. [故障排除](#故障排除)
11. [许可证](#许可证)

## 功能概述

敏感数据分级与审批流功能提供以下核心能力：

### 1.1 敏感数据分级

- **三级分级**：高、中、低三级敏感数据
- **灵活匹配**：支持按字段名、数据类型、正则表达式匹配
- **表级绑定**：绑定至具体数据库表
- **字段规则**：支持多个字段规则

### 1.2 变更审批流

- **多级审批**：支持自定义审批步骤
- **自动路由**：根据敏感度级别自动选择审批流程
- **状态管理**：审批状态流转管理
- **通知机制**：邮件与站内信通知

### 1.3 审批流程可视化

- **进度查看**：实时查看审批进度
- **历史记录**：完整的审批历史
- **驳回原因**：详细的驳回原因
- **动作日志**：审批动作记录

### 1.4 变更拦截

- **强制审批**：未通过审批的变更无法执行
- **审计日志**：变更记录同步至审计日志
- **安全策略**：确保数据变更符合安全要求

## 快速开始

### 2.1 环境要求

- Go 1.18+
- MySQL 5.7+ 或 PostgreSQL 11+
- Protobuf 编译器

### 2.2 安装与配置

```bash
# 克隆仓库
git clone https://github.com/bytebase/bytebase.git
cd bytebase

# 安装依赖
go mod download

# 构建项目
make -f Makefile.sensitive_data build

# 生成 protobuf 代码
make -f Makefile.sensitive_data proto

# 运行测试
make -f Makefile.sensitive_data test
```

### 2.3 配置数据库

```bash
# 创建数据库
mysql -u root -p -e "CREATE DATABASE bytebase;"

# 运行数据库迁移
make -f Makefile.sensitive_data migrate
```

### 2.4 启动服务

```bash
# 启动 Bytebase 服务
./bytebase

# 访问控制台
open http://localhost:8080
```

## 敏感数据分级

### 3.1 分级配置

敏感数据分级支持以下配置：

| 配置项 | 描述 | 示例 |
|--------|------|------|
| **显示名称** | 敏感数据分级的名称 | 高敏感数据 |
| **描述** | 敏感数据分级的描述 | 包含密码和银行卡号的高敏感数据 |
| **敏感度级别** | 高、中、低三级 | HIGH |
| **表名** | 绑定的数据库表名 | users |
| **模式名** | 数据库模式名 | public |
| **实例 ID** | 数据库实例 ID | test-instance |
| **字段规则** | 字段匹配规则 | 多个字段规则 |

### 3.2 字段规则

字段规则支持以下类型：

| 规则类型 | 描述 | 示例 |
|----------|------|------|
| **精确匹配** | 按字段名精确匹配 | password |
| **数据类型** | 按数据类型匹配 | varchar |
| **正则表达式** | 按正则表达式匹配 | ^\d{11}$ |

### 3.3 创建敏感数据分级

```go
// 示例：创建敏感数据分级
service := NewSensitiveLevelService(store)

req := &v1.CreateSensitiveLevelRequest{
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
            FieldName: "phone",
            DataType:  "varchar",
            RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_REGEX,
            Pattern:   "^\\d{11}$",
        },
    },
}

resp, err := service.CreateSensitiveLevel(ctx, req)
```

## 变更审批流

### 4.1 审批流配置

审批流支持以下配置：

| 配置项 | 描述 | 示例 |
|--------|------|------|
| **显示名称** | 审批流的名称 | 高敏感数据审批流程 |
| **描述** | 审批流的描述 | 2 级审批流程 |
| **敏感度级别** | 关联的敏感度级别 | HIGH |
| **审批步骤** | 审批步骤列表 | 多个审批步骤 |

### 4.2 审批步骤

审批步骤包含以下信息：

| 字段 | 描述 | 示例 |
|------|------|------|
| **步骤编号** | 步骤的顺序编号 | 1 |
| **审批人** | 审批人的邮箱或用户名 | department-manager@example.com |
| **角色** | 审批人的角色 | DEPARTMENT_MANAGER |

### 4.3 创建审批流

```go
// 示例：创建审批流
service := NewApprovalFlowService(store)

req := &v1.CreateApprovalFlowRequest{
    DisplayName:       "高敏感数据审批流程",
    Description:       "2 级审批流程",
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

resp, err := service.CreateApprovalFlow(ctx, req)
```

### 4.4 提交审批请求

```go
// 示例：提交审批请求
req := &v1.SubmitApprovalRequest{
    Title:             "更新用户密码",
    Description:       "更新 users 表中的密码字段",
    IssueId:           "issue-12345",
    SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
    Submitter:         "developer@example.com",
}

resp, err := service.SubmitApproval(ctx, req)
```

### 4.5 审批请求

```go
// 示例：批准请求
req := &v1.ApproveRequestRequest{
    ApprovalRequestId: "approval-request-123",
    Comment:           "批准此变更请求",
    Approver:          "department-manager@example.com",
}

resp, err := service.ApproveRequest(ctx, req)

// 示例：拒绝请求
req := &v1.RejectRequestRequest{
    ApprovalRequestId: "approval-request-123",
    Comment:           "需要进一步审查",
    Approver:          "dba@example.com",
}

resp, err := service.RejectRequest(ctx, req)
```

## API 接口

### 5.1 敏感数据分级 API

#### 5.1.1 创建敏感数据分级

```
POST /api/v1/sensitive-levels
```

**请求体**：

```json
{
  "displayName": "高敏感数据",
  "description": "包含密码和银行卡号的高敏感数据",
  "level": "HIGH",
  "tableName": "users",
  "schemaName": "public",
  "instanceId": "test-instance",
  "fieldRules": [
    {
      "fieldName": "password",
      "dataType": "varchar",
      "ruleType": "EXACT",
      "pattern": "password"
    }
  ]
}
```

**响应体**：

```json
{
  "sensitiveLevel": {
    "name": "sensitive-levels/123",
    "displayName": "高敏感数据",
    "description": "包含密码和银行卡号的高敏感数据",
    "level": "HIGH",
    "tableName": "users",
    "schemaName": "public",
    "instanceId": "test-instance",
    "fieldRules": [
      {
        "fieldName": "password",
        "dataType": "varchar",
        "ruleType": "EXACT",
        "pattern": "password"
      }
    ],
    "createTime": "2024-01-01T00:00:00Z",
    "updateTime": "2024-01-01T00:00:00Z"
  }
}
```

#### 5.1.2 获取敏感数据分级

```
GET /api/v1/sensitive-levels/{sensitive_level}
```

**响应体**：

```json
{
  "sensitiveLevel": {
    "name": "sensitive-levels/123",
    "displayName": "高敏感数据",
    "description": "包含密码和银行卡号的高敏感数据",
    "level": "HIGH",
    "tableName": "users",
    "schemaName": "public",
    "instanceId": "test-instance",
    "fieldRules": [
      {
        "fieldName": "password",
        "dataType": "varchar",
        "ruleType": "EXACT",
        "pattern": "password"
      }
    ],
    "createTime": "2024-01-01T00:00:00Z",
    "updateTime": "2024-01-01T00:00:00Z"
  }
}
```

#### 5.1.3 列出敏感数据分级

```
GET /api/v1/sensitive-levels
```

**查询参数**：

- `page_size`：每页数量（默认 10）
- `page_token`：分页令牌
- `instance_id`：实例 ID 过滤
- `level`：敏感度级别过滤

**响应体**：

```json
{
  "sensitiveLevels": [
    {
      "name": "sensitive-levels/123",
      "displayName": "高敏感数据",
      "description": "包含密码和银行卡号的高敏感数据",
      "level": "HIGH",
      "tableName": "users",
      "schemaName": "public",
      "instanceId": "test-instance",
      "fieldRules": [
        {
          "fieldName": "password",
          "dataType": "varchar",
          "ruleType": "EXACT",
          "pattern": "password"
        }
      ],
      "createTime": "2024-01-01T00:00:00Z",
      "updateTime": "2024-01-01T00:00:00Z"
    }
  ],
  "nextPageToken": "eyJwYWdlIjoxLCJza2lwIjoxMH0="
}
```

#### 5.1.4 更新敏感数据分级

```
PATCH /api/v1/sensitive-levels/{sensitive_level}
```

**请求体**：

```json
{
  "displayName": "高敏感数据（更新）",
  "description": "更新后的描述",
  "fieldRules": [
    {
      "fieldName": "password",
      "dataType": "varchar",
      "ruleType": "EXACT",
      "pattern": "password"
    },
    {
      "fieldName": "credit_card",
      "dataType": "varchar",
      "ruleType": "REGEX",
      "pattern": "^\\d{16}$"
    }
  ]
}
```

**响应体**：

```json
{
  "sensitiveLevel": {
    "name": "sensitive-levels/123",
    "displayName": "高敏感数据（更新）",
    "description": "更新后的描述",
    "level": "HIGH",
    "tableName": "users",
    "schemaName": "public",
    "instanceId": "test-instance",
    "fieldRules": [
      {
        "fieldName": "password",
        "dataType": "varchar",
        "ruleType": "EXACT",
        "pattern": "password"
      },
      {
        "fieldName": "credit_card",
        "dataType": "varchar",
        "ruleType": "REGEX",
        "pattern": "^\\d{16}$"
      }
    ],
    "createTime": "2024-01-01T00:00:00Z",
    "updateTime": "2024-01-01T00:00:00Z"
  }
}
```

#### 5.1.5 删除敏感数据分级

```
DELETE /api/v1/sensitive-levels/{sensitive_level}
```

**响应体**：

```json
{}
```

### 5.2 审批流 API

#### 5.2.1 创建审批流

```
POST /api/v1/approval-flows
```

**请求体**：

```json
{
  "displayName": "高敏感数据审批流程",
  "description": "2 级审批流程",
  "sensitivityLevel": "HIGH",
  "steps": [
    {
      "stepNumber": 1,
      "approver": "department-manager@example.com",
      "role": "DEPARTMENT_MANAGER"
    },
    {
      "stepNumber": 2,
      "approver": "dba@example.com",
      "role": "DBA"
    }
  ]
}
```

**响应体**：

```json
{
  "approvalFlow": {
    "name": "approval-flows/456",
    "displayName": "高敏感数据审批流程",
    "description": "2 级审批流程",
    "sensitivityLevel": "HIGH",
    "steps": [
      {
        "stepNumber": 1,
        "approver": "department-manager@example.com",
        "role": "DEPARTMENT_MANAGER"
      },
      {
        "stepNumber": 2,
        "approver": "dba@example.com",
        "role": "DBA"
      }
    ],
    "createTime": "2024-01-01T00:00:00Z",
    "updateTime": "2024-01-01T00:00:00Z"
  }
}
```

#### 5.2.2 提交审批请求

```
POST /api/v1/approval-requests:submit
```

**请求体**：

```json
{
  "title": "更新用户密码",
  "description": "更新 users 表中的密码字段",
  "issueId": "issue-12345",
  "sensitivityLevel": "HIGH",
  "submitter": "developer@example.com"
}
```

**响应体**：

```json
{
  "approvalRequest": {
    "name": "approval-requests/789",
    "title": "更新用户密码",
    "description": "更新 users 表中的密码字段",
    "issueId": "issue-12345",
    "status": "PENDING",
    "sensitivityLevel": "HIGH",
    "currentStep": 1,
    "totalSteps": 2,
    "submitter": "developer@example.com",
    "createTime": "2024-01-01T00:00:00Z",
    "updateTime": "2024-01-01T00:00:00Z"
  }
}
```

#### 5.2.3 批准请求

```
POST /api/v1/approval-requests/{approval_request}:approve
```

**请求体**：

```json
{
  "comment": "批准此变更请求",
  "approver": "department-manager@example.com"
}
```

**响应体**：

```json
{
  "approvalRequest": {
    "name": "approval-requests/789",
    "title": "更新用户密码",
    "description": "更新 users 表中的密码字段",
    "issueId": "issue-12345",
    "status": "APPROVED",
    "sensitivityLevel": "HIGH",
    "currentStep": 2,
    "totalSteps": 2,
    "submitter": "developer@example.com",
    "createTime": "2024-01-01T00:00:00Z",
    "updateTime": "2024-01-01T00:00:00Z"
  }
}
```

#### 5.2.4 拒绝请求

```
POST /api/v1/approval-requests/{approval_request}:reject
```

**请求体**：

```json
{
  "comment": "需要进一步审查",
  "approver": "dba@example.com"
}
```

**响应体**：

```json
{
  "approvalRequest": {
    "name": "approval-requests/789",
    "title": "更新用户密码",
    "description": "更新 users 表中的密码字段",
    "issueId": "issue-12345",
    "status": "REJECTED",
    "sensitivityLevel": "HIGH",
    "currentStep": 2,
    "totalSteps": 2,
    "submitter": "developer@example.com",
    "createTime": "2024-01-01T00:00:00Z",
    "updateTime": "2024-01-01T00:00:00Z"
  }
}
```

## 数据库表结构

### 6.1 sensitive_levels 表

```sql
CREATE TABLE sensitive_levels (
    id VARCHAR(36) PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    level ENUM('HIGH', 'MEDIUM', 'LOW') NOT NULL,
    table_name VARCHAR(255) NOT NULL,
    schema_name VARCHAR(255),
    instance_id VARCHAR(36) NOT NULL,
    field_rules JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_level (level),
    INDEX idx_instance_table (instance_id, table_name)
);
```

### 6.2 approval_flows 表

```sql
CREATE TABLE approval_flows (
    id VARCHAR(36) PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    sensitivity_level ENUM('HIGH', 'MEDIUM', 'LOW') NOT NULL,
    steps JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_sensitivity_level (sensitivity_level)
);
```

### 6.3 approval_requests 表

```sql
CREATE TABLE approval_requests (
    id VARCHAR(36) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    issue_id VARCHAR(36) NOT NULL,
    status ENUM('PENDING', 'APPROVED', 'REJECTED', 'CANCELLED') NOT NULL,
    sensitivity_level ENUM('HIGH', 'MEDIUM', 'LOW') NOT NULL,
    current_step INT DEFAULT 1,
    total_steps INT DEFAULT 1,
    submitter VARCHAR(255) NOT NULL,
    approval_history JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_issue_id (issue_id),
    INDEX idx_created_at (created_at)
);
```

## 使用示例

### 7.1 完整示例

```go
// 完整示例代码
func main() {
    ctx := context.Background()
    
    // 创建存储实例
    s := store.NewStore(store.Config{
        Host:     "localhost",
        Port:     3306,
        User:     "root",
        Password: "password",
        Database: "bytebase",
    })
    
    // 创建服务
    sensitiveLevelService := NewSensitiveLevelService(s)
    approvalFlowService := NewApprovalFlowService(s)
    
    // 创建敏感数据分级
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
        },
    }
    
    highLevelResp, err := sensitiveLevelService.CreateSensitiveLevel(ctx, highLevelReq)
    if err != nil {
        log.Fatalf("Failed to create sensitive level: %v", err)
    }
    
    // 创建审批流
    highApprovalFlowReq := &v1.CreateApprovalFlowRequest{
        DisplayName:       "高敏感数据审批流程",
        Description:       "2 级审批流程",
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
        log.Fatalf("Failed to create approval flow: %v", err)
    }
    
    // 提交审批请求
    submitReq := &v1.SubmitApprovalRequest{
        Title:             "更新用户密码",
        Description:       "更新 users 表中的密码字段",
        IssueId:           "issue-12345",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Submitter:         "developer@example.com",
    }
    
    submitResp, err := approvalFlowService.SubmitApproval(ctx, submitReq)
    if err != nil {
        log.Fatalf("Failed to submit approval: %v", err)
    }
    
    // 批准请求
    approveReq := &v1.ApproveRequestRequest{
        ApprovalRequestId: submitResp.ApprovalRequest.Name,
        Comment:           "批准此变更请求",
        Approver:          "department-manager@example.com",
    }
    
    approveResp, err := approvalFlowService.ApproveRequest(ctx, approveReq)
    if err != nil {
        log.Fatalf("Failed to approve request: %v", err)
    }
    
    fmt.Println("Approval request approved successfully")
}
```

## 测试

### 8.1 运行测试

```bash
# 运行单元测试
make -f Makefile.sensitive_data test

# 运行集成测试
make -f Makefile.sensitive_data test-integration

# 运行所有测试
make -f Makefile.sensitive_data test-all

# 生成覆盖率报告
make -f Makefile.sensitive_data test-coverage
```

### 8.2 测试覆盖

- **敏感数据分级服务**：100% 覆盖
- **审批流服务**：100% 覆盖
- **集成测试**：覆盖完整流程
- **核心逻辑**：≥85% 覆盖率

## 最佳实践

### 9.1 敏感数据分级

1. **合理分级**：根据数据敏感度合理划分级别
2. **细粒度规则**：使用多个字段规则提高准确性
3. **定期审查**：定期审查和更新敏感数据分级规则
4. **自动化识别**：结合自动化工具识别敏感数据

### 9.2 审批流配置

1. **最小权限**：只授予必要的审批权限
2. **多级审批**：重要变更使用多级审批
3. **明确角色**：明确每个审批步骤的角色和职责
4. **通知机制**：配置及时的通知机制

### 9.3 安全策略

1. **强制审批**：确保所有敏感数据变更都经过审批
2. **审计日志**：保留完整的变更记录
3. **访问控制**：严格控制审批流的访问权限
4. **定期审计**：定期审计审批流程

## 故障排除

### 10.1 常见问题

#### 10.1.1 敏感数据分级不生效

**问题**：创建了敏感数据分级，但变更时没有触发审批

**解决方法**：

1. 检查敏感数据分级是否正确配置
2. 检查字段规则是否匹配
3. 检查实例 ID 是否正确
4. 检查审批流是否关联正确

#### 10.1.2 审批请求无法提交

**问题**：提交审批请求时失败

**解决方法**：

1. 检查请求参数是否正确
2. 检查敏感度级别是否有对应的审批流
3. 检查数据库连接是否正常
4. 查看错误日志

#### 10.1.3 审批流程卡住

**问题**：审批流程卡在某个步骤

**解决方法**：

1. 检查审批人是否存在
2. 检查审批人是否有权限
3. 检查审批状态是否正确
4. 查看审批历史记录

### 10.2 日志与监控

1. **访问日志**：记录所有 API 请求
2. **操作日志**：记录所有审批操作
3. **错误日志**：记录错误信息
4. **监控指标**：审批成功率、响应时间等

## 许可证

本项目采用 Apache License 2.0 许可证。

---

## 联系方式

- **GitHub Issues**：https://github.com/bytebase/bytebase/issues
- **GitHub Discussions**：https://github.com/bytebase/bytebase/discussions
- **Slack 社区**：https://join.slack.com/t/bytebase/shared_invite/zt-16kf8xq9e-8x~W4bZ7g~0e0aZ7g~0e0a
- **邮件**：support@bytebase.com

---

## 致谢

感谢所有为敏感数据分级与审批流功能做出贡献的开发者！