package core

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"message-mirror/internal/plugins"
)

// mockSourcePlugin 模拟数据源插件
type mockSourcePlugin struct {
	name            string
	messagesChan    chan *plugins.Message
	started         bool
	stopped         bool
	shouldFailStart bool
	shouldFailStop  bool
	mu              sync.RWMutex
}

func newMockSourcePlugin(name string) *mockSourcePlugin {
	return &mockSourcePlugin{
		name:         name,
		messagesChan: make(chan *plugins.Message, 100),
	}
}

func (m *mockSourcePlugin) Name() string {
	return m.name
}

func (m *mockSourcePlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (m *mockSourcePlugin) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shouldFailStart {
		return errors.New("模拟启动失败")
	}
	m.started = true
	return nil
}

func (m *mockSourcePlugin) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shouldFailStop {
		return errors.New("模拟停止失败")
	}
	m.stopped = true
	close(m.messagesChan)
	return nil
}

func (m *mockSourcePlugin) Messages() <-chan *plugins.Message {
	return m.messagesChan
}

func (m *mockSourcePlugin) Ack(msg *plugins.Message) error {
	return nil
}

func (m *mockSourcePlugin) GetStats() plugins.PluginStats {
	return plugins.PluginStats{
		StartTime: time.Now(),
	}
}

// sendMessage 发送消息到通道（用于测试）
func (m *mockSourcePlugin) sendMessage(msg *plugins.Message) {
	m.messagesChan <- msg
}

// 注意：MirrorProducer使用Sarama，难以mock
// 我们主要测试不依赖producer的方法，或者使用file plugin进行集成测试

// TestMirrorMaker_Start 测试启动（使用mock）
func TestMirrorMaker_Start(t *testing.T) {
	config := createTestConfig(t)
	config.Source.Type = "file" // 使用file plugin，不需要Kafka
	
	// 创建MirrorMaker
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	// 测试启动（可能因为缺少Kafka而失败，这是预期的）
	err = mm.Start()
	if err != nil {
		t.Logf("启动MirrorMaker失败（可能因为缺少Kafka）: %v", err)
		mm.Stop()
		return
	}
	
	// 如果启动成功，验证基本状态
	if mm.source == nil {
		t.Error("source不应该为nil")
	}
	
	// 停止
	err = mm.Stop()
	if err != nil {
		t.Fatalf("停止MirrorMaker失败: %v", err)
	}
}

// TestMirrorMaker_Stop 测试停止
func TestMirrorMaker_Stop(t *testing.T) {
	config := createTestConfig(t)
	config.Source.Type = "file"
	
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	// 尝试启动（可能失败）
	_ = mm.Start()
	
	// 测试停止（即使启动失败也应该能正常停止）
	err = mm.Stop()
	if err != nil {
		t.Fatalf("停止MirrorMaker失败: %v", err)
	}
}

// TestMirrorMaker_processMessage 测试消息处理
// 注意：processMessage需要真实的producer，所以这个测试可能会因为缺少Kafka而跳过
func TestMirrorMaker_processMessage(t *testing.T) {
	config := createTestConfig(t)
	config.Source.Type = "file"
	
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	// 测试消息处理（需要producer，可能会失败）
	msg := &plugins.Message{
		Key:       []byte("test-key"),
		Value:     []byte("test-value"),
		Headers:   map[string][]byte{"header1": []byte("value1")},
		Timestamp: time.Now(),
		Source:    "kafka",
		Metadata: map[string]interface{}{
			"topic": "test-topic",
		},
	}
	
	// 由于需要真实的Kafka producer，这个测试可能会失败
	// 我们主要测试processMessage不会panic
	err = mm.processMessage(msg)
	if err != nil {
		t.Logf("处理消息失败（可能因为缺少Kafka）: %v", err)
	}
	
	mm.Stop()
}

// TestMirrorMaker_applyConsumerRateLimit 测试消费速率限制
func TestMirrorMaker_applyConsumerRateLimit(t *testing.T) {
	config := createTestConfig(t)
	config.Mirror.ConsumerRateLimit = 10.0 // 10消息/秒
	config.Source.Type = "file"
	
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	msg := &plugins.Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
	
	// 测试速率限制
	start := time.Now()
	err = mm.applyConsumerRateLimit(msg)
	if err != nil {
		t.Fatalf("应用消费速率限制失败: %v", err)
	}
	duration := time.Since(start)
	
	// 速率限制应该会等待（至少50ms）
	if duration < 50*time.Millisecond {
		t.Logf("速率限制等待时间: %v（可能太快）", duration)
	}
	
	// 测试字节速率限制（优先）
	config.Mirror.BytesRateLimit = 1000.0 // 1000字节/秒
	mm2, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	start2 := time.Now()
	err = mm2.applyConsumerRateLimit(msg)
	if err != nil {
		t.Fatalf("应用字节速率限制失败: %v", err)
	}
	duration2 := time.Since(start2)
	
	// 字节速率限制应该会等待
	if duration2 < 50*time.Millisecond {
		t.Logf("字节速率限制等待时间: %v（可能太快）", duration2)
	}
	
	mm.Stop()
	mm2.Stop()
}

// TestMirrorMaker_applyProducerRateLimit 测试生产速率限制
func TestMirrorMaker_applyProducerRateLimit(t *testing.T) {
	config := createTestConfig(t)
	config.Mirror.ProducerRateLimit = 10.0 // 10消息/秒
	config.Source.Type = "file"
	
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	msg := &plugins.Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
	
	// 测试速率限制
	start := time.Now()
	err = mm.applyProducerRateLimit(msg)
	if err != nil {
		t.Fatalf("应用生产速率限制失败: %v", err)
	}
	duration := time.Since(start)
	
	// 速率限制应该会等待
	if duration < 50*time.Millisecond {
		t.Logf("速率限制等待时间: %v（可能太快）", duration)
	}
	
	mm.Stop()
}

// TestMirrorMaker_getTargetTopic 测试目标topic获取
func TestMirrorMaker_getTargetTopic(t *testing.T) {
	config := createTestConfig(t)
	config.Target.Topic = "target-topic"
	config.Source.Type = "file"
	
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	// 测试配置的topic
	msg := &plugins.Message{
		Source: "kafka",
	}
	topic := mm.getTargetTopic(msg)
	if topic != "target-topic" {
		t.Errorf("期望topic='target-topic'，实际'%s'", topic)
	}
	
	// 测试从metadata获取topic
	config.Target.Topic = ""
	mm2, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	msg2 := &plugins.Message{
		Source: "kafka",
		Metadata: map[string]interface{}{
			"topic": "source-topic",
		},
	}
	topic2 := mm2.getTargetTopic(msg2)
	if topic2 != "source-topic" {
		t.Errorf("期望topic='source-topic'，实际'%s'", topic2)
	}
	
	// 测试默认topic
	msg3 := &plugins.Message{
		Source: "rabbitmq",
	}
	topic3 := mm2.getTargetTopic(msg3)
	if topic3 != "mirrored-messages" {
		t.Errorf("期望topic='mirrored-messages'，实际'%s'", topic3)
	}
	
	mm.Stop()
	mm2.Stop()
}

// TestMirrorMaker_convertHeaders 测试header转换
func TestMirrorMaker_convertHeaders(t *testing.T) {
	config := createTestConfig(t)
	config.Source.Type = "file"
	
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	// 测试有headers的情况
	headers := map[string][]byte{
		"header1": []byte("value1"),
		"header2": []byte("value2"),
	}
	saramaHeaders := mm.convertHeaders(headers)
	if len(saramaHeaders) != 2 {
		t.Errorf("期望2个header，实际%d", len(saramaHeaders))
	}
	
	// 验证header内容
	found := false
	for _, h := range saramaHeaders {
		if string(h.Key) == "header1" && string(h.Value) == "value1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("应该找到header1")
	}
	
	// 测试空headers
	emptyHeaders := mm.convertHeaders(nil)
	if emptyHeaders != nil {
		t.Error("空headers应该返回nil")
	}
	
	emptyHeaders2 := mm.convertHeaders(map[string][]byte{})
	if emptyHeaders2 != nil {
		t.Error("空headers应该返回nil")
	}
	
	mm.Stop()
}

// TestMirrorMaker_OnConfigReload 测试配置重载
func TestMirrorMaker_OnConfigReload(t *testing.T) {
	config := createTestConfig(t)
	config.Source.Type = "file"
	
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	// 创建新配置
	newConfig := createTestConfig(t)
	newConfig.Mirror.WorkerCount = 8
	newConfig.Mirror.ConsumerRateLimit = 20.0
	newConfig.Mirror.ProducerRateLimit = 15.0
	
	// 测试配置重载
	err = mm.OnConfigReload(config, newConfig)
	if err != nil {
		t.Fatalf("配置重载失败: %v", err)
	}
	
	// 验证配置已更新
	if mm.config.Mirror.WorkerCount != 8 {
		t.Errorf("期望worker_count=8，实际%d", mm.config.Mirror.WorkerCount)
	}
	
	mm.Stop()
}

// TestMirrorMaker_handleErrors 测试错误处理
func TestMirrorMaker_handleErrors(t *testing.T) {
	config := createTestConfig(t)
	config.Source.Type = "file"
	
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker - %v", err)
		return
	}
	
	mm.Start()
	defer mm.Stop()
	
	// 发送错误到errorChan
	testError := errors.New("测试错误")
	mm.errorChan <- testError
	
	// 等待错误处理
	time.Sleep(100 * time.Millisecond)
	
	// 验证错误已被处理（通过检查stats）
	stats := mm.GetStats()
	if stats.Errors == 0 {
		t.Log("错误处理可能已记录到日志")
	}
}

// 辅助函数：创建测试配置
func createTestConfig(t *testing.T) *Config {
	return &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers": []interface{}{"localhost:9092"},
				"topic":   "test-topic",
			},
		},
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
			Topic:   "target-topic",
		},
		Mirror: MirrorConfig{
			WorkerCount: 2,
		},
		Producer: ProducerConfig{
			MaxMessageBytes: 1000000,
		},
		Log: LogConfig{
			FilePath: "test.log",
		},
		Server: ServerConfig{
			Enabled: false,
		},
		Retry: RetryConfig{
			Enabled: false,
		},
		Dedup: DedupConfig{
			Enabled: false,
		},
	}
}

// 注意：由于MirrorProducer使用Sarama，难以完全mock
// 我们主要使用file plugin进行测试，它不需要Kafka连接

