# 敏感数据分级与审批流功能 - 贡献指南

欢迎为 Bytebase 的敏感数据分级与审批流功能贡献代码！本指南将帮助您了解如何参与开发和贡献。

## 目录

1. [开始之前](#开始之前)
2. [开发环境设置](#开发环境设置)
3. [代码结构](#代码结构)
4. [开发流程](#开发流程)
5. [编码规范](#编码规范)
6. [测试指南](#测试指南)
7. [提交代码](#提交代码)
8. [文档](#文档)
9. [常见问题](#常见问题)

## 开始之前

### 1.1 了解项目

敏感数据分级与审批流功能是 Bytebase 的核心安全功能之一，主要包括：

- 敏感数据分级配置
- 变更审批流管理
- 审批流程可视化
- 变更拦截逻辑
- API 接口

### 1.2 技术栈

- **语言**：Go 1.18+
- **框架**：gRPC, Protobuf
- **数据库**：MySQL, PostgreSQL
- **测试**：Go 测试框架
- **构建**：Makefile

### 1.3 贡献方式

- 修复 bug
- 实现新功能
- 改进文档
- 优化性能
- 增加测试覆盖率

## 开发环境设置

### 2.1 安装依赖

```bash
# 安装 Go 1.18+
# 参考：https://golang.org/doc/install

# 安装 Protobuf 编译器
brew install protobuf  # macOS
sudo apt-get install protobuf-compiler  # Ubuntu/Debian

# 安装 Go 的 Protobuf 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 安装测试工具
go install github.com/stretchr/testify/assert@latest
go install github.com/stretchr/testify/require@latest
```

### 2.2 克隆仓库

```bash
git clone https://github.com/bytebase/bytebase.git
cd bytebase
```

### 2.3 设置数据库

```bash
# 创建测试数据库
mysql -u root -p -e "CREATE DATABASE bytebase_test;"

# 设置环境变量
export BYTEBASE_DB_HOST=localhost
export BYTEBASE_DB_PORT=3306
export BYTEBASE_DB_USER=root
export BYTEBASE_DB_PASSWORD=your_password
export BYTEBASE_DB_NAME=bytebase_test
```

### 2.4 构建项目

```bash
# 构建敏感数据分级与审批流功能
make -f Makefile.sensitive_data build

# 生成 protobuf 代码
make -f Makefile.sensitive_data proto
```

## 代码结构

### 3.1 项目结构

```
bytebase/
├── backend/
│   ├── api/
│   │   └── v1/
│   │       ├── sensitive_level_service.go      # 敏感数据分级服务实现
│   │       ├── approval_flow_service.go         # 审批流服务实现
│   │       ├── sensitive_level_service_test.go  # 敏感数据分级服务测试
│   │       ├── approval_flow_service_test.go    # 审批流服务测试
│   │       └── sensitive_level_approval_flow_integration_test.go  # 集成测试
│   └── store/
│       ├── sensitive_level.go                   # 敏感数据分级存储实现
│       ├── approval_flow.go                     # 审批流存储实现
│       └── schema/
│           └── 20240101000000_add_sensitive_level_and_approval_flow_tables.sql  # 数据库迁移
├── proto/
│   └── v1/
│       └── v1/
│           ├── sensitive_level_service.proto    # 敏感数据分级服务 Protobuf 定义
│           └── approval_flow_service.proto      # 审批流服务 Protobuf 定义
├── docs/
│   └── sensitive_data_approval_flow.md          # 功能文档
├── examples/
│   └── sensitive_data_approval_flow_example.go  # 使用示例
├── Makefile.sensitive_data                      # 构建和测试脚本
└── .gitignore.sensitive_data                    # Git 忽略配置
```

### 3.2 主要模块

1. **Protobuf 定义**：定义 API 接口和数据结构
2. **服务实现**：实现业务逻辑和 API 接口
3. **存储层**：实现数据库操作
4. **测试**：单元测试和集成测试
5. **文档**：功能文档和使用示例

## 开发流程

### 4.1 创建分支

```bash
# 从 main 分支创建新分支
git checkout -b feature/your-feature-name
```

### 4.2 实现功能

1. **更新 Protobuf 定义**（如果需要）
2. **实现服务逻辑**
3. **实现存储层逻辑**
4. **编写测试用例**
5. **更新文档**

### 4.3 构建和测试

```bash
# 构建项目
make -f Makefile.sensitive_data build

# 运行测试
make -f Makefile.sensitive_data test

# 运行测试并生成覆盖率报告
make -f Makefile.sensitive_data test-coverage
```

### 4.4 提交代码

```bash
# 添加修改的文件
git add .

# 提交代码
# 提交信息格式：<类型>(<模块>): <描述>
# 类型：feat, fix, docs, style, refactor, test, chore
# 示例：feat(sensitive-level): 添加新的敏感数据分级规则
git commit -m "feat(sensitive-level): 添加新的敏感数据分级规则"
```

### 4.5 创建 Pull Request

1. 推送分支到远程仓库
2. 在 GitHub 上创建 Pull Request
3. 等待代码审查
4. 根据反馈进行修改
5. 合并到 main 分支

## 编码规范

### 5.1 Go 编码规范

遵循 Go 官方编码规范：

- 使用 `gofmt` 格式化代码
- 使用 `go vet` 检查代码
- 使用 `golint` 检查代码风格
- 保持函数简洁，每个函数不超过 50 行
- 保持代码注释清晰

### 5.2 命名规范

- **包名**：小写，使用短名称
- **类型名**：大写开头，驼峰式
- **函数名**：大写开头，驼峰式
- **变量名**：小写开头，驼峰式
- **常量名**：全大写，下划线分隔

### 5.3 错误处理

- 始终检查和处理错误
- 使用有意义的错误信息
- 避免忽略错误
- 对于预期的错误，使用 `errors.New()` 或 `fmt.Errorf()`

### 5.4 测试规范

- 每个函数都应该有对应的测试用例
- 测试用例应该覆盖正常情况和边界情况
- 测试用例应该独立，不依赖其他测试用例
- 使用 `testify` 库进行断言

## 测试指南

### 6.1 单元测试

```go
// 示例单元测试
func TestCreateSensitiveLevel(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
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
    require.NoError(t, err)
    
    // 创建敏感数据分级服务
    service := NewSensitiveLevelService(s)
    
    // 测试创建敏感数据分级
    req := &v1.CreateSensitiveLevelRequest{
        DisplayName: "Test Sensitive Level",
        Description: "Test description",
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
    
    resp, err := service.CreateSensitiveLevel(ctx, req)
    require.NoError(t, err)
    require.NotNil(t, resp)
    
    assert.Equal(t, "Test Sensitive Level", resp.SensitiveLevel.DisplayName)
    assert.Equal(t, v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH, resp.SensitiveLevel.Level)
}
```

### 6.2 集成测试

```go
// 示例集成测试
func TestSensitiveLevelAndApprovalFlowIntegration(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // 创建服务
    sensitiveLevelService := NewSensitiveLevelService(s)
    approvalFlowService := NewApprovalFlowService(s)
    
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
    require.NoError(t, err)
    
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
    require.NoError(t, err)
    
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
    require.NoError(t, err)
    
    // 提交审批请求
    submitReq := &v1.SubmitApprovalRequest{
        Title:             "更新用户密码",
        Description:       "更新 users 表中的密码字段",
        IssueId:           "issue-12345",
        SensitivityLevel:  v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        Submitter:         "developer@example.com",
    }
    
    submitResp, err := approvalFlowService.SubmitApproval(ctx, submitReq)
    require.NoError(t, err)
    
    // 批准请求
    approveReq := &v1.ApproveRequestRequest{
        ApprovalRequestId: submitResp.ApprovalRequest.Name,
        Comment:           "批准此变更请求",
        Approver:          "department-manager@example.com",
    }
    
    approveResp, err := approvalFlowService.ApproveRequest(ctx, approveReq)
    require.NoError(t, err)
    
    // 验证结果
    assert.Equal(t, v1.ApprovalStatus_APPROVAL_STATUS_APPROVED, approveResp.ApprovalRequest.Status)
}
```

## 提交代码

### 7.1 提交信息规范

提交信息应该清晰描述所做的更改，格式如下：

```
<类型>(<模块>): <简短描述>

<详细描述>

<相关 Issue>
```

**类型**：
- `feat`：新功能
- `fix`：修复 bug
- `docs`：文档更新
- `style`：代码风格调整
- `refactor`：代码重构
- `test`：测试相关
- `chore`：构建过程或辅助工具的变动

**示例**：

```
feat(sensitive-level): 添加新的敏感数据分级规则

- 支持按数据类型匹配敏感字段
- 支持正则表达式匹配
- 增加单元测试覆盖

Closes #123
```

### 7.2 Pull Request 规范

1. **标题**：清晰描述所做的更改
2. **描述**：详细说明更改的内容和原因
3. **相关 Issue**：引用相关的 Issue
4. **测试**：确保所有测试通过
5. **覆盖率**：确保代码覆盖率达到要求

## 文档

### 8.1 文档类型

1. **功能文档**：`docs/sensitive_data_approval_flow.md`
2. **API 文档**：通过 Protobuf 定义自动生成
3. **使用示例**：`examples/sensitive_data_approval_flow_example.go`
4. **贡献指南**：本文件

### 8.2 文档规范

- 使用 Markdown 格式
- 保持文档清晰简洁
- 提供示例代码
- 定期更新文档

## 常见问题

### 9.1 构建失败

**问题**：构建时出现错误

**解决方法**：

1. 检查 Go 版本是否为 1.18+
2. 检查 Protobuf 编译器是否安装
3. 检查 Go 的 Protobuf 插件是否安装
4. 检查依赖是否正确安装

### 9.2 测试失败

**问题**：测试时出现错误

**解决方法**：

1. 检查数据库连接是否正确
2. 检查测试数据是否正确
3. 检查测试用例是否有问题
4. 查看详细的错误信息

### 9.3 代码审查不通过

**问题**：Pull Request 被拒绝

**解决方法**：

1. 仔细阅读审查意见
2. 根据意见进行修改
3. 回复审查意见
4. 重新提交

### 9.4 其他问题

如果遇到其他问题，可以：

1. 查看项目的 Issues 页面
2. 在 GitHub Discussions 中提问
3. 加入项目的 Slack 或 Discord 社区
4. 发送邮件给维护者

## 联系方式

- **GitHub Issues**：https://github.com/bytebase/bytebase/issues
- **GitHub Discussions**：https://github.com/bytebase/bytebase/discussions
- **Slack 社区**：https://join.slack.com/t/bytebase/shared_invite/zt-16kf8xq9e-8x~W4bZ7g~0e0aZ7g~0e0a
- **邮件**：support@bytebase.com

## 许可证

本项目采用 Apache License 2.0 许可证，贡献的代码将遵循相同的许可证。

---

感谢您的贡献！您的努力将帮助 Bytebase 变得更好。