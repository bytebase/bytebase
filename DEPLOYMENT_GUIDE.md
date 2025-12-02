# 敏感数据分级与审批流功能部署指南

## 概述

本文档提供了敏感数据分级与审批流功能的详细部署指南，包括：
- 环境要求
- 部署方式（本地部署、Docker部署、Kubernetes部署）
- 配置说明
- 监控和日志
- 故障排除

## 环境要求

### 系统要求
- **操作系统**: Linux, macOS, Windows
- **Go版本**: 1.21+ (用于构建)
- **Protobuf编译器**: 3.20+ (用于代码生成)
- **数据库**: PostgreSQL 12+ (用于存储数据)
- **缓存**: Redis 6+ (用于缓存和消息队列)

### 资源要求
- **CPU**: 2核+ (建议4核)
- **内存**: 4GB+ (建议8GB)
- **存储**: 20GB+ (建议50GB)

## 部署方式

### 1. 本地部署

#### 1.1 依赖安装

```bash
# 安装Go (如果未安装)
go version

# 安装Protobuf编译器
protoc --version

# 安装Protobuf Go插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 安装PostgreSQL
psql --version

# 安装Redis
redis-cli --version
```

#### 1.2 构建和运行

```bash
# 克隆代码仓库
git clone <repository-url>
cd bytebase

# 生成Protobuf代码
./build_sensitive_data.sh --generate

# 构建服务
./build_sensitive_data.sh --build

# 启动数据库和缓存
# PostgreSQL
createdb sensitive_data
psql -d sensitive_data -f docker/postgres/init.sql

# Redis
redis-server

# 启动敏感数据分级服务
./bin/sensitive_level_service

# 启动审批流服务
./bin/approval_flow_service
```

### 2. Docker部署

#### 2.1 使用Docker Compose

```bash
# 构建并启动所有服务
docker-compose -f docker-compose.sensitive_data.yml up -d

# 查看服务状态
docker-compose -f docker-compose.sensitive_data.yml ps

# 查看日志
docker-compose -f docker-compose.sensitive_data.yml logs -f

# 停止服务
docker-compose -f docker-compose.sensitive_data.yml down
```

#### 2.2 单独构建和运行

```bash
# 构建敏感数据分级服务镜像
docker build -t sensitive-level-service -f Dockerfile.sensitive_level .

# 构建审批流服务镜像
docker build -t approval-flow-service -f Dockerfile.approval_flow .

# 启动数据库
docker run -d \
  --name postgres \
  -e POSTGRES_DB=sensitive_data \
  -e POSTGRES_USER=sensitive_data_user \
  -e POSTGRES_PASSWORD=sensitive_data_password \
  -p 5432:5432 \
  postgres:15-alpine

# 启动Redis
docker run -d \
  --name redis \
  -p 6379:6379 \
  redis:7-alpine

# 启动敏感数据分级服务
docker run -d \
  --name sensitive-level-service \
  --link postgres:postgres \
  --link redis:redis \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_NAME=sensitive_data \
  -e DB_USER=sensitive_data_user \
  -e DB_PASSWORD=sensitive_data_password \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -p 50051:50051 \
  -p 8080:8080 \
  sensitive-level-service

# 启动审批流服务
docker run -d \
  --name approval-flow-service \
  --link postgres:postgres \
  --link redis:redis \
  --link sensitive-level-service:sensitive-level-service \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_NAME=sensitive_data \
  -e DB_USER=sensitive_data_user \
  -e DB_PASSWORD=sensitive_data_password \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -e SENSITIVE_LEVEL_SERVICE_ADDR=sensitive-level-service:50051 \
  -p 50052:50052 \
  -p 8081:8081 \
  approval-flow-service
```

### 3. Kubernetes部署

#### 3.1 部署配置

```yaml
# kubernetes/sensitive-data-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sensitive-level-service
spec:
  replicas: 2
  selector:
    matchLabels:
      app: sensitive-level-service
  template:
    metadata:
      labels:
        app: sensitive-level-service
    spec:
      containers:
      - name: sensitive-level-service
        image: sensitive-level-service:latest
        ports:
        - containerPort: 50051
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "postgres"
        - name: DB_PORT
          value: "5432"
        - name: DB_NAME
          value: "sensitive_data"
        - name: DB_USER
          value: "sensitive_data_user"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: sensitive-data-secrets
              key: db-password
        - name: REDIS_HOST
          value: "redis"
        - name: REDIS_PORT
          value: "6379"
```

#### 3.2 部署命令

```bash
# 创建命名空间
kubectl create namespace sensitive-data

# 创建Secret
kubectl create secret generic sensitive-data-secrets \
  --namespace sensitive-data \
  --from-literal=db-password=sensitive_data_password

# 部署服务
kubectl apply -f kubernetes/

# 查看部署状态
kubectl get pods -n sensitive-data
kubectl get services -n sensitive-data
```

## 配置说明

### 环境变量配置

| 环境变量 | 描述 | 默认值 |
|---------|------|--------|
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_NAME` | 数据库名称 | `sensitive_data` |
| `DB_USER` | 数据库用户名 | `sensitive_data_user` |
| `DB_PASSWORD` | 数据库密码 | - |
| `REDIS_HOST` | Redis主机 | `localhost` |
| `REDIS_PORT` | Redis端口 | `6379` |
| `GRPC_PORT` | gRPC服务端口 | `50051` |
| `HTTP_PORT` | HTTP服务端口 | `8080` |
| `LOG_LEVEL` | 日志级别 | `info` |
| `SENSITIVE_LEVEL_SERVICE_ADDR` | 敏感数据分级服务地址 | - |

### 配置文件

配置文件位于 `configs/` 目录下：

```yaml
# configs/sensitive_level_config.yaml
database:
  host: localhost
  port: 5432
  name: sensitive_data
  user: sensitive_data_user
  password: sensitive_data_password

redis:
  host: localhost
  port: 6379

service:
  grpc_port: 50051
  http_port: 8080

logging:
  level: info
  format: json
```

## 监控和日志

### 健康检查

每个服务都提供健康检查端点：

```bash
# 敏感数据分级服务
curl http://localhost:8080/health

# 审批流服务
curl http://localhost:8081/health
```

### 指标监控

服务暴露Prometheus指标：

```bash
# 敏感数据分级服务指标
curl http://localhost:8080/metrics

# 审批流服务指标
curl http://localhost:8081/metrics
```

### 日志配置

日志支持以下格式：
- JSON格式（默认）
- 文本格式
- 结构化日志

## 故障排除

### 常见问题

#### 1. 数据库连接失败

```bash
# 检查数据库连接
psql -h localhost -p 5432 -U sensitive_data_user -d sensitive_data

# 检查数据库服务状态
sudo systemctl status postgresql
```

#### 2. Redis连接失败

```bash
# 检查Redis连接
redis-cli ping

# 检查Redis服务状态
sudo systemctl status redis
```

#### 3. 服务启动失败

```bash
# 查看服务日志
docker logs sensitive-level-service

# 检查服务端口占用
netstat -tlnp | grep 50051
```

#### 4. gRPC服务不可达

```bash
# 检查gRPC连接
grpcurl -plaintext localhost:50051 list

# 检查防火墙设置
sudo ufw status
```

### 日志分析

```bash
# 查看最近的错误日志
docker logs sensitive-level-service | grep ERROR

# 查看特定时间段的日志
docker logs --since 2024-01-01 sensitive-level-service
```

## 性能优化

### 数据库优化

```sql
# 创建索引
CREATE INDEX idx_sensitive_level_name ON sensitive_levels(name);
CREATE INDEX idx_approval_flow_status ON approval_flows(status);
CREATE INDEX idx_approval_request_created_at ON approval_requests(created_at);

# 配置连接池
ALTER SYSTEM SET max_connections = 1000;
ALTER SYSTEM SET shared_buffers = '4GB';
```

### Redis优化

```redis
# 配置持久化
appendonly yes
appendfsync everysec

# 配置内存策略
maxmemory 4gb
maxmemory-policy allkeys-lru
```

## 安全配置

### 网络安全

```bash
# 配置防火墙
sudo ufw allow 50051/tcp
sudo ufw allow 8080/tcp
sudo ufw enable
```

### 数据加密

```bash
# 启用TLS
export TLS_ENABLED=true
export TLS_CERT_FILE=/path/to/cert.pem
export TLS_KEY_FILE=/path/to/key.pem
```

## 备份和恢复

### 数据库备份

```bash
# 备份数据库
pg_dump -h localhost -p 5432 -U sensitive_data_user sensitive_data > backup.sql

# 恢复数据库
psql -h localhost -p 5432 -U sensitive_data_user sensitive_data < backup.sql
```

### 配置备份

```bash
# 备份配置文件
tar -czf config_backup.tar.gz configs/

# 恢复配置文件
tar -xzf config_backup.tar.gz
```

## 升级指南

### 版本升级

```bash
# 停止旧版本服务
docker-compose -f docker-compose.sensitive_data.yml down

# 拉取新版本代码
git pull origin main

# 重新构建服务
docker-compose -f docker-compose.sensitive_data.yml build

# 启动新版本服务
docker-compose -f docker-compose.sensitive_data.yml up -d
```

### 数据库迁移

```bash
# 运行数据库迁移
./bin/sensitive_level_service migrate
./bin/approval_flow_service migrate
```

## 联系支持

如果遇到问题，请：
1. 查看日志文件
2. 检查网络连接
3. 确认所有依赖服务正常运行
4. 联系技术支持团队

---

**文档版本**: v1.0.0  
**最后更新**: 2024年1月1日  
**维护团队**: Bytebase技术团队