# Message Mirror - AI Agent Instructions

## Project Overview
Message Mirror is a pluggable message mirroring tool built in Go that reads from multiple data sources (Kafka, RabbitMQ, File) and writes to Kafka, similar to Apache Kafka's MirrorMaker.

## Architecture

### Core Components (3-Layer Architecture)
```
internal/
├── core/          # Mirror orchestration, config, producer
├── plugins/       # Data source plugins (Kafka, RabbitMQ, File)
└── pkg/           # Utilities (metrics, logger, retry, deduplicator, etc.)
```

**Data Flow**: `SourcePlugin` → `Message` (unified format) → `Worker pool` → `MirrorProducer` → Target Kafka

### Plugin System Pattern
- All plugins implement `plugins.SourcePlugin` interface
- Plugins register via `globalRegistry.Register(name, factory)` in `init()`
- Unified `Message` struct in [plugin.go](../internal/plugins/plugin.go) with Key, Value, Headers, Timestamp
- Example: [kafka_plugin.go](../internal/plugins/kafka_plugin.go), [rabbitmq_plugin.go](../internal/plugins/rabbitmq_plugin.go)

### Configuration Management
- Uses Viper with `mapstructure` tags for YAML → struct mapping
- Hot-reload support via `ConfigManager.ReloadConfig()` in [config_manager.go](../internal/core/config_manager.go)
- Source config uses `Type` field + plugin-specific `Config map[string]interface{}`
- Example configs: `config.yaml.example`, `config.rabbitmq.yaml`, `config.file.yaml`

## Development Workflows

### Build & Test
```bash
make build          # Builds with version/build time injection
make test           # Runs all tests
make run            # Build + run with config.yaml
make build-all      # Cross-platform builds (linux/darwin/windows, amd64/arm64)
make release        # Full release: test + build-all + package
```

### Docker Workflow
```bash
cd docker && docker-compose up -d     # Start Mirror + monitoring
docker-compose -f docker-compose.full.yml up  # Full stack (Kafka+Prometheus+Grafana)
```

### Testing Standards
- Test files co-located with source (e.g., `mirror_test.go`, `mirror_test_extended.go`)
- Table-driven tests with `t.Run(tc.name, func(t *testing.T) {...})`
- Coverage target: >80% for core components (see `coverage.html`)
- Use `context.Context` for cancellation in all async operations

## Project-Specific Conventions

### Error Handling
- **Always** wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Public API errors must be descriptive: `fmt.Errorf("无法连接到Kafka集群 %v: %w", brokers, err)`
- Never return raw errors from external libraries

### Concurrency Patterns
- Protect all shared state with `sync.RWMutex` (e.g., `Stats` struct in [mirror.go](../internal/core/mirror.go))
- Use `context.Context` for lifecycle management in goroutines
- Channel buffer sizes are configurable (e.g., `worker_count`, `channel_buffer_size`)
- Graceful shutdown via `wg.Wait()` + `cancel()` pattern

### Configuration Patterns
- All config structs use `mapstructure` tags for YAML binding
- Default values set in `SetDefault()` calls (see [config.go](../internal/core/config.go))
- Sensitive data (passwords) supported via encrypted config (see `internal/pkg/security/`)
- Example: `Source.Type` determines plugin, `Source.Config` holds plugin-specific fields

### Logging & Metrics
- Custom logger in `internal/pkg/logger/` with rotation, async buffering
- Prometheus metrics exposed at `/metrics` endpoint (port 8080 default)
- All metrics follow pattern: `mirror_<component>_<metric>` (e.g., `mirror_messages_consumed_total`)
- Log levels: Debug, Info, Warn, Error; configured via `log.level`

## Integration Points

### External Dependencies
- **Kafka**: IBM/sarama library, supports SASL_PLAINTEXT/SASL_SSL, compression (gzip/snappy/lz4/zstd)
- **RabbitMQ**: streadway/amqp, manual Ack after successful production
- **File Watcher**: fsnotify/fsnotify for inotify-based monitoring

### HTTP Server
- Health check: `GET /health` (liveness), `GET /ready` (readiness)
- Prometheus: `GET /metrics`
- Config API: `GET /config`, `POST /config/reload`
- Web UI: `GET /` (embedded in `web/ui.go`)

### Rate Limiting Strategy
- Three limiters: `consumerLimiter` (msg/s), `producerLimiter` (msg/s), `bytesLimiter` (bytes/s)
- `bytesLimiter` takes precedence if configured (more accurate throttling)
- Token bucket algorithm in `internal/pkg/ratelimiter/`

## Critical Patterns

### Adding a New Plugin
1. Implement `plugins.SourcePlugin` interface
2. Register in `init()`: `plugins.Register("mytype", func() plugins.SourcePlugin { return &MyPlugin{} })`
3. Convert source data to unified `plugins.Message` struct
4. Handle cleanup in `Stop()` method
5. Add config example in `config.<type>.yaml`

### Updating Core Logic
- Modifications to `MirrorMaker.Start()` require careful goroutine coordination
- Always use `mm.wg.Add(1)` before launching goroutines
- Respect `mm.ctx.Done()` for cancellation
- Update metrics counters atomically via mutex

### Batch Processing Pattern
- Enabled via `optimization.enable_batch` config
- `BatchProcessor` collects messages up to `batch_size` or `batch_timeout`
- Used in producer for higher throughput (see `internal/pkg/optimization/`)

## Common Pitfalls
- **DON'T** modify plugin configs after `Initialize()` - not thread-safe
- **DON'T** forget `defer producer.Close()` - causes Kafka partition leaks
- **DON'T** use `log.Println` - use `mm.logger.Info/Error/etc` for structured logging
- **DON'T** hardcode Kafka topic names - use `config.Target.Topic`

## Key Files for Reference
- Architecture: [system-architecture.md](../docs/architecture/system-architecture.md)
- Coding standards: [coding-standards.md](../docs/development/coding-standards.md)
- Project structure: [PROJECT_STRUCTURE.md](../PROJECT_STRUCTURE.md)
- Refactoring history: [REFACTORING_GUIDE.md](../REFACTORING_GUIDE.md)
- Full .cursorrules: [.cursorrules](../.cursorrules)
