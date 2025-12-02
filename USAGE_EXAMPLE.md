# 敏感数据分级与审批流功能使用示例

## 概述

本文档提供了敏感数据分级与审批流功能的详细使用示例，包括：
- 敏感数据分级配置示例
- 审批流配置示例
- 完整的工作流程示例
- API调用示例
- 最佳实践

## 快速开始

### 1. 环境准备

```bash
# 启动数据库和缓存
docker-compose -f docker-compose.sensitive_data.yml up -d postgres redis

# 启动敏感数据分级服务
docker-compose -f docker-compose.sensitive_data.yml up -d sensitive-level-service

# 启动审批流服务
docker-compose -f docker-compose.sensitive_data.yml up -d approval-flow-service
```

### 2. 验证服务

```bash
# 检查敏感数据分级服务健康状态
curl http://localhost:8080/health

# 检查审批流服务健康状态
curl http://localhost:8081/health

# 检查gRPC服务
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50052 list
```

## 敏感数据分级使用示例

### 1. 创建敏感数据分级

```bash
# 使用HTTP API创建
curl -X POST http://localhost:8080/v1/sensitive-levels \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高敏感数据",
    "level": 3,
    "description": "包含个人身份信息、财务数据等高度敏感信息",
    "field_rules": [
      {
        "field_name": "身份证号",
        "pattern": "^[1-9]\\d{5}(18|19|20)\\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\\d{3}[0-9Xx]$",
        "action": "ENCRYPT"
      },
      {
        "field_name": "银行卡号",
        "pattern": "^\\d{16,19}$",
        "action": "MASK"
      }
    ]
  }'

# 使用gRPC API创建
grpcurl -plaintext -d '{
  "name": "中敏感数据",
  "level": 2,
  "description": "包含客户信息、业务数据等中等敏感信息",
  "field_rules": [
    {
      "field_name": "手机号",
      "pattern": "^1[3-9]\\d{9}$",
      "action": "MASK"
    }
  ]
}' localhost:50051 v1.SensitiveLevelService/CreateSensitiveLevel
```

### 2. 查询敏感数据分级

```bash
# 获取所有敏感数据分级
curl http://localhost:8080/v1/sensitive-levels

# 获取特定敏感数据分级
curl http://localhost:8080/v1/sensitive-levels/{id}

# 使用gRPC查询
grpcurl -plaintext -d '{
  "page_size": 10,
  "page_token": ""
}' localhost:50051 v1.SensitiveLevelService/ListSensitiveLevels
```

### 3. 更新敏感数据分级

```bash
# 更新敏感数据分级
curl -X PUT http://localhost:8080/v1/sensitive-levels/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高敏感数据(更新)",
    "level": 3,
    "description": "更新后的描述"
  }'

# 使用gRPC更新
grpcurl -plaintext -d '{
  "id": "{id}",
  "name": "高敏感数据(更新)",
  "level": 3,
  "description": "更新后的描述"
}' localhost:50051 v1.SensitiveLevelService/UpdateSensitiveLevel
```

### 4. 删除敏感数据分级

```bash
# 删除敏感数据分级
curl -X DELETE http://localhost:8080/v1/sensitive-levels/{id}

# 使用gRPC删除
grpcurl -plaintext -d '{
  "id": "{id}"
}' localhost:50051 v1.SensitiveLevelService/DeleteSensitiveLevel
```

## 审批流使用示例

### 1. 创建审批流

```bash
# 创建高敏感数据审批流
curl -X POST http://localhost:8081/v1/approval-flows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高敏感数据变更审批",
    "description": "高敏感数据变更需要2级审批",
    "sensitive_level_id": "{sensitive_level_id}",
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

# 创建中敏感数据审批流
curl -X POST http://localhost:8081/v1/approval-flows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "中敏感数据变更审批",
    "description": "中敏感数据变更需要1级审批",
    "sensitive_level_id": "{sensitive_level_id}",
    "steps": [
      {
        "name": "直接主管审批",
        "type": "DIRECT_MANAGER",
        "required_approvers": 1,
        "timeout": 43200
      }
    ],
    "auto_approve": false
  }'

# 创建低敏感数据审批流（自动审批）
curl -X POST http://localhost:8081/v1/approval-flows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "低敏感数据变更审批",
    "description": "低敏感数据变更自动审批",
    "sensitive_level_id": "{sensitive_level_id}",
    "steps": [],
    "auto_approve": true
  }'
```

### 2. 提交审批请求

```bash
# 提交高敏感数据变更审批
curl -X POST http://localhost:8081/v1/approval-requests \
  -H "Content-Type: application/json" \
  -d '{
    "approval_flow_id": "{approval_flow_id}",
    "requester_id": "user123",
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

# 使用gRPC提交审批
grpcurl -plaintext -d '{
  "approval_flow_id": "{approval_flow_id}",
  "requester_id": "user123",
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
}' localhost:50052 v1.ApprovalFlowService/SubmitApproval
```

### 3. 审批请求

```bash
# 审批请求（通过）
curl -X POST http://localhost:8081/v1/approval-requests/{request_id}/approve \
  -H "Content-Type: application/json" \
  -d '{
    "approver_id": "manager456",
    "approver_name": "李四",
    "comment": "同意变更",
    "step_index": 0
  }'

# 审批请求（拒绝）
curl -X POST http://localhost:8081/v1/approval-requests/{request_id}/reject \
  -H "Content-Type: application/json" \
  -d '{
    "approver_id": "manager456",
    "approver_name": "李四",
    "comment": "变更理由不充分",
    "step_index": 0
  }'

# 使用gRPC审批
grpcurl -plaintext -d '{
  "request_id": "{request_id}",
  "approver_id": "manager456",
  "approver_name": "李四",
  "comment": "同意变更",
  "step_index": 0
}' localhost:50052 v1.ApprovalFlowService/ApproveRequest
```

### 4. 查询审批请求

```bash
# 获取所有审批请求
curl http://localhost:8081/v1/approval-requests

# 获取特定审批请求
curl http://localhost:8081/v1/approval-requests/{request_id}

# 按状态查询审批请求
curl http://localhost:8081/v1/approval-requests?status=PENDING

# 使用gRPC查询
grpcurl -plaintext -d '{
  "page_size": 10,
  "page_token": "",
  "status": "PENDING"
}' localhost:50052 v1.ApprovalFlowService/ListApprovalRequests
```

## 完整工作流程示例

### 场景：用户数据变更审批

```bash
# 1. 创建敏感数据分级
curl -X POST http://localhost:8080/v1/sensitive-levels \
  -H "Content-Type: application/json" \
  -d '{
    "name": "用户敏感数据",
    "level": 3,
    "description": "包含用户个人身份信息的敏感数据",
    "field_rules": [
      {
        "field_name": "身份证号",
        "pattern": "^[1-9]\\d{5}(18|19|20)\\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\\d{3}[0-9Xx]$",
        "action": "ENCRYPT"
      }
    ]
  }'

# 2. 创建审批流
curl -X POST http://localhost:8081/v1/approval-flows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "用户数据变更审批",
    "description": "用户敏感数据变更需要2级审批",
    "sensitive_level_id": "{sensitive_level_id}",
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

# 3. 提交审批请求
curl -X POST http://localhost:8081/v1/approval-requests \
  -H "Content-Type: application/json" \
  -d '{
    "approval_flow_id": "{approval_flow_id}",
    "requester_id": "user789",
    "requester_name": "王五",
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

# 4. 部门经理审批（通过）
curl -X POST http://localhost:8081/v1/approval-requests/{request_id}/approve \
  -H "Content-Type: application/json" \
  -d '{
    "approver_id": "manager101",
    "approver_name": "赵六",
    "comment": "同意用户身份证号变更",
    "step_index": 0
  }'

# 5. 数据安全官审批（通过）
curl -X POST http://localhost:8081/v1/approval-requests/{request_id}/approve \
  -H "Content-Type: application/json" \
  -d '{
    "approver_id": "dso102",
    "approver_name": "孙七",
    "comment": "已审核，同意变更",
    "step_index": 1
  }'

# 6. 验证审批结果
curl http://localhost:8081/v1/approval-requests/{request_id}
```

## 错误处理示例

### 1. 无效的敏感数据分级

```bash
# 尝试创建无效的敏感数据分级
curl -X POST http://localhost:8080/v1/sensitive-levels \
  -H "Content-Type: application/json" \
  -d '{
    "name": "",
    "level": 0,
    "description": "无效的敏感数据分级"
  }'

# 预期响应
{
  "error": {
    "code": 400,
    "message": "Invalid sensitive level: name cannot be empty, level must be between 1 and 5"
  }
}
```

### 2. 审批请求不存在

```bash
# 尝试审批不存在的请求
curl -X POST http://localhost:8081/v1/approval-requests/invalid-id/approve \
  -H "Content-Type: application/json" \
  -d '{
    "approver_id": "manager101",
    "approver_name": "赵六",
    "comment": "同意",
    "step_index": 0
  }'

# 预期响应
{
  "error": {
    "code": 404,
    "message": "Approval request not found"
  }
}
```

## 监控和日志示例

### 1. 查看服务指标

```bash
# 查看敏感数据分级服务指标
curl http://localhost:8080/metrics

# 查看审批流服务指标
curl http://localhost:8081/metrics
```

### 2. 查看服务日志

```bash
# 查看敏感数据分级服务日志
docker logs sensitive-level-service

# 查看审批流服务日志
docker logs approval-flow-service

# 查看最近的错误日志
docker logs sensitive-level-service | grep ERROR
```

## 最佳实践

### 1. 敏感数据分级策略

```bash
# 推荐的敏感数据分级配置
# 1级：公开数据（无需保护）
# 2级：内部数据（需要基本保护）
# 3级：敏感数据（需要严格保护）
# 4级：机密数据（需要高度保护）
# 5级：绝密数据（需要最高级保护）

# 创建分级配置
for level in 1 2 3 4 5; do
  curl -X POST http://localhost:8080/v1/sensitive-levels \
    -H "Content-Type: application/json" \
    -d "{\n      \"name\": \"分级${level}\",\n      \"level\": ${level},\n      \"description\": \"分级${level}数据\"\n    }"
done
```

### 2. 审批流最佳实践

```bash
# 推荐的审批流配置
# 高敏感数据：2级审批（部门经理 + 数据安全官）
# 中敏感数据：1级审批（直接主管）
# 低敏感数据：自动审批

# 创建审批流配置
curl -X POST http://localhost:8081/v1/approval-flows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高敏感数据审批",
    "sensitive_level_id": "high_level_id",
    "steps": [
      {
        "name": "部门经理审批",
        "type": "MANAGER",
        "required_approvers": 1
      },
      {
        "name": "数据安全官审批",
        "type": "DATA_SECURITY_OFFICER",
        "required_approvers": 1
      }
    ]
  }'
```

### 3. 监控和告警

```bash
# 配置Prometheus告警规则
# 示例：当审批请求超时超过10个时触发告警

# 创建告警规则文件
cat > alerting_rules.yml << 'EOF'
groups:
- name: sensitive_data_alerts
  rules:
  - alert: ApprovalRequestTimeout
    expr: sum by (approval_flow_id) (approval_requests_timeout_total) > 10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Approval request timeout detected"
      description: "More than 10 approval requests have timed out"
EOF
```

## 故障排除

### 1. 服务启动失败

```bash
# 检查服务状态
docker-compose -f docker-compose.sensitive_data.yml ps

# 查看服务日志
docker logs sensitive-level-service

# 检查端口占用
netstat -tlnp | grep 50051
```

### 2. 数据库连接失败

```bash
# 检查数据库连接
psql -h localhost -p 5432 -U sensitive_data_user -d sensitive_data

# 检查数据库服务状态
sudo systemctl status postgresql
```

### 3. Redis连接失败

```bash
# 检查Redis连接
redis-cli ping

# 检查Redis服务状态
sudo systemctl status redis
```

---

**文档版本**: v1.0.0  
**最后更新**: 2024年1月1日  
**维护团队**: Bytebase技术团队