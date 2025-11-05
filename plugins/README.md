# 插件系统说明

本目录用于说明插件系统的组织方式。

## 插件文件位置

由于Go语言的限制（同一个包的所有文件必须在同一目录），所有插件实现文件都位于项目根目录，而不是本目录。

## 当前插件实现

插件实现文件位于根目录：

- `kafka_plugin.go` - Kafka数据源插件
- `rabbitmq_plugin.go` - RabbitMQ数据源插件
- `file_plugin.go` - 文件监控数据源插件

## 插件接口

插件接口定义在 `plugin.go` 中：

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

## 插件注册

插件通过 `init()` 函数自动注册：

```go
func init() {
    RegisterPlugin("kafka", NewKafkaPlugin)
    RegisterPlugin("rabbitmq", NewRabbitMQPlugin)
    RegisterPlugin("file", NewFilePlugin)
}
```

## 为什么插件文件在根目录？

1. **Go语言限制**: 同一个包（`package main`）的所有文件必须在同一目录
2. **自动注册**: 插件通过 `init()` 函数自动注册，需要在同一包中
3. **类型访问**: 插件需要访问 `Message`、`SourcePlugin` 等类型，这些都在 `package main` 中

## 未来扩展

如果将来需要将插件改为独立的包（`package plugins`），需要：

1. 将插件接口和类型移到 `plugins/` 目录
2. 将插件实现改为 `package plugins`
3. 修改插件注册机制
4. 处理类型导出和导入

## 添加新插件

要添加新的数据源插件：

1. 在根目录创建新的插件文件（如 `redis_plugin.go`）
2. 实现 `SourcePlugin` 接口
3. 在 `init()` 函数中注册插件
4. 更新配置文档

示例：

```go
// redis_plugin.go
package main

func NewRedisPlugin() SourcePlugin {
    return &RedisPlugin{}
}

func init() {
    RegisterPlugin("redis", NewRedisPlugin)
}
```

