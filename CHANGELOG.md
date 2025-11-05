# Changelog

所有重要的项目变更都会记录在这个文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [0.1.1] - 2025-11-05

### 新增
- ✨ 插件化架构，支持多种数据源插件（Kafka、RabbitMQ、文件监控）
- ✨ 支持从Kafka集群镜像消息到目标集群
- ✨ 支持从RabbitMQ队列消费消息并写入Kafka
- ✨ 支持文件系统监控，自动读取文件并写入Kafka
- ✨ 多Topic支持，可配置单个或多个topic
- ✨ 高并发处理，支持可配置的worker协程池
- ✨ 消息保留（key、value、headers、时间戳）
- ✨ 分区保留选项
- ✨ SASL认证支持（SASL_PLAINTEXT、SASL_SSL）
- ✨ 压缩支持（gzip、snappy、lz4、zstd）
- ✨ 幂等性生产，避免重复消息
- ✨ 统计监控，实时统计消息处理速率和错误
- ✨ 流控功能，支持消费和生产速率限制
- ✨ HTTP健康检查和就绪检查接口
- ✨ Prometheus指标暴露
- ✨ 延迟监控
- ✨ 错误重试机制（指数退避）
- ✨ 消息去重功能（支持多种策略）
- ✨ Docker化支持，包括docker-compose部署
- ✨ TLS/SSL支持
- ✨ 安全审计日志
- ✨ 性能优化（批处理、连接池、缓存）
- ✨ 完整的单元测试和集成测试
- ✨ 优雅关闭，支持信号处理
- ✨ 版本信息显示（--version命令）
- ✨ 多平台编译支持（Linux、macOS、Windows）

### 改进
- 🔧 配置管理优化，支持YAML配置文件
- 🔧 日志管理优化，支持文件轮转
- 🔧 错误处理改进，提供更详细的错误信息

### 文档
- 📝 完整的README文档
- 📝 部署指南（DEPLOYMENT.md）
- 📝 项目结构说明（PROJECT_STRUCTURE.md）
- 📝 插件系统文档（plugins/README.md）
- 📝 测试说明（tests/README.md）

### 测试
- ✅ 单元测试覆盖核心组件
- ✅ 集成测试覆盖配置加载和消息流程
- ✅ 性能基准测试

---

## [未发布]

### 计划中
- 消息转换和过滤功能
- 更多数据源插件支持
- 图形化监控界面
- 配置热重载增强

---

[0.1.1]: https://github.com/yourusername/message-mirror/releases/tag/v0.1.1

