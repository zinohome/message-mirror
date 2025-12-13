package plugins

import (
	"context"
	"log"
	"sync"
	"time"

	"message-mirror/internal/pkg/logger"
)

// Message 统一的消息格式
type Message struct {
	Key       []byte
	Value     []byte
	Headers   map[string][]byte
	Timestamp time.Time
	Source    string                 // 消息来源标识
	Metadata  map[string]interface{} // 额外的元数据，插件可以自定义
}

// SourcePlugin 数据源插件接口
type SourcePlugin interface {
	// Name 返回插件名称
	Name() string

	// Initialize 初始化插件，传入配置
	Initialize(config map[string]interface{}) error

	// Start 启动插件，开始消费消息
	Start(ctx context.Context) error

	// Stop 停止插件
	Stop() error

	// Messages 返回消息通道
	Messages() <-chan *Message

	// Ack 确认消息已处理（可选，某些插件可能需要）
	Ack(msg *Message) error

	// GetStats 获取插件统计信息
	GetStats() PluginStats
}

// PluginStats 插件统计信息
type PluginStats struct {
	MessagesReceived int64
	MessagesAcked    int64
	Errors           int64
	LastMessageTime  time.Time
	StartTime        time.Time
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	plugins map[string]func() SourcePlugin
}

var globalRegistry = &PluginRegistry{
	plugins: make(map[string]func() SourcePlugin),
}

var (
	loggerMu       sync.RWMutex
	loggerInstance *logger.Logger
)

// Register 注册插件
func (r *PluginRegistry) Register(name string, factory func() SourcePlugin) {
	r.plugins[name] = factory
}

// Create 创建插件实例
func (r *PluginRegistry) Create(name string) (SourcePlugin, error) {
	factory, ok := r.plugins[name]
	if !ok {
		return nil, &PluginNotFoundError{Name: name}
	}
	return factory(), nil
}

// List 列出所有已注册的插件
func (r *PluginRegistry) List() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// PluginNotFoundError 插件未找到错误
type PluginNotFoundError struct {
	Name string
}

func (e *PluginNotFoundError) Error() string {
	return "插件未找到: " + e.Name
}

// RegisterPlugin 全局函数：注册插件
func RegisterPlugin(name string, factory func() SourcePlugin) {
	globalRegistry.Register(name, factory)
}

// CreatePlugin 全局函数：创建插件实例
func CreatePlugin(name string) (SourcePlugin, error) {
	return globalRegistry.Create(name)
}

// SetLogger 注入统一日志实例供所有插件使用
func SetLogger(l *logger.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	loggerInstance = l
}

func pluginLogf(format string, args ...interface{}) {
	loggerMu.RLock()
	l := loggerInstance
	loggerMu.RUnlock()

	if l != nil {
		l.Printf(format, args...)
		return
	}
	log.Printf(format, args...)
}

func pluginLogln(args ...interface{}) {
	loggerMu.RLock()
	l := loggerInstance
	loggerMu.RUnlock()

	if l != nil {
		l.Println(args...)
		return
	}
	log.Println(args...)
}
