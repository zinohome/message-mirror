# Release Notes - v0.1.1

## 版本信息
- **版本号**: v0.1.1
- **发布日期**: 2025-11-05
- **Git提交**: $(git rev-parse --short HEAD)

## 下载

### Linux
- [Linux AMD64](message-mirror-v0.1.1-linux-amd64.tar.gz)
- [Linux ARM64](message-mirror-v0.1.1-linux-arm64.tar.gz)

### macOS
- [macOS AMD64](message-mirror-v0.1.1-darwin-amd64.tar.gz)
- [macOS ARM64 (Apple Silicon)](message-mirror-v0.1.1-darwin-arm64.tar.gz)

### Windows
- [Windows AMD64](message-mirror-v0.1.1-windows-amd64.zip)

## 安装

### Linux / macOS
```bash
# 下载并解压
tar -xzf message-mirror-v0.1.1-linux-amd64.tar.gz
# 或
tar -xzf message-mirror-v0.1.1-darwin-amd64.tar.gz

# 移动到 PATH 目录（可选）
sudo mv message-mirror-linux-amd64 /usr/local/bin/message-mirror
# 或
sudo mv message-mirror-darwin-amd64 /usr/local/bin/message-mirror
```

### Windows
```powershell
# 下载并解压
Expand-Archive message-mirror-v0.1.1-windows-amd64.zip

# 将 message-mirror-windows-amd64.exe 添加到 PATH
```

## 验证安装
```bash
message-mirror version
```

应该显示：
```
Message Mirror
版本: 0.1.1
构建时间: 2025-11-05_14:17:45
Git提交: b49abc9
```

## 主要特性

### 核心功能
- ✅ 插件化架构，支持多种数据源（Kafka、RabbitMQ、文件）
- ✅ 跨集群消息镜像
- ✅ 高并发处理
- ✅ 消息保留（key、value、headers、时间戳）
- ✅ 多Topic支持

### 可靠性
- ✅ 错误重试机制（指数退避）
- ✅ 消息去重
- ✅ 幂等性生产
- ✅ 优雅关闭

### 监控和可观测性
- ✅ Prometheus指标
- ✅ HTTP健康检查
- ✅ 统计监控
- ✅ 延迟监控

### 安全性
- ✅ SASL认证支持
- ✅ TLS/SSL支持
- ✅ 安全审计日志

### 性能优化
- ✅ 批处理支持
- ✅ 连接池
- ✅ 消息缓存
- ✅ 流控功能

### 部署
- ✅ Docker支持
- ✅ docker-compose部署
- ✅ 多平台编译

## 快速开始

### 1. 配置文件

创建 `config.yaml`：

```yaml
source:
  type: kafka
  brokers:
    - "localhost:9092"
  topics:
    - "test-topic"
  
target:
  brokers:
    - "localhost:9093"
  topic_prefix: "mirrored-"
  
mirror:
  workers: 4
  batch_enabled: true
  batch_size: 100
```

### 2. 运行

```bash
message-mirror -c config.yaml
```

### 3. 查看指标

```bash
curl http://localhost:8080/metrics
```

## 文档

- [README.md](README.md) - 完整文档
- [DEPLOYMENT.md](DEPLOYMENT.md) - 部署指南
- [CHANGELOG.md](CHANGELOG.md) - 变更日志
- [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md) - 项目结构

## 测试

运行测试：
```bash
make test
```

查看测试覆盖率：
```bash
go test -cover -coverprofile=coverage.out .
go tool cover -html=coverage.out
```

## 已知问题

- 无

## 后续计划

- 消息转换和过滤功能
- 更多数据源插件支持
- 图形化监控界面
- 配置热重载增强

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

[请在此添加许可证信息]

