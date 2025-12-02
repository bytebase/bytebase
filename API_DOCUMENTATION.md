# 敏感数据分级与审批流功能API文档

## 概述

本文档详细描述了敏感数据分级与审批流功能的所有API接口，包括：
- 敏感数据分级服务API
- 审批流服务API
- 错误码说明
- 认证和授权
- 最佳实践

## 服务概述

### 敏感数据分级服务
- **服务名称**: SensitiveLevelService
- **gRPC端口**: 50051
- **HTTP端口**: 8080
- **协议**: gRPC + HTTP/JSON

### 审批流服务
- **服务名称**: ApprovalFlowService
- **gRPC端口**: 50052
- **HTTP端口**: 8081
- **协议**: gRPC + HTTP/JSON

## 敏感数据分级服务API

### 1. CreateSensitiveLevel

**功能**: 创建敏感数据分级

**gRPC接口**:
```proto
rpc CreateSensitiveLevel(CreateSensitiveLevelRequest) returns (CreateSensitiveLevelResponse) {
  option (google.api.http) = {
    post: "/v1/sensitive-levels"
    body: "*"
  };
}
```

**请求参数**:
```proto
message CreateSensitiveLevelRequest {
  // 敏感数据分级名称（必填）
  string name = 1;
  
  // 敏感级别（1-5，必填）
  int32 level = 2;
  
  // 描述（可选）
  string description = 3;
  
  // 字段规则（可选）
  repeated FieldRule field_rules = 4;
}

message FieldRule {
  // 字段名称（必填）
  string field_name = 1;
  
  // 匹配模式（正则表达式，必填）
  string pattern = 2;
  
  // 处理动作（必填）
  Action action = 3;
}

enum Action {
  // 未知动作
  ACTION_UNSPECIFIED = 0;
  
  // 加密
  ENCRYPT = 1;
  
  // 脱敏
  MASK = 2;
  
  // 拒绝
  REJECT = 3;
  
  // 审计
  AUDIT = 4;
}
```

**响应参数**:
```proto
message CreateSensitiveLevelResponse {
  // 创建的敏感数据分级
  SensitiveLevel sensitive_level = 1;
}

message SensitiveLevel {
  // 唯一标识符
  string id = 1;
  
  // 名称
  string name = 2;
  
  // 敏感级别
  int32 level = 3;
  
  // 描述
  string description = 4;
  
  // 字段规则
  repeated FieldRule field_rules = 5;
  
  // 创建时间
  google.protobuf.Timestamp created_at = 6;
  
  // 更新时间
  google.protobuf.Timestamp updated_at = 7;
}
```

**HTTP示例**:
```bash
curl -X POST http://localhost:8080/v1/sensitive-levels \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高敏感数据",
    "level": 3,
    "description": "包含个人身份信息的敏感数据",
    "field_rules": [
      {
        "field_name": "身份证号",
        "pattern": "^[1-9]\\d{5}(18|19|20)\\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\\d{3}[0-9Xx]$",
        "action": "ENCRYPT"
      }
    ]
  }'
```

**响应示例**:
```json
{
  "sensitive_level": {
    "id": "sl_1234567890",
    "name": "高敏感数据",
    "level": 3,
    "description": "包含个人身份信息的敏感数据",
    "field_rules": [
      {
        "field_name": "身份证号",
        "pattern": "^[1-9]\\d{5}(18|19|20)\\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\\d{3}[0-9Xx]$",
        "action": "ENCRYPT"
      }
    ],
    "created_at": "2024-01-01T12:00:00Z",
    "updated_at": "2024-01-01T12:00:00Z"
  }
}
```

### 2. GetSensitiveLevel

**功能**: 获取敏感数据分级详情

**gRPC接口**:
```proto
rpc GetSensitiveLevel(GetSensitiveLevelRequest) returns (GetSensitiveLevelResponse) {
  option (google.api.http) = {
    get: "/v1/sensitive-levels/{id}"
  };
}
```

**请求参数**:
```proto
message GetSensitiveLevelRequest {
  // 敏感数据分级ID（必填）
  string id = 1;
}
```

**响应参数**:
```proto
message GetSensitiveLevelResponse {
  // 敏感数据分级
  SensitiveLevel sensitive_level = 1;
}
```

**HTTP示例**:
```bash
curl http://localhost:8080/v1/sensitive-levels/sl_1234567890
```

### 3. ListSensitiveLevels

**功能**: 列出敏感数据分级

**gRPC接口**:
```proto
rpc ListSensitiveLevels(ListSensitiveLevelsRequest) returns (ListSensitiveLevelsResponse) {
  option (google.api.http) = {
    get: "/v1/sensitive-levels"
  };
}
```

**请求参数**:
```proto
message ListSensitiveLevelsRequest {
  // 分页大小（默认10，最大100）
  int32 page_size = 1;
  
  // 分页令牌
  string page_token = 2;
  
  // 按级别过滤
  int32 level = 3;
  
  // 按名称过滤（模糊匹配）
  string name_contains = 4;
}
```

**响应参数**:
```proto
message ListSensitiveLevelsResponse {
  // 敏感数据分级列表
  repeated SensitiveLevel sensitive_levels = 1;
  
  // 下一页令牌
  string next_page_token = 2;
  
  // 总数
  int32 total_size = 3;
}
```

**HTTP示例**:
```bash
curl http://localhost:8080/v1/sensitive-levels?page_size=20&level=3
```

### 4. UpdateSensitiveLevel

**功能**: 更新敏感数据分级

**gRPC接口**:
```proto
rpc UpdateSensitiveLevel(UpdateSensitiveLevelRequest) returns (UpdateSensitiveLevelResponse) {
  option (google.api.http) = {
    put: "/v1/sensitive-levels/{id}"
    body: "*"
  };
}
```

**请求参数**:
```proto
message UpdateSensitiveLevelRequest {
  // 敏感数据分级ID（必填）
  string id = 1;
  
  // 名称（可选）
  string name = 2;
  
  // 敏感级别（可选）
  int32 level = 3;
  
  // 描述（可选）
  string description = 4;
  
  // 字段规则（可选）
  repeated FieldRule field_rules = 5;
}
```

**响应参数**:
```proto
message UpdateSensitiveLevelResponse {
  // 更新后的敏感数据分级
  SensitiveLevel sensitive_level = 1;
}
```

**HTTP示例**:
```bash
curl -X PUT http://localhost:8080/v1/sensitive-levels/sl_1234567890 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高敏感数据(更新)",
    "level": 3,
    "description": "更新后的描述"
  }'
```

### 5. DeleteSensitiveLevel

**功能**: 删除敏感数据分级

**gRPC接口**:
```proto
rpc DeleteSensitiveLevel(DeleteSensitiveLevelRequest) returns (DeleteSensitiveLevelResponse) {
  option (google.api.http) = {
    delete: "/v1/sensitive-levels/{id}"
  };
}
```

**请求参数**:
```proto
message DeleteSensitiveLevelRequest {
  // 敏感数据分级ID（必填）
  string id = 1;
}
```

**响应参数**:
```proto
message DeleteSensitiveLevelResponse {
  // 空响应
}
```

**HTTP示例**:
```bash
curl -X DELETE http://localhost:8080/v1/sensitive-levels/sl_1234567890
```

## 审批流服务API

### 1. CreateApprovalFlow

**功能**: 创建审批流

**gRPC接口**:
```proto
rpc CreateApprovalFlow(CreateApprovalFlowRequest) returns (CreateApprovalFlowResponse) {
  option (google.api.http) = {
    post: "/v1/approval-flows"
    body: "*"
  };
}
```

**请求参数**:
```proto
message CreateApprovalFlowRequest {
  // 审批流名称（必填）
  string name = 1;
  
  // 描述（可选）
  string description = 2;
  
  // 关联的敏感数据分级ID（必填）
  string sensitive_level_id = 3;
  
  // 审批步骤（必填）
  repeated ApprovalStep steps = 4;
  
  // 是否自动审批（默认false）
  bool auto_approve = 5;
}

message ApprovalStep {
  // 步骤名称（必填）
  string name = 1;
  
  // 审批类型（必填）
  ApprovalType type = 2;
  
  // 需要的审批人数（默认1）
  int32 required_approvers = 3;
  
  // 超时时间（秒，默认86400）
  int32 timeout = 4;
}

enum ApprovalType {
  // 未知类型
  APPROVAL_TYPE_UNSPECIFIED = 0;
  
  // 直接主管
  DIRECT_MANAGER = 1;
  
  // 部门经理
  MANAGER = 2;
  
  // 数据安全官
  DATA_SECURITY_OFFICER = 3;
  
  // 自定义用户
  CUSTOM_USER = 4;
  
  // 自定义组
  CUSTOM_GROUP = 5;
}
```

**响应参数**:
```proto
message CreateApprovalFlowResponse {
  // 创建的审批流
  ApprovalFlow approval_flow = 1;
}

message ApprovalFlow {
  // 唯一标识符
  string id = 1;
  
  // 名称
  string name = 2;
  
  // 描述
  string description = 3;
  
  // 关联的敏感数据分级ID
  string sensitive_level_id = 4;
  
  // 审批步骤
  repeated ApprovalStep steps = 5;
  
  // 是否自动审批
  bool auto_approve = 6;
  
  // 创建时间
  google.protobuf.Timestamp created_at = 7;
  
  // 更新时间
  google.protobuf.Timestamp updated_at = 8;
}
```

**HTTP示例**:
```bash
curl -X POST http://localhost:8081/v1/approval-flows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高敏感数据审批流",
    "description": "高敏感数据变更需要2级审批",
    "sensitive_level_id": "sl_1234567890",
    "steps": [
      {
        "name": "部门经理审批",
        "type": "MANAGER",
        "required_approvers": 1,
        "timeout": 86400
      },
      {
        "name": "数据安全官审批",
        "type": "DATA_SECURITY_OFFICER",
        "required_approvers": 1,
        "timeout": 86400
      }
    ],
    "auto_approve": false
  }'
```

### 2. GetApprovalFlow

**功能**: 获取审批流详情

**gRPC接口**:
```proto
rpc GetApprovalFlow(GetApprovalFlowRequest) returns (GetApprovalFlowResponse) {
  option (google.api.http) = {
    get: "/v1/approval-flows/{id}"
  };
}
```

**请求参数**:
```proto
message GetApprovalFlowRequest {
  // 审批流ID（必填）
  string id = 1;
}
```

**响应参数**:
```proto
message GetApprovalFlowResponse {
  // 审批流
  ApprovalFlow approval_flow = 1;
}
```

**HTTP示例**:
```bash
curl http://localhost:8081/v1/approval-flows/af_1234567890
```

### 3. ListApprovalFlows

**功能**: 列出审批流

**gRPC接口**:
```proto
rpc ListApprovalFlows(ListApprovalFlowsRequest) returns (ListApprovalFlowsResponse) {
  option (google.api.http) = {
    get: "/v1/approval-flows"
  };
}
```

**请求参数**:
```proto
message ListApprovalFlowsRequest {
  // 分页大小（默认10，最大100）
  int32 page_size = 1;
  
  // 分页令牌
  string page_token = 2;
  
  // 按敏感数据分级ID过滤
  string sensitive_level_id = 3;
  
  // 按名称过滤（模糊匹配）
  string name_contains = 4;
}
```

**响应参数**:
```proto
message ListApprovalFlowsResponse {
  // 审批流列表
  repeated ApprovalFlow approval_flows = 1;
  
  // 下一页令牌
  string next_page_token = 2;
  
  // 总数
  int32 total_size = 3;
}
```

**HTTP示例**:
```bash
curl http://localhost:8081/v1/approval-flows?sensitive_level_id=sl_1234567890
```

### 4. UpdateApprovalFlow

**功能**: 更新审批流

**gRPC接口**:
```proto
rpc UpdateApprovalFlow(UpdateApprovalFlowRequest) returns (UpdateApprovalFlowResponse) {
  option (google.api.http) = {
    put: "/v1/approval-flows/{id}"
    body: "*"
  };
}
```

**请求参数**:
```proto
message UpdateApprovalFlowRequest {
  // 审批流ID（必填）
  string id = 1;
  
  // 名称（可选）
  string name = 2;
  
  // 描述（可选）
  string description = 3;
  
  // 审批步骤（可选）
  repeated ApprovalStep steps = 4;
  
  // 是否自动审批（可选）
  bool auto_approve = 5;
}
```

**响应参数**:
```proto
message UpdateApprovalFlowResponse {
  // 更新后的审批流
  ApprovalFlow approval_flow = 1;
}
```

**HTTP示例**:
```bash
curl -X PUT http://localhost:8081/v1/approval-flows/af_1234567890 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高敏感数据审批流(更新)",
    "description": "更新后的描述",
    "steps": [
      {
        "name": "部门经理审批",
        "type": "MANAGER",
        "required_approvers": 1,
        "timeout": 86400
      }
    ]
  }'
```

### 5. DeleteApprovalFlow

**功能**: 删除审批流

**gRPC接口**:
```proto
rpc DeleteApprovalFlow(DeleteApprovalFlowRequest) returns (DeleteApprovalFlowResponse) {
  option (google.api.http) = {
    delete: "/v1/approval-flows/{id}"
  };
}
```

**请求参数**:
```proto
message DeleteApprovalFlowRequest {
  // 审批流ID（必填）
  string id = 1;
}
```

**响应参数**:
```proto
message DeleteApprovalFlowResponse {
  // 空响应
}
```

**HTTP示例**:
```bash
curl -X DELETE http://localhost:8081/v1/approval-flows/af_1234567890
```

### 6. SubmitApproval

**功能**: 提交审批请求

**gRPC接口**:
```proto
rpc SubmitApproval(SubmitApprovalRequest) returns (SubmitApprovalResponse) {
  option (google.api.http) = {
    post: "/v1/approval-requests"
    body: "*"
  };
}
```

**请求参数**:
```proto
message SubmitApprovalRequest {
  // 审批流ID（必填）
  string approval_flow_id = 1;
  
  // 申请人ID（必填）
  string requester_id = 2;
  
  // 申请人姓名（必填）
  string requester_name = 3;
  
  // 变更类型（必填）
  ChangeType change_type = 4;
  
  // 变更详情（必填）
  ChangeDetails change_details = 5;
  
  // 变更理由（必填）
  string reason = 6;
  
  // 紧急程度（默认NORMAL）
  Urgency urgency = 7;
}

enum ChangeType {
  // 未知类型
  CHANGE_TYPE_UNSPECIFIED = 0;
  
  // 创建
  CREATE = 1;
  
  // 更新
  UPDATE = 2;
  
  // 删除
  DELETE = 3;
  
  // 查询
  QUERY = 4;
}

message ChangeDetails {
  // 表名（必填）
  string table = 1;
  
  // 字段名（必填）
  string field = 2;
  
  // 旧值（可选）
  string old_value = 3;
  
  // 新值（可选）
  string new_value = 4;
  
  // 变更数据（JSON格式，可选）
  string data = 5;
}

enum Urgency {
  // 未知紧急程度
  URGENCY_UNSPECIFIED = 0;
  
  // 普通
  NORMAL = 1;
  
  // 紧急
  URGENT = 2;
  
  // 非常紧急
  CRITICAL = 3;
}
```

**响应参数**:
```proto
message SubmitApprovalResponse {
  // 创建的审批请求
  ApprovalRequest approval_request = 1;
}

message ApprovalRequest {
  // 唯一标识符
  string id = 1;
  
  // 审批流ID
  string approval_flow_id = 2;
  
  // 申请人ID
  string requester_id = 3;
  
  // 申请人姓名
  string requester_name = 4;
  
  // 变更类型
  ChangeType change_type = 5;
  
  // 变更详情
  ChangeDetails change_details = 6;
  
  // 变更理由
  string reason = 7;
  
  // 紧急程度
  Urgency urgency = 8;
  
  // 状态
  ApprovalStatus status = 9;
  
  // 当前步骤索引
  int32 current_step_index = 10;
  
  // 审批历史
  repeated ApprovalHistory history = 11;
  
  // 创建时间
  google.protobuf.Timestamp created_at = 12;
  
  // 更新时间
  google.protobuf.Timestamp updated_at = 13;
  
  // 完成时间
  google.protobuf.Timestamp completed_at = 14;
}

enum ApprovalStatus {
  // 未知状态
  APPROVAL_STATUS_UNSPECIFIED = 0;
  
  // 待审批
  PENDING = 1;
  
  // 审批中
  IN_PROGRESS = 2;
  
  // 已批准
  APPROVED = 3;
  
  // 已拒绝
  REJECTED = 4;
  
  // 已取消
  CANCELLED = 5;
  
  // 已超时
  TIMEOUT = 6;
}

message ApprovalHistory {
  // 步骤索引
  int32 step_index = 1;
  
  // 审批人ID
  string approver_id = 2;
  
  // 审批人姓名
  string approver_name = 3;
  
  // 审批结果
  ApprovalResult result = 4;
  
  // 审批意见
  string comment = 5;
  
  // 审批时间
  google.protobuf.Timestamp approved_at = 6;
}

enum ApprovalResult {
  // 未知结果
  APPROVAL_RESULT_UNSPECIFIED = 0;
  
  // 批准
  APPROVE = 1;
  
  // 拒绝
  REJECT = 2;
  
  // 弃权
  ABSTAIN = 3;
}
```

**HTTP示例**:
```bash
curl -X POST http://localhost:8081/v1/approval-requests \
  -H "Content-Type: application/json" \
  -d '{
    "approval_flow_id": "af_1234567890",
    "requester_id": "user_123",
    "requester_name": "张三",
    "change_type": "UPDATE",
    "change_details": {
      "table": "users",
      "field": "身份证号",
      "old_value": "310101199001011234",
      "new_value": "310101199001011235"
    },
    "reason": "用户身份证号变更",
    "urgency": "NORMAL"
  }'
```

### 7. GetApprovalRequest

**功能**: 获取审批请求详情

**gRPC接口**:
```proto
rpc GetApprovalRequest(GetApprovalRequestRequest) returns (GetApprovalRequestResponse) {
  option (google.api.http) = {
    get: "/v1/approval-requests/{id}"
  };
}
```

**请求参数**:
```proto
message GetApprovalRequestRequest {
  // 审批请求ID（必填）
  string id = 1;
}
```

**响应参数**:
```proto
message GetApprovalRequestResponse {
  // 审批请求
  ApprovalRequest approval_request = 1;
}
```

**HTTP示例**:
```bash
curl http://localhost:8081/v1/approval-requests/ar_1234567890
```

### 8. ListApprovalRequests

**功能**: 列出审批请求

**gRPC接口**:
```proto
rpc ListApprovalRequests(ListApprovalRequestsRequest) returns (ListApprovalRequestsResponse) {
  option (google.api.http) = {
    get: "/v1/approval-requests"
  };
}
```

**请求参数**:
```proto
message ListApprovalRequestsRequest {
  // 分页大小（默认10，最大100）
  int32 page_size = 1;
  
  // 分页令牌
  string page_token = 2;
  
  // 按审批流ID过滤
  string approval_flow_id = 3;
  
  // 按申请人ID过滤
  string requester_id = 4;
  
  // 按状态过滤
  ApprovalStatus status = 5;
  
  // 按紧急程度过滤
  Urgency urgency = 6;
  
  // 按创建时间范围过滤
  google.protobuf.Timestamp created_after = 7;
  google.protobuf.Timestamp created_before = 8;
}
```

**响应参数**:
```proto
message ListApprovalRequestsResponse {
  // 审批请求列表
  repeated ApprovalRequest approval_requests = 1;
  
  // 下一页令牌
  string next_page_token = 2;
  
  // 总数
  int32 total_size = 3;
}
```

**HTTP示例**:
```bash
curl http://localhost:8081/v1/approval-requests?status=PENDING&urgency=URGENT
```

### 9. ApproveRequest

**功能**: 批准审批请求

**gRPC接口**:
```proto
rpc ApproveRequest(ApproveRequestRequest) returns (ApproveRequestResponse) {
  option (google.api.http) = {
    post: "/v1/approval-requests/{id}/approve"
    body: "*"
  };
}
```

**请求参数**:
```proto
message ApproveRequestRequest {
  // 审批请求ID（必填）
  string id = 1;
  
  // 审批人ID（必填）
  string approver_id = 2;
  
  // 审批人姓名（必填）
  string approver_name = 3;
  
  // 审批意见（可选）
  string comment = 4;
  
  // 步骤索引（必填）
  int32 step_index = 5;
}
```

**响应参数**:
```proto
message ApproveRequestResponse {
  // 更新后的审批请求
  ApprovalRequest approval_request = 1;
}
```

**HTTP示例**:
```bash
curl -X POST http://localhost:8081/v1/approval-requests/ar_1234567890/approve \
  -H "Content-Type: application/json" \
  -d '{
    "approver_id": "manager_456",
    "approver_name": "李四",
    "comment": "同意变更",
    "step_index": 0
  }'
```

### 10. RejectRequest

**功能**: 拒绝审批请求

**gRPC接口**:
```proto
rpc RejectRequest(RejectRequestRequest) returns (RejectRequestResponse) {
  option (google.api.http) = {
    post: "/v1/approval-requests/{id}/reject"
    body: "*"
  };
}
```

**请求参数**:
```proto
message RejectRequestRequest {
  // 审批请求ID（必填）
  string id = 1;
  
  // 审批人ID（必填）
  string approver_id = 2;
  
  // 审批人姓名（必填）
  string approver_name = 3;
  
  // 拒绝理由（必填）
  string comment = 4;
  
  // 步骤索引（必填）
  int32 step_index = 5;
}
```

**响应参数**:
```proto
message RejectRequestResponse {
  // 更新后的审批请求
  ApprovalRequest approval_request = 1;
}
```

**HTTP示例**:
```bash
curl -X POST http://localhost:8081/v1/approval-requests/ar_1234567890/reject \
  -H "Content-Type: application/json" \
  -d '{
    "approver_id": "manager_456",
    "approver_name": "李四",
    "comment": "变更理由不充分",
    "step_index": 0
  }'
```

## 错误码说明

### 通用错误码

| 错误码 | 描述 | HTTP状态码 |
|--------|------|------------|
| 2 | 无效参数 | 400 |
| 3 | 权限不足 | 403 |
| 5 | 资源不存在 | 404 |
| 6 | 资源已存在 | 409 |
| 13 | 内部错误 | 500 |
| 14 | 不可用 | 503 |

### 业务错误码

| 错误码 | 描述 | HTTP状态码 |
|--------|------|------------|
| 1001 | 敏感数据分级名称不能为空 | 400 |
| 1002 | 敏感级别必须在1-5之间 | 400 |
| 1003 | 敏感数据分级已存在 | 409 |
| 1004 | 审批流名称不能为空 | 400 |
| 1005 | 审批步骤不能为空 | 400 |
| 1006 | 审批请求已存在 | 409 |
| 1007 | 审批请求状态不允许此操作 | 400 |
| 1008 | 审批人无权限审批此请求 | 403 |
| 1009 | 审批步骤已完成 | 400 |
| 1010 | 审批请求已超时 | 400 |

## 认证和授权

### 认证方式
- **API Key**: 在请求头中添加 `X-API-Key`
- **JWT Token**: 在请求头中添加 `Authorization: Bearer <token>`
- **OAuth 2.0**: 支持OAuth 2.0认证

### 授权策略
- **RBAC**: 基于角色的访问控制
- **ABAC**: 基于属性的访问控制
- **ACL**: 访问控制列表

## 最佳实践

### 1. 批量操作

```bash
# 批量创建敏感数据分级
curl -X POST http://localhost:8080/v1/sensitive-levels/batch \
  -H "Content-Type: application/json" \
  -d '{
    "sensitive_levels": [
      {
        "name": "分级1",
        "level": 1
      },
      {
        "name": "分级2",
        "level": 2
      }
    ]
  }'
```

### 2. 异步操作

```bash
# 异步提交审批请求
curl -X POST http://localhost:8081/v1/approval-requests/async \
  -H "Content-Type: application/json" \
  -d '{
    "approval_flow_id": "af_1234567890",
    "requester_id": "user_123",
    "requester_name": "张三",
    "change_type": "UPDATE",
    "change_details": {
      "table": "users",
      "field": "身份证号",
      "old_value": "310101199001011234",
      "new_value": "310101199001011235"
    },
    "reason": "用户身份证号变更"
  }'
```

### 3. 事件通知

```bash
# 订阅审批事件
curl -X POST http://localhost:8081/v1/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-webhook-url.com",
    "events": [
      "APPROVAL_REQUEST_CREATED",
      "APPROVAL_REQUEST_APPROVED",
      "APPROVAL_REQUEST_REJECTED"
    ]
  }'
```

## 监控和指标

### 1. 健康检查

```bash
# 敏感数据分级服务健康检查
curl http://localhost:8080/health

# 审批流服务健康检查
curl http://localhost:8081/health
```

### 2. 指标收集

```bash
# 敏感数据分级服务指标
curl http://localhost:8080/metrics

# 审批流服务指标
curl http://localhost:8081/metrics
```

### 3. 日志收集

```bash
# 查看服务日志
docker logs sensitive-level-service
docker logs approval-flow-service
```

---

**文档版本**: v1.0.0  
**最后更新**: 2024年1月1日  
**维护团队**: Bytebase技术团队