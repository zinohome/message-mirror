# 开发指南

## 环境要求

### 必需
- Go 1.21 或更高版本
- Git
- Make (可选，但推荐)

### 可选
- Docker 和 Docker Compose (用于本地测试)
- Kafka (用于测试)
- RabbitMQ (用于测试)

## 环境搭建

### 1. 克隆项目

```bash
git clone https://github.com/zinohome/message-mirror.git
cd message-mirror
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 构建项目

```bash
make build
# 或
go build -o message-mirror .
```

### 4. 运行测试

```bash
make test
# 或
go test -v .
```

## 开发流程

### 1. 创建功能分支

```bash
git checkout -b feat/new-feature
```

### 2. 开发功能

- 编写代码
- 编写测试
- 更新文档

### 3. 运行测试

```bash
make test
go test -cover .
```

### 4. 提交代码

```bash
git add .
git commit -m "feat(scope): 功能描述"
```

### 5. 推送并创建PR

```bash
git push origin feat/new-feature
```

## 本地测试

### 使用Docker Compose

```bash
# 启动完整环境（包括Kafka）
make docker-compose-up-full

# 运行程序
make run

# 停止环境
make docker-compose-down-full
```

### 手动测试

1. 启动Kafka集群
2. 创建配置文件 `config.yaml`
3. 运行程序: `./message-mirror -c config.yaml`

## 调试

### 使用日志

```go
log.Printf("调试信息: %v", value)
```

### 使用调试器

```bash
# 使用Delve调试器
dlv debug .
```

### 查看指标

```bash
curl http://localhost:8080/metrics
```

## 性能分析

### CPU分析

```bash
go tool pprof http://localhost:8080/debug/pprof/profile
```

### 内存分析

```bash
go tool pprof http://localhost:8080/debug/pprof/heap
```

### 基准测试

```bash
go test -bench=. -benchmem
```

## 代码质量

### 格式化

```bash
go fmt ./...
```

### 静态分析

```bash
go vet ./...
golint ./...
```

### 测试覆盖率

```bash
go test -cover -coverprofile=coverage.out .
go tool cover -html=coverage.out
```

## 常见问题

### 依赖问题

```bash
go mod tidy
go mod download
```

### 编译错误

检查Go版本是否满足要求：
```bash
go version
```

### 测试失败

检查测试环境是否正确配置，特别是Kafka和RabbitMQ连接。

