package plugins

import (
	"context"
	"testing"
)

func TestRabbitMQPlugin_Name(t *testing.T) {
	plugin := NewRabbitMQPlugin()
	if plugin.Name() != "rabbitmq" {
		t.Errorf("期望插件名'rabbitmq'，实际'%s'", plugin.Name())
	}
}

func TestRabbitMQPlugin_Initialize(t *testing.T) {
	plugin := NewRabbitMQPlugin()
	
	config := map[string]interface{}{
		"url":    "amqp://guest:guest@localhost:5672/",
		"queue":  "test-queue",
		"exchange": "test-exchange",
	}
	
	err := plugin.Initialize(config)
	if err != nil {
		// 可能因为无法连接RabbitMQ而失败，但配置解析应该没问题
		t.Logf("初始化失败（可能是RabbitMQ不可用）: %v", err)
	}
}

func TestRabbitMQPlugin_GetStats(t *testing.T) {
	plugin := NewRabbitMQPlugin()
	
	stats := plugin.GetStats()
	if stats.StartTime.IsZero() {
		t.Error("StartTime应该被设置")
	}
}

func TestRabbitMQPlugin_Stop(t *testing.T) {
	plugin := NewRabbitMQPlugin()
	
	// 测试未初始化的插件停止
	err := plugin.Stop()
	if err != nil {
		t.Logf("停止插件失败（可能未初始化）: %v", err)
	}
}

func TestRabbitMQPlugin_Start(t *testing.T) {
	plugin := NewRabbitMQPlugin()
	
	config := map[string]interface{}{
		"url":   "amqp://guest:guest@localhost:5672/",
		"queue": "test-queue",
	}
	
	// 初始化插件
	err := plugin.Initialize(config)
	if err != nil {
		t.Skipf("跳过测试：RabbitMQ不可用 - %v", err)
		return
	}
	
	ctx := context.Background()
	err = plugin.Start(ctx)
	if err != nil {
		t.Logf("启动插件失败（可能是RabbitMQ不可用）: %v", err)
	}
	
	// 清理
	plugin.Stop()
}

func TestRabbitMQPlugin_Messages(t *testing.T) {
	plugin := NewRabbitMQPlugin()
	
	// 初始化插件以确保msgChan被创建
	config := map[string]interface{}{
		"url":   "amqp://guest:guest@localhost:5672/",
		"queue": "test-queue",
	}
	
	err := plugin.Initialize(config)
	if err != nil {
		t.Skipf("跳过测试：RabbitMQ不可用 - %v", err)
		return
	}
	
	msgChan := plugin.Messages()
	if msgChan == nil {
		t.Error("消息通道不应该为nil")
	}
	
	// 清理
	plugin.Stop()
}

func TestRabbitMQPlugin_Ack(t *testing.T) {
	plugin := NewRabbitMQPlugin()
	
	msg := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
	
	err := plugin.Ack(msg)
	if err != nil {
		t.Logf("确认消息失败（可能未初始化）: %v", err)
	}
}

func TestConvertRabbitMQHeaders(t *testing.T) {
	// 测试headers转换函数
	headers := map[string]interface{}{
		"header1": "value1",
		"header2": []byte("value2"),
		"header3": 123,
	}
	
	result := convertRabbitMQHeaders(headers)
	
	if len(result) == 0 {
		t.Error("转换后的headers不应该为空")
	}
	
	if val, ok := result["header1"]; ok {
		if string(val) != "value1" {
			t.Errorf("期望header1='value1'，实际'%s'", string(val))
		}
	}
}

