# 敏感数据分级与审批流功能项目结构

## 概述

本文档详细描述了敏感数据分级与审批流功能的项目结构和组织方式，包括：
- 项目目录结构
- 各模块职责
- 代码组织原则
- 构建和部署流程

## 项目目录结构

```
bytebase/
├── backend/                      # 后端代码
│   ├── api/                      # API接口定义
│   │   └── v1/                   # v1版本API
│   │       ├── sensitive_level_service.go          # 敏感数据分级服务实现
│   │       ├── approval_flow_service.go            # 审批流服务实现
│   │       ├── sensitive_level_service.proto       # Protobuf接口定义
│   │       ├── approval_flow_service.proto         # Protobuf接口定义
│   │       ├── sensitive_level_service_test.go     # 单元测试
│   │       ├── approval_flow_service_test.go       # 单元测试
│   │       └── sensitive_level_approval_flow_integration_test.go  # 集成测试
│   ├── generated-go/             # 生成的Go代码
│   │   └── v1/                   # v1版本生成代码
│   ├── service/                  # 业务逻辑层
│   │   ├── sensitive_level/      # 敏感数据分级业务逻辑
│   │   └── approval_flow/        # 审批流业务逻辑
│   ├── repository/               # 数据访问层
│   │   ├── sensitive_level/      # 敏感数据分级数据访问
│   │   └── approval_flow/        # 审批流数据访问
│   ├── model/                    # 数据模型
│   │   ├── sensitive_level/      # 敏感数据分级模型
│   │   └── approval_flow/        # 审批流模型
│   └── util/                     # 工具函数
├── docs/                         # 文档
│   ├── sensitive_data_approval_flow.md           # 功能文档
│   ├── sensitive_data_approval_flow_api.md       # API文档
│   └── api/                      # API文档目录
├── examples/                     # 使用示例
│   └── sensitive_data_approval_flow_example.go   # 完整使用示例
├── configs/                      # 配置文件
│   ├── sensitive_level_config.yaml                # 敏感数据分级配置
│   └── approval_flow_config.yaml                  # 审批流配置
├── docker/                       # Docker相关配置
│   ├── postgres/                 # PostgreSQL配置
│   │   └── init.sql              # 数据库初始化脚本
│   ├── nginx/                    # Nginx配置
│   │   ├── nginx.conf            # Nginx主配置
│   │   ├── conf.d/               # 虚拟主机配置
│   │   └── certs/                # SSL证书
│   ├── prometheus/               # Prometheus配置
│   │   └── prometheus.yml        # Prometheus配置
│   └── grafana/                  # Grafana配置
│       ├── dashboards/           # 仪表板配置
│       └── provisioning/         # 数据源配置
├── kubernetes/                   # Kubernetes配置
│   ├── sensitive-data-deployment.yaml            # 部署配置
│   ├── sensitive-data-service.yaml               # 服务配置
│   └── sensitive-data-ingress.yaml               # Ingress配置
├── tests/                        # 测试相关
│   ├── integration/              # 集成测试
│   └── performance/              # 性能测试
├── build_sensitive_data.sh       # 构建脚本（Bash）
├── build_sensitive_data.ps1      # 构建脚本（PowerShell）
├── docker-compose.sensitive_data.yml  # Docker Compose配置
├── Dockerfile.sensitive_level    # 敏感数据分级服务Dockerfile
├── Dockerfile.approval_flow      # 审批流服务Dockerfile
├── go.mod.sensitive_data         # Go模块配置
├── Makefile.sensitive_data       # Makefile构建配置
├── .gitignore.sensitive_data     # Git忽略配置
├── CHANGELOG_SENSITIVE_DATA.md   # 变更日志
├── CONTRIBUTING_SENSITIVE_DATA.md # 贡献指南
├── README_SENSITIVE_DATA.md      # 功能说明文档
├── SECURITY_SENSITIVE_DATA.md    # 安全指南
├── LICENSE_SENSITIVE_DATA.md     # 许可证信息
└── DEPLOYMENT_GUIDE.md           # 部署指南
```

## 各模块职责

### 后端代码结构

#### API层 (backend/api/v1/)
- **职责**: 定义和实现对外API接口
- **文件**: 
  - `*.proto`: Protobuf接口定义
  - `*.go`: API服务实现
  - `*_test.go`: 单元测试

#### 业务逻辑层 (backend/service/)
- **职责**: 实现核心业务逻辑
- **结构**:
  - `sensitive_level/`: 敏感数据分级业务逻辑
  - `approval_flow/`: 审批流业务逻辑

#### 数据访问层 (backend/repository/)
- **职责**: 实现数据持久化和访问
- **结构**:
  - `sensitive_level/`: 敏感数据分级数据访问
  - `approval_flow/`: 审批流数据访问

#### 数据模型层 (backend/model/)
- **职责**: 定义数据模型和实体
- **结构**:
  - `sensitive_level/`: 敏感数据分级模型
  - `approval_flow/`: 审批流模型

#### 工具层 (backend/util/)
- **职责**: 提供通用工具函数
- **内容**: 日志、错误处理、配置管理等

### 文档结构

#### 功能文档 (docs/)
- **职责**: 说明功能使用方法和API接口
- **文件**:
  - `sensitive_data_approval_flow.md`: 功能说明文档
  - `sensitive_data_approval_flow_api.md`: API文档

#### 示例代码 (examples/)
- **职责**: 提供完整的使用示例
- **文件**:
  - `sensitive_data_approval_flow_example.go`: 完整使用示例

### 配置文件结构

#### 应用配置 (configs/)
- **职责**: 管理应用配置
- **文件**:
  - `sensitive_level_config.yaml`: 敏感数据分级配置
  - `approval_flow_config.yaml`: 审批流配置

#### 部署配置 (docker/, kubernetes/)
- **职责**: 管理部署相关配置
- **结构**:
  - `docker/`: Docker相关配置
  - `kubernetes/`: Kubernetes相关配置

## 代码组织原则

### 1. 单一职责原则
- 每个文件和模块只负责一个功能
- API层专注于接口定义和请求处理
- 业务逻辑层专注于业务规则实现
- 数据访问层专注于数据持久化

### 2. 分层架构
- 清晰的层级划分
- 层间通过接口通信
- 降低模块间耦合

### 3. 依赖管理
- 使用Go Modules管理依赖
- 明确的依赖关系
- 最小化依赖数量

### 4. 测试驱动开发
- 每个功能都有对应的单元测试
- 集成测试验证系统整体功能
- 性能测试确保系统性能

### 5. 文档驱动开发
- 每个功能都有详细的文档
- API接口有完整的文档说明
- 部署和配置有详细指南

## 构建和部署流程

### 构建流程

```
1. 代码生成 (Protobuf)
   protoc --proto_path=backend/api/v1 \
          --go_out=backend/generated-go/v1 \
          --go-grpc_out=backend/generated-go/v1 \
          backend/api/v1/*.proto

2. 编译构建
   go build -o bin/sensitive_level_service backend/api/v1/sensitive_level_service.go
   go build -o bin/approval_flow_service backend/api/v1/approval_flow_service.go

3. 测试
   go test ./backend/api/v1/...
   go test ./backend/service/...
   go test ./backend/repository/...

4. 打包
   docker build -t sensitive-level-service -f Dockerfile.sensitive_level .
   docker build -t approval-flow-service -f Dockerfile.approval_flow .
```

### 部署流程

```
1. 环境准备
   - 安装依赖服务（PostgreSQL, Redis）
   - 配置数据库和缓存

2. 服务部署
   - 使用Docker Compose部署
   - 或使用Kubernetes部署

3. 配置管理
   - 配置环境变量
   - 配置文件管理

4. 监控和日志
   - 配置Prometheus监控
   - 配置Grafana可视化
   - 配置日志收集

5. 健康检查
   - 配置健康检查端点
   - 配置自动恢复机制
```

## 开发工作流

### 1. 功能开发

```
1. 创建Protobuf接口定义
2. 生成Go代码
3. 实现业务逻辑
4. 实现数据访问层
5. 编写单元测试
6. 编写集成测试
```

### 2. 代码审查

```
1. 提交代码到GitHub
2. 创建Pull Request
3. 代码审查
4. 自动化测试
5. 合并到主分支
```

### 3. 版本发布

```
1. 版本标记
2. 构建Docker镜像
3. 部署到测试环境
4. 集成测试
5. 部署到生产环境
```

## 代码规范

### Go代码规范

```go
// 包名使用小写
package sensitive_level

// 结构体使用驼峰命名
type SensitiveLevel struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Level     int       `json:"level"`
    CreatedAt time.Time `json:"created_at"`
}

// 函数名使用驼峰命名
func CreateSensitiveLevel(ctx context.Context, level *SensitiveLevel) (*SensitiveLevel, error) {
    // 实现逻辑
}

// 接口名以er结尾
type SensitiveLevelService interface {
    Create(ctx context.Context, level *SensitiveLevel) (*SensitiveLevel, error)
    Get(ctx context.Context, id string) (*SensitiveLevel, error)
    List(ctx context.Context, filter *Filter) ([]*SensitiveLevel, error)
    Update(ctx context.Context, level *SensitiveLevel) (*SensitiveLevel, error)
    Delete(ctx context.Context, id string) error
}
```

### Protobuf规范

```proto
// 包名使用小写
syntax = "proto3";

package v1;

// 导入必要的包
import "google/protobuf/timestamp.proto";
import "google/api/annotations.proto";

// 服务名使用驼峰命名
service SensitiveLevelService {
  // RPC方法名使用驼峰命名
  rpc CreateSensitiveLevel(CreateSensitiveLevelRequest) returns (CreateSensitiveLevelResponse) {
    option (google.api.http) = {
      post: "/v1/sensitive-levels"
      body: "*"
    };
  }
}

// 消息名使用驼峰命名
message SensitiveLevel {
  string id = 1;
  string name = 2;
  int32 level = 3;
  google.protobuf.Timestamp created_at = 4;
}
```

## 测试策略

### 单元测试

```go
func TestCreateSensitiveLevel(t *testing.T) {
    // 创建测试存储
    store := testutil.NewStore(t)
    
    // 创建服务
    service := NewSensitiveLevelService(store)
    
    // 测试创建
    level := &SensitiveLevel{
        Name: "高敏感",
        Level: 3,
    }
    
    created, err := service.Create(context.Background(), level)
    require.NoError(t, err)
    require.NotNil(t, created)
    require.Equal(t, "高敏感", created.Name)
}
```

### 集成测试

```go
func TestSensitiveLevelAndApprovalFlowIntegration(t *testing.T) {
    // 创建测试存储
    store := testutil.NewStore(t)
    
    // 创建服务
    sensitiveLevelService := NewSensitiveLevelService(store)
    approvalFlowService := NewApprovalFlowService(store, sensitiveLevelService)
    
    // 测试完整流程
    // 1. 创建敏感数据分级
    // 2. 创建审批流
    // 3. 提交审批请求
    // 4. 审批请求
    // 5. 验证结果
}
```

## 性能优化

### 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_sensitive_level_name ON sensitive_levels(name);
CREATE INDEX idx_approval_flow_status ON approval_flows(status);
CREATE INDEX idx_approval_request_created_at ON approval_requests(created_at);

-- 配置连接池
ALTER SYSTEM SET max_connections = 1000;
ALTER SYSTEM SET shared_buffers = '4GB';
```

### 缓存优化

```go
// 使用Redis缓存
func (s *SensitiveLevelService) Get(ctx context.Context, id string) (*SensitiveLevel, error) {
    // 先从缓存获取
    cached, err := s.redis.Get(ctx, fmt.Sprintf("sensitive_level:%s", id)).Result()
    if err == nil {
        var level SensitiveLevel
        if err := json.Unmarshal([]byte(cached), &level); err == nil {
            return &level, nil
        }
    }
    
    // 从数据库获取
    level, err := s.repository.Get(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 存入缓存
    data, _ := json.Marshal(level)
    s.redis.Set(ctx, fmt.Sprintf("sensitive_level:%s", id), data, 10*time.Minute)
    
    return level, nil
}
```

## 安全考虑

### 认证和授权

```go
// 认证中间件
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // 验证token
        if !validateToken(token) {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### 数据加密

```go
// 敏感数据加密
func encryptData(data []byte) ([]byte, error) {
    key := []byte("encryption-key-32-bytes")
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    return gcm.Seal(nonce, nonce, data, nil), nil
}
```

## 监控和日志

### 指标收集

```go
// Prometheus指标
var (
    requestCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "sensitive_level_requests_total",
            Help: "Total number of sensitive level requests",
        },
        []string{"method", "status"},
    )
    
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "sensitive_level_request_duration_seconds",
            Help: "Duration of sensitive level requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method"},
    )
)
```

### 日志记录

```go
// 结构化日志
logger := zerolog.New(os.Stdout).With().
    Timestamp().
    Str("service", "sensitive-level").
    Logger()

logger.Info().
    Str("method", "Create").
    Str("name", level.Name).
    Int("level", level.Level).
    Msg("Sensitive level created")
```

---

**文档版本**: v1.0.0  
**最后更新**: 2024年1月1日  
**维护团队**: Bytebase技术团队