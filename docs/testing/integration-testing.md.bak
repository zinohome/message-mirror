# 集成测试框架指南

## 概述

message-mirror项目使用testcontainers-go进行集成测试，支持快速启动和清理Docker容器来测试与外部服务的交互。

## 设置要求

### 前提条件
- Docker Engine 17.09.0+
- 网络连接以拉取容器镜像
- 足够的磁盘空间

### 安装

```bash
go get github.com/testcontainers/testcontainers-go@latest
```

## 运行集成测试

### 跳过集成测试（快速开发）
```bash
go test -short ./...
```

### 仅运行集成测试
```bash
go test -tags integration -run Integration ./...
```

### 运行特定集成测试
```bash
go test -tags integration -run TestIntegration_ZookeeperContainer ./internal/core
```

### 运行所有测试（包括集成测试）
```bash
go test -tags integration ./...
```

## 可用的集成测试

### 1. TestIntegration_ZookeeperContainer
- **位置**: `internal/core/integration_container_test.go`
- **功能**: 启动Zookeeper容器，验证端口映射
- **依赖**: Docker
- **预期结果**: 容器启动成功，端口2181可访问

### 2. TestIntegration_RabbitMQContainer
- **位置**: `internal/core/integration_container_test.go`
- **功能**: 启动RabbitMQ容器，验证AMQP和Management端口
- **依赖**: Docker, RabbitMQ镜像
- **预期结果**: 容器启动，端口5672和15672可访问

### 3. TestIntegration_ConfigValidation
- **位置**: `internal/core/integration_container_test.go`
- **功能**: 测试配置验证逻辑
- **依赖**: 无（不需要Docker）
- **预期结果**: 配置验证通过

### 4. TestIntegration_MirrorMakerInitialization
- **位置**: `internal/core/integration_container_test.go`
- **功能**: 测试MirrorMaker初始化流程
- **依赖**: 无（使用未绑定的端口）
- **预期结果**: 初始化失败（预期，Kafka未运行），错误处理正确

### 5. TestIntegration_ContainerNetworking
- **位置**: `internal/core/integration_container_test.go`
- **功能**: 测试Docker网络创建和验证
- **依赖**: Docker
- **预期结果**: 网络创建成功，获取网络ID

## 测试架构

### 单容器测试
```go
// 创建容器
req := testcontainers.ContainerRequest{
    Image: "...",
    ExposedPorts: []string{"..."},
}

container, err := testcontainers.GenericContainer(ctx, 
    testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started: true,
    })
defer container.Terminate(ctx)

// 获取映射端口
port, _ := container.MappedPort(ctx, "...")
```

### 多容器协作
```go
// 创建网络
network, _ := testcontainers.GenericNetwork(ctx, ...)
defer network.Remove(ctx)

// 在网络中启动容器
```

## 最佳实践

### 1. 超时管理
```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()
```

### 2. 资源清理
```go
defer container.Terminate(ctx)  // 即使测试失败也会执行
```

### 3. 等待条件
```go
WaitingFor: wait.ForLog("startup message"),
// 或
WaitingFor: wait.ForHTTP("/health").WithPort("8080"),
```

### 4. 错误处理
```go
if err != nil {
    t.Logf("启动容器失败（可能缺少Docker）: %v", err)
    return  // 优雅地跳过
}
```

### 5. 快速迭代
```bash
# 使用 -short 标志跳过集成测试进行快速迭代
go test -short ./...

# 只在提交前运行完整测试
go test -tags integration ./...
```

## 故障排查

### Docker守护程序未运行
```
error: Cannot connect to the Docker daemon
```
**解决方案**: 启动Docker Engine

### 镜像拉取失败
```
error: Cannot pull image: net/http: request canceled
```
**解决方案**: 检查网络连接，手动拉取镜像：
```bash
docker pull confluentinc/cp-zookeeper:7.5.0
docker pull rabbitmq:3.12-management
```

### 端口冲突
```
error: bind: address already in use
```
**解决方案**: testcontainers自动使用空闲端口，但需要足够的可用端口

### 容器启动超时
```
error: waiting for container: deadline exceeded
```
**解决方案**: 增加超时时间，或检查系统资源

## 扩展集成测试

### 添加新的容器测试
```go
func TestIntegration_MyService(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试")
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    
    // 创建和测试容器
}
```

### 测试Kafka→Mirror→Kafka流程
```go
func TestIntegration_KafkaFlow(t *testing.T) {
    // 1. 启动源Kafka
    // 2. 启动目标Kafka
    // 3. 配置Mirror
    // 4. 发送测试消息到源Kafka
    // 5. 验证消息到达目标Kafka
    // 6. 验证消息去重和重试
}
```

## 性能考虑

- **首次运行**: 需要拉取Docker镜像（可能较慢）
- **后续运行**: 使用本地镜像缓存（快速）
- **资源占用**: 容器启动时占用内存，测试完成后自动释放
- **网络**: 容器间通信使用Docker网络（性能好）

## CI/CD集成

在GitHub Actions中运行集成测试：
```yaml
- name: Run integration tests
  run: go test -tags integration ./...
```

注意：GitHub Actions的Linux runner包含Docker，但需要配置适当的权限。

## 参考资源

- [testcontainers-go文档](https://golang.testcontainers.org/)
- [Docker镜像列表](https://hub.docker.com/)
- [go test命令文档](https://pkg.go.dev/cmd/go#hdr-Test_packages)
