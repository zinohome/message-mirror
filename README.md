# Message Mirror

一个用Go语言开发的**插件化消息镜像工具**，支持从多种数据源（Kafka、RabbitMQ、文件）读取消息并写入Kafka。类似于Apache Kafka的MirrorMaker，但使用Go语言实现，具有更好的性能和资源利用，并且支持插件化架构。

## 功能特性

- ✅ **插件化架构**: 支持多种数据源插件（Kafka、RabbitMQ、文件监控）
- ✅ **跨集群消息复制**: 从源Kafka集群消费消息并发送到目标集群
- ✅ **RabbitMQ支持**: 从RabbitMQ队列消费消息并写入Kafka
- ✅ **文件监控**: 监控指定目录下的文件，自动读取并写入Kafka
- ✅ **多Topic支持**: 支持单个或多个topic的镜像
- ✅ **高并发处理**: 可配置多个worker协程并行处理消息
- ✅ **消息保留**: 保留消息的key、value、headers和时间戳
- ✅ **分区保留**: 可选择保留源集群的分区信息
- ✅ **SASL认证**: 支持SASL_PLAINTEXT和SASL_SSL安全协议
- ✅ **压缩支持**: 支持多种压缩算法（gzip、snappy、lz4、zstd）
- ✅ **幂等性**: 支持幂等性生产，避免重复消息
- ✅ **统计监控**: 实时统计消息处理速率和错误信息
- ✅ **流控功能**: 支持消费和生产速率限制，防止过载
- ✅ **健康检查**: HTTP接口提供健康检查和就绪检查
- ✅ **Prometheus指标**: 暴露Prometheus格式的监控指标
- ✅ **延迟监控**: 实时监控消息处理延迟
- ✅ **错误重试**: 指数退避的重试机制，提高可靠性
- ✅ **消息去重**: 支持消息去重，避免重复处理
- ✅ **Docker支持**: 完整的Docker化支持，包括docker-compose部署
- ✅ **TLS/SSL支持**: 支持TLS加密连接，保护数据传输安全
- ✅ **安全审计**: 支持安全审计日志，记录配置变更和访问
- ✅ **性能优化**: 批处理、连接池、缓存等性能优化
- ✅ **单元测试**: 完整的单元测试和集成测试覆盖
- ✅ **优雅关闭**: 支持信号处理，实现优雅关闭
- ✅ **易于扩展**: 插件接口设计简单，方便添加新的数据源

## 架构原理

### 插件化架构

```
数据源插件 (SourcePlugin)
    ↓
统一消息格式 (Message)
    ↓
Worker协程池
    ↓
MirrorMaker生产者
    ↓
目标Kafka集群
```

### 支持的数据源

1. **Kafka插件**: 从Kafka集群消费消息
2. **RabbitMQ插件**: 从RabbitMQ队列消费消息
3. **文件监控插件**: 监控文件系统，读取文件内容

### 工作流程

1. **插件初始化**: 根据配置创建并初始化数据源插件
2. **消息消费**: 插件从数据源读取消息，转换为统一格式
3. **Worker处理**: 多个goroutine并行处理消息
4. **消息发送**: 异步生产者将消息发送到目标Kafka集群
5. **消息确认**: 处理成功后确认消息（如RabbitMQ的Ack、文件移动）
6. **统计监控**: 实时跟踪消息处理情况和性能指标

## 安装

### 前置要求

- Go 1.21 或更高版本（本地编译）
- Docker 和 Docker Compose（Docker部署）
- Kafka集群（源集群和目标集群）

### 本地编译

```bash
go mod download
go build -o message-mirror
```

### Docker部署

#### 快速开始

1. **克隆项目**:
```bash
git clone <repository-url>
cd message-mirror
```

2. **准备配置文件**:
```bash
mkdir -p config logs data
cp config.yaml.example config/config.yaml
# 编辑配置文件，设置Kafka集群地址等
vi config/config.yaml
```

3. **使用docker-compose启动**:
```bash
# 仅启动Message Mirror服务
cd docker && docker-compose up -d

# 或者启动完整环境（包括Kafka集群）
cd docker && docker-compose -f docker-compose.full.yml up -d
```

4. **查看日志**:
```bash
cd docker && docker-compose logs -f message-mirror
```

5. **停止服务**:
```bash
cd docker && docker-compose down
```

#### Docker镜像构建

```bash
# 构建镜像
docker build -f docker/Dockerfile -t message-mirror:latest .

# 运行容器
docker run -d \
  --name message-mirror \
  -p 8080:8080 \
  -v $(pwd)/config:/app/config:ro \
  -v $(pwd)/logs:/app/logs \
  -v $(pwd)/data:/app/data \
  message-mirror:latest
```

#### Docker Compose配置说明

项目提供了两个docker-compose文件：

1. **docker-compose.yml**: 仅包含Message Mirror服务，适用于已有Kafka集群的场景
2. **docker-compose.full.yml**: 包含完整的Kafka集群（Zookeeper + 源Kafka + 目标Kafka），适用于开发测试环境

#### 监控与可视化

启用监控功能（Prometheus + Grafana）:

```bash
docker-compose --profile monitoring up -d
```

访问地址:
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)
- Message Mirror健康检查: http://localhost:8080/health
- Prometheus指标: http://localhost:8080/metrics

#### 目录结构

```
.
├── config/              # 配置文件目录（挂载到容器）
├── logs/                # 日志文件目录（挂载到容器）
├── data/                # 数据目录（挂载到容器）
├── docker/               # Docker相关文件
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── docker-compose.full.yml
│   ├── .dockerignore
│   └── monitoring/      # 监控配置
│       ├── prometheus.yml
│       └── grafana/
├── tests/                # 测试文件
│   └── *_test.go
└── ...
```

#### 环境变量

- `CONFIG_PATH`: 配置文件路径（默认: `/app/config/config.yaml`）
- `TZ`: 时区设置（默认: `Asia/Shanghai`）

#### 健康检查

容器内置健康检查，可通过以下方式验证：

```bash
# 检查健康状态
curl http://localhost:8080/health

# 检查就绪状态
curl http://localhost:8080/ready

# 查看Prometheus指标
curl http://localhost:8080/metrics
```

## 配置

### Kafka数据源配置示例

创建配置文件 `config.yaml`：

```yaml
# 数据源配置（支持插件）
source:
  type: "kafka"  # 支持的类型: kafka, rabbitmq, file
  
  # Kafka插件配置
  brokers:
    - "localhost:9092"
  topic: "source-topic"
  group_id: "message-mirror-group"
  auto_offset_reset: "earliest"
  security_protocol: "PLAINTEXT"

# 目标Kafka集群配置
target:
  brokers:
    - "localhost:9093"
  topic: ""  # 如果为空，使用源topic名称

# 镜像配置
mirror:
  worker_count: 4

# 生产者配置
producer:
  compression_type: "snappy"
  idempotent: true
```

### RabbitMQ数据源配置示例

使用 `config.rabbitmq.yaml`：

```yaml
source:
  type: "rabbitmq"
  url: "amqp://guest:guest@localhost:5672/"
  exchange: "my-exchange"  # 可选
  queue: "my-queue"
  queue_options:
    durable: true
    auto_delete: false
```

### 文件监控数据源配置示例

使用 `config.file.yaml`：

```yaml
source:
  type: "file"
  watch_dir: "/data/watch"      # 监控目录
  finish_dir: "/data/finish"    # 完成目录（处理后的文件会移动到这里）
  file_pattern: "*.txt"         # 文件匹配模式
  poll_interval: 1s             # 轮询间隔
  recursive: false              # 是否递归监控子目录
```

### 配置说明

#### Source（数据源配置）

**通用配置**:
- `type`: 数据源类型（kafka、rabbitmq、file）

**Kafka插件配置**:
- `brokers`: Kafka broker地址列表
- `topic`: 单个topic名称
- `topics`: 多个topic名称列表（与topic二选一）
- `group_id`: 消费者组ID
- `auto_offset_reset`: 偏移量重置策略（earliest/latest）
- `session_timeout`: 会话超时时间
- `batch_size`: 批处理大小
- `security_protocol`: 安全协议（PLAINTEXT、SASL_PLAINTEXT、SASL_SSL）
- `sasl_username` / `sasl_password`: SASL认证信息

**RabbitMQ插件配置**:
- `url`: RabbitMQ连接URL
- `exchange`: Exchange名称（可选）
- `queue`: 队列名称
- `queue_options`: 队列选项（durable、auto_delete、exclusive）

**文件监控插件配置**:
- `watch_dir`: 监控目录路径
- `finish_dir`: 完成目录路径（处理后的文件会移动到这里）
- `file_pattern`: 文件匹配模式（如 "*.txt", "*.json", "*"）
- `poll_interval`: 轮询间隔
- `recursive`: 是否递归监控子目录

#### Target（目标Kafka集群配置）
- `brokers`: Kafka broker地址列表
- `topic`: 目标topic名称（为空则使用源topic名称或默认"mirrored-messages"）
- `security_protocol`: 安全协议

#### Mirror（镜像配置）
- `preserve_partition`: 是否保留分区信息（仅对Kafka源有效）
- `worker_count`: 工作协程数量（建议为CPU核心数）
- `consumer_rate_limit`: 消费速率限制（消息/秒，0表示不限制）
- `consumer_burst_size`: 消费突发大小（允许的突发消息数）
- `producer_rate_limit`: 生产速率限制（消息/秒，0表示不限制）
- `producer_burst_size`: 生产突发大小（允许的突发消息数）
- `bytes_rate_limit`: 字节速率限制（字节/秒，0表示不限制，优先于消息速率限制）
- `bytes_burst_size`: 字节突发大小（默认10MB）

#### Producer（生产者配置）
- `compression_type`: 压缩类型（none/gzip/snappy/lz4/zstd）
- `required_acks`: 确认机制（0/1/-1）
- `idempotent`: 是否启用幂等性

#### Log（日志配置）
- `file_path`: 日志文件路径（默认: "message-mirror.log"）
- `stats_interval`: 统计信息打印间隔（默认: 10s）
- `rotate_interval`: 日志轮转间隔，多长时间轮转一次（默认: 24h）
- `max_archive_files`: 保留的归档文件数量（默认: 7）
- `async_buffer_size`: 异步缓冲区大小（默认: 1000）

#### Server（HTTP服务器配置）
- `enabled`: 是否启用HTTP服务器（默认: true）
- `address`: HTTP服务器监听地址（默认: ":8080"）

#### Retry（重试配置）
- `enabled`: 是否启用重试机制（默认: true）
- `max_retries`: 最大重试次数（默认: 3）
- `initial_interval`: 初始重试间隔（默认: 100ms）
- `max_interval`: 最大重试间隔（默认: 10s）
- `multiplier`: 退避倍数，用于指数退避（默认: 2.0）
- `jitter`: 是否添加随机抖动（默认: true）

#### Dedup（去重配置）
- `enabled`: 是否启用消息去重（默认: false）
- `strategy`: 去重策略（默认: "key_value"）
  - `key`: 仅使用消息Key去重
  - `value`: 仅使用消息Value去重
  - `key_value`: 使用Key和Value组合去重（推荐）
  - `hash`: 使用完整消息内容（包括headers）去重
- `ttl`: 去重记录的TTL（过期时间，默认: 24h）
- `max_entries`: 最大去重记录数（默认: 1000000，防止内存溢出）
- `cleanup_interval`: 清理过期记录的间隔（默认: 1h）

#### Security（安全配置）
- `tls.enabled`: 是否启用TLS（默认: false）
- `tls.cert_file`: 客户端证书文件路径
- `tls.key_file`: 客户端私钥文件路径
- `tls.ca_file`: CA证书文件路径
- `tls.insecure_skip_verify`: 是否跳过证书验证（生产环境不推荐，默认: false）
- `tls.min_version`: 最小TLS版本（1.0, 1.1, 1.2, 1.3，默认: "1.2"）
- `tls.max_version`: 最大TLS版本（可选）
- `audit_enabled`: 是否启用安全审计日志（默认: false）
- `config_encryption`: 是否启用配置加密（默认: false）
- `encryption_key`: 加密密钥（建议使用环境变量）

## 使用方法

### 基本使用

```bash
./message-mirror -c config.yaml
```

### 使用不同数据源

**Kafka数据源**:
```bash
./message-mirror -c config.yaml
```

**RabbitMQ数据源**:
```bash
./message-mirror -c config.rabbitmq.yaml
```

**文件监控数据源**:
```bash
./message-mirror -c config.file.yaml
```

### 指定配置文件

```bash
./message-mirror --config /path/to/config.yaml
```

### 环境变量

可以通过环境变量 `CONFIG_PATH` 指定配置文件路径：

```bash
export CONFIG_PATH=/path/to/config.yaml
./message-mirror
```

## 流控功能

### 速率限制

程序支持完整的流控功能，防止读取和写入过快导致系统过载：

- **消费速率限制**: 限制从数据源读取消息的速率（消息/秒）
- **生产速率限制**: 限制向目标Kafka写入消息的速率（消息/秒）
- **字节速率限制**: 限制数据传输的字节速率（字节/秒，优先于消息速率限制）
- **突发支持**: 支持突发流量，允许短时间内超过限制

### 流控配置示例

```yaml
mirror:
  # 限制消费速率为1000消息/秒
  consumer_rate_limit: 1000
  consumer_burst_size: 200
  
  # 限制生产速率为500消息/秒
  producer_rate_limit: 500
  producer_burst_size: 100
  
  # 或者限制字节速率为10MB/秒（优先使用）
  bytes_rate_limit: 10485760  # 10MB = 10 * 1024 * 1024 字节
  bytes_burst_size: 20971520  # 20MB突发
```

### 流控说明

1. **字节速率限制优先**: 如果同时配置了字节速率限制和消息速率限制，字节速率限制会优先生效
2. **突发支持**: 突发大小允许在短时间内超过限制，提高系统响应性
3. **0表示不限制**: 将速率限制设置为0表示不限制该方向的速率

## 监控与健康检查

### HTTP接口

程序提供HTTP接口用于监控和健康检查：

- **健康检查**: `GET /health` - 检查服务健康状态
- **就绪检查**: `GET /ready` - 检查服务是否就绪
- **Prometheus指标**: `GET /metrics` - 暴露Prometheus格式的监控指标

### Prometheus指标

程序暴露以下Prometheus指标：

- `message_mirror_messages_consumed_total`: 消费消息总数
- `message_mirror_messages_produced_total`: 生产消息总数
- `message_mirror_messages_failed_total`: 失败消息总数
- `message_mirror_bytes_consumed_total`: 消费字节总数
- `message_mirror_bytes_produced_total`: 生产字节总数
- `message_mirror_message_latency_seconds`: 消息处理延迟（直方图）
- `message_mirror_message_rate_per_second`: 当前消息处理速率
- `message_mirror_byte_rate_per_second`: 当前字节处理速率
- `message_mirror_error_rate`: 当前错误率
- `message_mirror_health_status`: 健康状态（1=健康，0=不健康）

### 延迟监控

程序自动监控每条消息的处理延迟，并提供延迟分布直方图。延迟从消息消费开始到成功发送到目标集群的时间。

### 错误重试

程序支持指数退避的重试机制：

- **指数退避**: 重试间隔按配置的倍数递增
- **最大间隔限制**: 防止重试间隔过长
- **随机抖动**: 避免多个请求同时重试（thundering herd问题）
- **可配置**: 所有重试参数都可以在配置文件中设置

### 消息去重

程序支持消息去重功能，避免重复处理相同的消息：

- **多种去重策略**: 支持基于key、value、key+value或完整消息hash的去重
- **TTL支持**: 去重记录有过期时间，自动清理过期记录
- **内存管理**: 限制最大去重记录数，防止内存溢出
- **自动清理**: 定期清理过期记录，保持内存使用合理

#### 去重配置示例

```yaml
dedup:
  enabled: true
  strategy: "key_value"  # 使用key和value组合去重
  ttl: 24h               # 去重记录保留24小时
  max_entries: 1000000   # 最多保留100万条去重记录
  cleanup_interval: 1h   # 每小时清理一次过期记录
```

#### 去重策略说明

1. **key**: 仅使用消息Key去重，适合Key唯一的场景
2. **value**: 仅使用消息Value去重，适合Value内容唯一的场景
3. **key_value**: 使用Key和Value组合去重（推荐），平衡唯一性和性能
4. **hash**: 使用完整消息内容（包括headers）去重，最严格但性能稍低

## 性能优化建议

1. **Worker数量**: 根据CPU核心数和网络带宽调整 `worker_count`
2. **批处理大小**: 增加 `batch_size` 可以提高吞吐量，但会增加内存使用
3. **压缩**: 启用压缩可以减少网络传输，但会增加CPU使用
4. **幂等性**: 启用幂等性可以保证消息不重复，但会略微降低性能
5. **流控**: 合理设置速率限制，防止过载并保护目标系统
6. **重试机制**: 合理配置重试参数，平衡可靠性和性能
7. **消息去重**: 启用去重可以避免重复处理，但会增加内存使用

## 日志管理

### 日志配置

程序支持完整的日志管理功能：

- **日志文件路径**: 可配置日志文件保存位置
- **统计信息间隔**: 可配置统计信息打印间隔（默认10秒）
- **异步写入**: 日志采用异步写入，不阻塞主流程
- **自动轮转**: 按配置的时间间隔自动轮转日志文件
- **自动压缩**: 轮转后的日志文件自动压缩为tar.gz格式
- **归档管理**: 自动清理旧的归档文件，保留指定数量

### 日志配置示例

```yaml
log:
  file_path: "message-mirror.log"  # 日志文件路径
  stats_interval: 10s               # 统计信息打印间隔
  rotate_interval: 24h               # 日志轮转间隔（24小时）
  max_archive_files: 7               # 保留7个归档文件
  async_buffer_size: 1000           # 异步缓冲区大小
```

### 日志文件命名

- 当前日志: `message-mirror.log`
- 归档文件: `message-mirror.log.20240101-120000.tar.gz`

## 监控和统计

程序会根据配置的时间间隔自动打印统计信息到控制台和日志文件：

```
统计信息 - 运行时间: 1m30s, 消费消息: 10000 (111.11 msg/s), 生产消息: 10000, 错误: 0, 消费速率: 1024.00 KB/s
```

统计信息会同时输出到：
- 标准输出（控制台）
- 日志文件（如果配置了日志文件路径）

## 优雅关闭

程序支持优雅关闭，收到 SIGTERM 或 SIGINT 信号时会：
1. 停止接收新消息
2. 处理完当前消息
3. 关闭消费者和生产者
4. 打印最终统计信息

## 故障处理

- **消费者连接失败**: 程序会自动重试连接
- **生产者发送失败**: 错误会被记录到日志，消息会重试发送
- **网络中断**: 程序会尝试重新连接

## 插件开发

### 实现自定义插件

要实现新的数据源插件，需要实现 `SourcePlugin` 接口：

```go
type SourcePlugin interface {
    Name() string
    Initialize(config map[string]interface{}) error
    Start(ctx context.Context) error
    Stop() error
    Messages() <-chan *Message
    Ack(msg *Message) error
    GetStats() PluginStats
}
```

### 注册插件

在插件的 `init()` 函数中注册：

```go
func init() {
    RegisterPlugin("my-plugin", NewMyPlugin)
}
```

### 统一消息格式

所有插件都需要将数据转换为统一的 `Message` 格式：

```go
type Message struct {
    Key       []byte
    Value     []byte
    Headers   map[string][]byte
    Timestamp time.Time
    Source    string
    Metadata  map[string]interface{}
}
```

## 与Kafka MirrorMaker的对比

| 特性 | Kafka MirrorMaker | Message Mirror (Go) |
|------|------------------|---------------------|
| 语言 | Java | Go |
| 内存占用 | 较高 | 较低 |
| 启动速度 | 较慢 | 较快 |
| 并发模型 | 线程池 | Goroutine |
| 配置灵活性 | 中等 | 高 |
| 部署大小 | 较大 | 较小 |
| 插件化支持 | 否 | 是 |
| 多数据源支持 | 否（仅Kafka） | 是（Kafka、RabbitMQ、文件等） |
| Docker支持 | 是 | 是（完整支持） |

## Docker部署

详细的Docker部署说明请参考 [DEPLOYMENT.md](DEPLOYMENT.md)

项目结构说明请参考 [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md)

### 快速部署

```bash
# 准备配置文件
mkdir -p config logs data
cp config.yaml.example config/config.yaml

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f message-mirror

# 健康检查
curl http://localhost:8080/health
```

### Makefile支持

项目提供了Makefile，方便常用操作：

```bash
# 查看所有可用命令
make help

# 构建Docker镜像
make docker-build

# 启动Docker Compose服务
make docker-compose-up

# 启动完整环境（包括Kafka）
make docker-compose-up-full

# 查看日志
make docker-logs

# 健康检查
make health-check
```

### Docker Compose文件说明

- **docker-compose.yml**: 仅包含Message Mirror服务，适用于已有Kafka集群的场景
- **docker-compose.full.yml**: 包含完整的Kafka集群（Zookeeper + 源Kafka + 目标Kafka），适用于开发测试环境

### 监控集成

启动监控服务（Prometheus + Grafana）:

```bash
docker-compose --profile monitoring up -d
```

访问地址:
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)
- Message Mirror健康检查: http://localhost:8080/health
- Prometheus指标: http://localhost:8080/metrics

## 测试

### 运行测试

```bash
# 运行所有测试
go test -v ./tests/...

# 运行特定包的测试
go test -v ./tests/ratelimiter_test.go

# 运行性能测试
go test -bench=. -benchmem ./tests/...

# 运行测试并生成覆盖率报告
go test -cover -coverprofile=coverage.out ./tests/...
go tool cover -html=coverage.out
```

### 测试覆盖

项目包含以下测试：

- **单元测试**: 覆盖核心组件（限流器、重试器、去重器、插件系统等）
- **集成测试**: 测试配置加载、插件创建、消息流程等
- **性能测试**: 基准测试，评估关键路径性能
- **安全测试**: TLS配置、加密功能等

### 测试文件

所有测试文件位于`tests/`目录：

- `tests/ratelimiter_test.go`: 速率限制器测试
- `tests/retry_test.go`: 重试机制测试
- `tests/deduplicator_test.go`: 消息去重测试
- `tests/plugin_test.go`: 插件系统测试
- `tests/config_test.go`: 配置管理测试
- `tests/logger_test.go`: 日志管理器测试
- `tests/integration_test.go`: 集成测试
- `tests/benchmark_test.go`: 性能基准测试
- `tests/security_test.go`: 安全功能测试
- `tests/optimization_test.go`: 性能优化测试

详细说明请参考 [tests/README.md](tests/README.md)

## 性能优化

### 批处理优化

程序支持批处理优化，可以批量处理消息以提高吞吐量：

```yaml
# 在配置中启用批处理（通过worker_count和batch_size调整）
mirror:
  worker_count: 8  # 增加worker数量
```

### 连接池优化

程序内部维护连接池，复用Kafka连接，减少连接开销。

### 缓存优化

- **消息去重缓存**: 使用内存缓存，避免重复处理
- **配置缓存**: 缓存常用配置，减少IO操作

### 性能建议

1. **Worker数量**: 根据CPU核心数设置，建议为CPU核心数的1-2倍
2. **批处理大小**: 根据消息大小调整，大消息使用小批次，小消息使用大批次
3. **去重缓存**: 根据内存情况调整最大记录数
4. **压缩**: 网络带宽受限时启用压缩

## 安全增强

### TLS/SSL支持

程序支持TLS加密连接，保护数据传输安全：

```yaml
security:
  tls:
    enabled: true
    cert_file: "/path/to/client.crt"
    key_file: "/path/to/client.key"
    ca_file: "/path/to/ca.crt"
    min_version: "1.2"
```

### 安全审计

程序支持安全审计日志，记录配置变更和访问：

```yaml
security:
  audit_enabled: true
```

审计日志会记录：
- 配置变更
- 访问操作
- 安全事件

### 配置加密

程序支持配置加密，保护敏感信息：

```yaml
security:
  config_encryption: true
  encryption_key: "${ENCRYPTION_KEY}"  # 使用环境变量
```

### 安全建议

1. **生产环境**: 启用TLS，禁用`insecure_skip_verify`
2. **密钥管理**: 使用环境变量或密钥管理服务（如Vault）
3. **最小权限**: 容器以非root用户运行
4. **审计日志**: 生产环境启用安全审计
5. **证书管理**: 定期更新TLS证书

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！
