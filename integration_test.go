package main

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestIntegration_ConfigLoad 集成测试：配置加载
func TestIntegration_ConfigLoad(t *testing.T) {
	// 创建完整的配置文件
	configContent := `
source:
  type: "kafka"
  brokers:
    - "localhost:9092"
  topic: "source-topic"
  group_id: "test-group"

target:
  brokers:
    - "localhost:9093"
  topic: "target-topic"

mirror:
  worker_count: 2
  consumer_rate_limit: 100
  producer_rate_limit: 50

log:
  file_path: "integration-test.log"
  stats_interval: 5s

server:
  enabled: true
  address: ":8081"

retry:
  enabled: true
  max_retries: 2

dedup:
  enabled: true
  strategy: "key_value"
  ttl: 1h
`
	
	tmpFile, err := os.CreateTemp("", "integration-test-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	tmpFile.WriteString(configContent)
	tmpFile.Close()
	
	// 加载配置
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	
	// 验证所有配置项
	if config.Source.Type != "kafka" {
		t.Errorf("source.type配置错误")
	}
	
	if config.Mirror.WorkerCount != 2 {
		t.Errorf("mirror.worker_count配置错误")
	}
	
	if !config.Retry.Enabled {
		t.Error("retry.enabled应该为true")
	}
	
	if !config.Dedup.Enabled {
		t.Error("dedup.enabled应该为true")
	}
	
	// 清理
	os.Remove("integration-test.log")
}

// TestIntegration_PluginCreation 集成测试：插件创建
func TestIntegration_PluginCreation(t *testing.T) {
	// 测试创建Kafka插件
	kafkaPlugin, err := CreatePlugin("kafka")
	if err != nil {
		t.Fatalf("创建Kafka插件失败: %v", err)
	}
	
	if kafkaPlugin.Name() != "kafka" {
		t.Errorf("期望插件名'kafka'，实际'%s'", kafkaPlugin.Name())
	}
	
	// 测试创建RabbitMQ插件
	rabbitmqPlugin, err := CreatePlugin("rabbitmq")
	if err != nil {
		t.Fatalf("创建RabbitMQ插件失败: %v", err)
	}
	
	if rabbitmqPlugin.Name() != "rabbitmq" {
		t.Errorf("期望插件名'rabbitmq'，实际'%s'", rabbitmqPlugin.Name())
	}
	
	// 测试创建文件插件
	filePlugin, err := CreatePlugin("file")
	if err != nil {
		t.Fatalf("创建文件插件失败: %v", err)
	}
	
	if filePlugin.Name() != "file" {
		t.Errorf("期望插件名'file'，实际'%s'", filePlugin.Name())
	}
	
	// 测试不存在的插件
	_, err = CreatePlugin("not-exists")
	if err == nil {
		t.Error("应该返回插件未找到错误")
	}
}

// TestIntegration_MessageFlow 集成测试：消息流程
func TestIntegration_MessageFlow(t *testing.T) {
	// 创建消息
	msg := &Message{
		Key:       []byte("test-key"),
		Value:     []byte("test-value"),
		Headers:   map[string][]byte{"header1": []byte("value1")},
		Timestamp: time.Now(),
		Source:    "kafka",
		Metadata: map[string]interface{}{
			"topic":     "test-topic",
			"partition": int32(0),
		},
	}
	
	// 测试消息格式
	if len(msg.Key) == 0 {
		t.Error("消息Key不应该为空")
	}
	
	if len(msg.Value) == 0 {
		t.Error("消息Value不应该为空")
	}
	
	if msg.Source != "kafka" {
		t.Errorf("消息Source错误，期望'kafka'，实际'%s'", msg.Source)
	}
	
	// 测试metadata
	if topic, ok := msg.Metadata["topic"].(string); !ok || topic != "test-topic" {
		t.Error("消息metadata中的topic错误")
	}
}

// TestIntegration_LoggerAndMetrics 集成测试：日志和指标
func TestIntegration_LoggerAndMetrics(t *testing.T) {
	// 测试日志管理器
	logConfig := &LogConfig{
		FilePath:         "integration-test.log",
		StatsInterval:    5 * time.Second,
		RotateInterval:   1 * time.Hour,
		MaxArchiveFiles:  3,
		AsyncBufferSize:  100,
	}
	
	ctx := context.Background()
	logger, err := NewLogger(logConfig, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	defer func() {
		logger.Stop()
		os.Remove("integration-test.log")
	}()
	
	// 测试指标管理器
	metrics := NewMetrics()
	metrics.Register()
	
	// 记录一些指标
	metrics.RecordMessageConsumed(100)
	metrics.RecordMessageProduced(100)
	metrics.RecordLatency(10 * time.Millisecond)
	metrics.SetHealthStatus(true)
	
	// 更新速率
	metrics.UpdateRates()
	
	// 验证指标
	if metrics.MessagesConsumedTotal == nil {
		t.Error("MessagesConsumedTotal指标应该被注册")
	}
	
	// 清理
	logger.Stop()
}

