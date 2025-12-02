# 敏感数据分级与变更审批流功能

## 功能概述

Bytebase 新增了敏感数据分级与变更审批流功能，帮助企业更好地管理和保护敏感数据。该功能包括以下核心模块：

1. **敏感数据分级配置**：支持按字段名、数据类型、正则表达式匹配敏感字段
2. **变更审批流**：根据数据敏感度自动触发不同级别的审批流程
3. **审批流程可视化**：支持查看审批进度、驳回原因和历史记录
4. **变更拦截逻辑**：未通过审批的敏感数据变更无法执行
5. **API 接口**：提供完整的 CRUD 操作接口

## 敏感数据分级

### 敏感度等级

系统支持三级敏感数据分级：

- **高敏感**：密码、银行卡号等核心敏感数据，需要 2 级审批
- **中敏感**：邮箱、手机号等个人信息，需要 1 级审批  
- **低敏感**：姓名、地址等基本信息，无需审批

### 配置方式

敏感数据分级支持以下匹配规则：

1. **字段名匹配**：按字段名精确匹配
2. **数据类型匹配**：按数据类型匹配（如 varchar、text 等）
3. **正则表达式匹配**：按正则表达式模式匹配

### 示例配置

```protobuf
// 高敏感数据配置
SensitiveLevel {
  display_name: "用户敏感信息"
  description: "包含密码和银行卡号的高敏感数据"
  level: SENSITIVITY_LEVEL_HIGH
  table_name: "users"
  schema_name: "public"
  instance_id: "mysql-prod"
  field_rules: [
    {
      field_name: "password"
      data_type: "varchar"
      rule_type: MATCHING_RULE_TYPE_EXACT
      pattern: "password"
    },
    {
      field_name: "credit_card"
      data_type: "varchar"
      rule_type: MATCHING_RULE_TYPE_REGEX
      pattern: "^\\d{16}$"
    }
  ]
}
```

## 变更审批流

### 审批流程

系统根据数据敏感度自动触发不同的审批流程：

- **高敏感数据**：申请人 → 部门负责人 → DBA（2 级审批）
- **中敏感数据**：申请人 → DBA（1 级审批）
- **低敏感数据**：无需审批，自动通过

### 审批状态

审批请求有以下状态：

- **待审批**：等待审批人处理
- **审批中**：正在审批流程中
- **已批准**：所有审批步骤通过
- **已拒绝**：任何一步被拒绝
- **已取消**：申请人取消请求

### 审批动作

审批人可以执行以下操作：

- **批准**：同意变更请求
- **拒绝**：拒绝变更请求，需要填写原因
- **转发**：转发给其他审批人

## API 接口

### 敏感数据分级接口

#### 创建敏感数据分级

```bash
POST /api/v1/sensitive-levels
```

请求体：
```json
{
  "display_name": "用户敏感信息",
  "description": "包含密码和银行卡号的高敏感数据",
  "level": "SENSITIVITY_LEVEL_HIGH",
  "table_name": "users",
  "schema_name": "public",
  "instance_id": "mysql-prod",
  "field_rules": [
    {
      "field_name": "password",
      "data_type": "varchar",
      "rule_type": "MATCHING_RULE_TYPE_EXACT",
      "pattern": "password"
    }
  ]
}
```

#### 获取敏感数据分级

```bash
GET /api/v1/sensitive-levels/{sensitive_level_id}
```

#### 列出敏感数据分级

```bash
GET /api/v1/sensitive-levels?parent=instances/{instance_id}
```

#### 更新敏感数据分级

```bash
PATCH /api/v1/sensitive-levels/{sensitive_level_id}
```

#### 删除敏感数据分级

```bash
DELETE /api/v1/sensitive-levels/{sensitive_level_id}
```

### 审批流接口

#### 创建审批流

```bash
POST /api/v1/approval-flows
```

请求体：
```json
{
  "display_name": "高敏感数据审批流程",
  "description": "2 级审批流程",
  "sensitivity_level": "SENSITIVITY_LEVEL_HIGH",
  "steps": [
    {
      "step_number": 1,
      "approver": "department-manager@example.com",
      "role": "DEPARTMENT_MANAGER"
    },
    {
      "step_number": 2,
      "approver": "dba@example.com",
      "role": "DBA"
    }
  ]
}
```

#### 提交审批请求

```bash
POST /api/v1/approval-requests:submit
```

请求体：
```json
{
  "title": "更新用户密码",
  "description": "更新 users 表中的密码字段",
  "issue_id": "issue-12345",
  "sensitivity_level": "SENSITIVITY_LEVEL_HIGH",
  "submitter": "developer@example.com"
}
```

#### 批准请求

```bash
POST /api/v1/approval-requests/{approval_request_id}:approve
```

请求体：
```json
{
  "comment": "批准此变更请求",
  "approver": "dba@example.com"
}
```

#### 拒绝请求

```bash
POST /api/v1/approval-requests/{approval_request_id}:reject
```

请求体：
```json
{
  "comment": "拒绝此变更请求，存在安全风险",
  "approver": "department-manager@example.com"
}
```

## 数据库表结构

### sensitive_levels 表

存储敏感数据分级配置：

| 字段 | 类型 | 描述 |
|------|------|------|
| id | VARCHAR(255) | 主键 |
| display_name | VARCHAR(255) | 显示名称 |
| description | TEXT | 描述 |
| level | INT | 敏感度等级 |
| table_name | VARCHAR(255) | 表名 |
| schema_name | VARCHAR(255) | 模式名 |
| instance_id | VARCHAR(255) | 实例 ID |
| field_rules | TEXT | 字段规则（JSON 格式） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### approval_flows 表

存储审批流配置：

| 字段 | 类型 | 描述 |
|------|------|------|
| id | VARCHAR(255) | 主键 |
| display_name | VARCHAR(255) | 显示名称 |
| description | TEXT | 描述 |
| sensitivity_level | INT | 敏感度等级 |
| steps | TEXT | 审批步骤（JSON 格式） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### approval_requests 表

存储审批请求记录：

| 字段 | 类型 | 描述 |
|------|------|------|
| id | VARCHAR(255) | 主键 |
| title | VARCHAR(255) | 请求标题 |
| description | TEXT | 请求描述 |
| issue_id | VARCHAR(255) | 关联的 issue ID |
| sensitivity_level | INT | 敏感度等级 |
| approval_flow_id | VARCHAR(255) | 关联的审批流 ID |
| status | INT | 审批状态 |
| submitter | VARCHAR(255) | 申请人 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

## 使用示例

### 1. 配置敏感数据分级

```bash
# 创建高敏感数据配置
curl -X POST http://localhost:8080/api/v1/sensitive-levels \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "用户敏感信息",
    "description": "包含密码和银行卡号的高敏感数据",
    "level": "SENSITIVITY_LEVEL_HIGH",
    "table_name": "users",
    "schema_name": "public",
    "instance_id": "mysql-prod",
    "field_rules": [
      {
        "field_name": "password",
        "data_type": "varchar",
        "rule_type": "MATCHING_RULE_TYPE_EXACT",
        "pattern": "password"
      }
    ]
  }'
```

### 2. 配置审批流

```bash
# 创建高敏感数据审批流程
curl -X POST http://localhost:8080/api/v1/approval-flows \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "高敏感数据审批流程",
    "description": "2 级审批流程",
    "sensitivity_level": "SENSITIVITY_LEVEL_HIGH",
    "steps": [
      {
        "step_number": 1,
        "approver": "department-manager@example.com",
        "role": "DEPARTMENT_MANAGER"
      },
      {
        "step_number": 2,
        "approver": "dba@example.com",
        "role": "DBA"
      }
    ]
  }'
```

### 3. 提交审批请求

```bash
# 提交高敏感数据变更审批请求
curl -X POST http://localhost:8080/api/v1/approval-requests:submit \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "更新用户密码",
    "description": "更新 users 表中的密码字段",
    "issue_id": "issue-12345",
    "sensitivity_level": "SENSITIVITY_LEVEL_HIGH",
    "submitter": "developer@example.com"
  }'
```

### 4. 审批请求

```bash
# 部门负责人批准请求
curl -X POST http://localhost:8080/api/v1/approval-requests/{request_id}:approve \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "comment": "批准此变更请求",
    "approver": "department-manager@example.com"
  }'

# DBA 批准请求
curl -X POST http://localhost:8080/api/v1/approval-requests/{request_id}:approve \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "comment": "批准此变更请求",
    "approver": "dba@example.com"
  }'
```

## 测试

### 单元测试

```bash
# 运行敏感数据分级服务单元测试
go test ./backend/api/v1 -run TestSensitiveLevelService

# 运行审批流服务单元测试
go test ./backend/api/v1 -run TestApprovalFlowService
```

### 集成测试

```bash
# 运行集成测试
go test ./backend/api/v1 -run TestSensitiveLevelAndApprovalFlowIntegration
```

## 最佳实践

1. **合理分级**：根据数据实际敏感度进行分级，避免过度审批
2. **最小权限**：审批人应只拥有必要的审批权限
3. **定期审计**：定期审查敏感数据分级配置和审批记录
4. **自动化**：对于低敏感数据，可配置自动审批
5. **监控告警**：设置审批超时和异常操作告警

## 故障排除

### 常见问题

1. **审批请求未触发**：检查敏感数据分级配置是否正确
2. **审批人收不到通知**：检查邮件配置和用户邮箱
3. **审批流程卡住**：检查审批人权限和审批步骤配置
4. **API 调用失败**：检查 OAuth2.0 令牌和权限

### 日志查看

```bash
# 查看应用日志
tail -f /var/log/bytebase/bytebase.log

# 查看审计日志
curl -X GET http://localhost:8080/api/v1/audit-logs \
  -H "Authorization: Bearer $TOKEN"
```

## 版本历史

- **v1.0.0**：初始版本，包含敏感数据分级和基本审批流功能
- **v1.1.0**：添加审批流程可视化和通知功能
- **v1.2.0**：优化变更拦截逻辑和审计日志集成
- **v1.3.0**：增强 API 接口和测试覆盖率