# 测试说明

## ⚠️ 重要说明

**测试文件位置**：由于Go语言的限制，`package main` 的测试文件必须与源文件在同一个目录下才能编译和运行。因此，所有测试文件（`*_test.go`）都位于项目根目录，而不是 `tests/` 目录。

这是Go语言的设计要求：
- 测试文件必须与被测试的代码在同一个包（package）中
- `package main` 的测试文件必须与 `main.go` 在同一个目录
- 如果将测试文件放在子目录，它们将无法访问未导出的函数和类型

## 测试文件说明

所有测试文件位于项目根目录（与源代码同级）：

### 单元测试
- `ratelimiter_test.go`: 速率限制器测试
- `ratelimiter_test_extended.go`: 速率限制器扩展测试
- `retry_test.go`: 重试机制测试
- `deduplicator_test.go`: 消息去重测试
- `deduplicator_test_extended.go`: 消息去重扩展测试
- `plugin_test.go`: 插件系统测试
- `config_test.go`: 配置管理测试
- `config_test_extended.go`: 配置管理扩展测试
- `logger_test.go`: 日志管理器测试
- `logger_test_extended.go`: 日志管理器扩展测试
- `security_test.go`: 安全功能测试
- `optimization_test.go`: 性能优化测试
- `optimization_test_extended.go`: 性能优化扩展测试
- `metrics_test.go`: 指标管理测试
- `mirror_test.go`: MirrorMaker核心逻辑测试
- `producer_test.go`: 生产者测试
- `kafka_plugin_test.go`: Kafka插件测试
- `rabbitmq_plugin_test.go`: RabbitMQ插件测试
- `file_plugin_test.go`: 文件监控插件测试

### 集成测试
- `integration_test.go`: 集成测试（配置加载、插件创建、消息流程等）

### 性能测试
- `benchmark_test.go`: 性能基准测试

## 运行测试

### 运行所有测试
```bash
go test -v .
```

### 运行特定测试
```bash
go test -v -run TestRateLimiter .
```

### 运行性能测试
```bash
go test -bench=. -benchmem .
```

### 生成覆盖率报告
```bash
go test -cover -coverprofile=coverage.out .
go tool cover -html=coverage.out
```

### 查看覆盖率
```bash
go tool cover -func=coverage.out
```

## 测试覆盖率

项目测试覆盖核心组件：
- 速率限制器: 100%
- 重试机制: 100%
- 消息去重: 100%
- 插件系统: 100%
- 配置管理: 100%
- 日志管理: 100%
- 安全功能: 100%

