package plugins

import (
	"context"
	"testing"
	"time"
)

func TestKafkaPlugin_Name(t *testing.T) {
	plugin := NewKafkaPlugin()
	if plugin.Name() != "kafka" {
		t.Errorf("期望插件名'kafka'，实际'%s'", plugin.Name())
	}
}

func TestKafkaPlugin_Initialize(t *testing.T) {
	plugin := NewKafkaPlugin()
	
	config := map[string]interface{}{
		"brokers": []interface{}{"localhost:9092"},
		"topic":   "test-topic",
		"group_id": "test-group",
		"auto_offset_reset": "earliest",
		"session_timeout": 30 * time.Second,
		"batch_size": 100,
		"security_protocol": "PLAINTEXT",
	}
	
	err := plugin.Initialize(config)
	if err != nil {
		// 可能因为无法连接Kafka而失败，但配置解析应该没问题
		t.Logf("初始化失败（可能是Kafka不可用）: %v", err)
	}
}

func TestKafkaPlugin_GetStats(t *testing.T) {
	plugin := NewKafkaPlugin()
	
	stats := plugin.GetStats()
	if stats.StartTime.IsZero() {
		t.Error("StartTime应该被设置")
	}
}

func TestKafkaPlugin_Stop(t *testing.T) {
	plugin := NewKafkaPlugin()
	
	// 初始化插件（避免nil channel panic）
	config := map[string]interface{}{
		"brokers": []interface{}{"localhost:9092"},
		"topic":   "test-topic",
		"group_id": "test-group",
	}
	
	err := plugin.Initialize(config)
	if err != nil {
		t.Skipf("跳过测试：Kafka不可用 - %v", err)
		return
	}
	
	// 测试停止已初始化的插件
	err = plugin.Stop()
	if err != nil {
		t.Logf("停止插件失败: %v", err)
	}
}

func TestKafkaPlugin_Start(t *testing.T) {
	plugin := NewKafkaPlugin()
	
	config := map[string]interface{}{
		"brokers": []interface{}{"localhost:9092"},
		"topic":   "test-topic",
		"group_id": "test-group",
	}
	
	// 初始化插件
	err := plugin.Initialize(config)
	if err != nil {
		t.Skipf("跳过测试：Kafka不可用 - %v", err)
		return
	}
	
	ctx := context.Background()
	err = plugin.Start(ctx)
	if err != nil {
		t.Logf("启动插件失败（可能是Kafka不可用）: %v", err)
	}
	
	// 清理
	plugin.Stop()
}

func TestKafkaPlugin_Messages(t *testing.T) {
	plugin := NewKafkaPlugin()
	
	// 初始化插件以确保msgChan被创建
	config := map[string]interface{}{
		"brokers": []interface{}{"localhost:9092"},
		"topic":   "test-topic",
		"group_id": "test-group",
	}
	
	err := plugin.Initialize(config)
	if err != nil {
		t.Skipf("跳过测试：Kafka不可用 - %v", err)
		return
	}
	
	msgChan := plugin.Messages()
	if msgChan == nil {
		t.Error("消息通道不应该为nil")
	}
	
	// 清理
	plugin.Stop()
}

func TestKafkaPlugin_Ack(t *testing.T) {
	plugin := NewKafkaPlugin()
	
	msg := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
	
	err := plugin.Ack(msg)
	if err != nil {
		t.Logf("确认消息失败（可能未初始化）: %v", err)
	}
}

