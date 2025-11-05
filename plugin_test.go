package main

import (
	"context"
	"testing"
	"time"
)

func TestPluginRegistry(t *testing.T) {
	// 测试插件注册
	registry := &PluginRegistry{
		plugins: make(map[string]func() SourcePlugin),
	}
	
	// 注册测试插件
	testPluginCreated := false
	registry.Register("test", func() SourcePlugin {
		testPluginCreated = true
		return &testPlugin{}
	})
	
	// 创建插件
	plugin, err := registry.Create("test")
	if err != nil {
		t.Fatalf("创建插件失败: %v", err)
	}
	
	if !testPluginCreated {
		t.Error("插件工厂函数应该被调用")
	}
	
	if plugin == nil {
		t.Error("插件不应该为nil")
	}
	
	// 测试未注册的插件
	_, err = registry.Create("not-exists")
	if err == nil {
		t.Error("应该返回插件未找到错误")
	}
	
	_, ok := err.(*PluginNotFoundError)
	if !ok {
		t.Errorf("应该返回PluginNotFoundError，实际: %T", err)
	}
	
	// 测试列出插件
	plugins := registry.List()
	if len(plugins) != 1 {
		t.Errorf("期望1个插件，实际%d", len(plugins))
	}
	
	if plugins[0] != "test" {
		t.Errorf("期望插件名'test'，实际'%s'", plugins[0])
	}
}

func TestGlobalPluginRegistry(t *testing.T) {
	// 测试全局插件注册表
	// 注意：kafka、rabbitmq、file插件已经在init中注册
	
	plugins := globalRegistry.List()
	if len(plugins) == 0 {
		t.Error("应该有已注册的插件")
	}
	
	// 测试创建已注册的插件
	kafkaPlugin, err := CreatePlugin("kafka")
	if err != nil {
		t.Fatalf("创建Kafka插件失败: %v", err)
	}
	
	if kafkaPlugin == nil {
		t.Error("Kafka插件不应该为nil")
	}
	
	if kafkaPlugin.Name() != "kafka" {
		t.Errorf("期望插件名'kafka'，实际'%s'", kafkaPlugin.Name())
	}
}

// 测试插件实现
type testPlugin struct{}

func (p *testPlugin) Name() string {
	return "test"
}

func (p *testPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (p *testPlugin) Start(ctx context.Context) error {
	return nil
}

func (p *testPlugin) Stop() error {
	return nil
}

func (p *testPlugin) Messages() <-chan *Message {
	return make(chan *Message)
}

func (p *testPlugin) Ack(msg *Message) error {
	return nil
}

func (p *testPlugin) GetStats() PluginStats {
	return PluginStats{
		StartTime: time.Now(),
	}
}

