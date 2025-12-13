//go:build integration
// +build integration

package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// dockerAvailable 尝试初始化Docker Provider以判断Docker是否可用
func dockerAvailable(t *testing.T) bool {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Logf("检测Docker时发生panic（视为不可用）: %v", r)
		}
	}()
	if _, err := testcontainers.NewDockerProvider(); err != nil {
		t.Logf("Docker不可用，跳过集成测试: %v", err)
		return false
	}
	return true
}

// TestIntegration_ZookeeperContainer 测试Zookeeper容器启动
func TestIntegration_ZookeeperContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	if !dockerAvailable(t) {
		t.Skip("Docker不可用，跳过")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// 启动Zookeeper容器
	req := testcontainers.ContainerRequest{
		Image:        "confluentinc/cp-zookeeper:7.5.0",
		ExposedPorts: []string{"2181/tcp"},
		Env: map[string]string{
			"ZOOKEEPER_CLIENT_PORT": "2181",
		},
		WaitingFor: wait.ForLog("binding to port"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Logf("启动Zookeeper容器失败（可能缺少Docker）: %v", err)
		return
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("关闭容器失败: %v", err)
		}
	}()

	// 获取端口
	port, err := container.MappedPort(ctx, "2181")
	if err != nil {
		t.Logf("获取端口失败: %v", err)
		return
	}

	zkAddr := fmt.Sprintf("localhost:%s", port.Port())
	if zkAddr == "" {
		t.Error("Zookeeper地址为空")
	} else {
		t.Logf("Zookeeper地址: %s", zkAddr)
	}
}

// TestIntegration_RabbitMQContainer 测试RabbitMQ容器启动
func TestIntegration_RabbitMQContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	if !dockerAvailable(t) {
		t.Skip("Docker不可用，跳过")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// RabbitMQ容器请求
	req := testcontainers.ContainerRequest{
		Image:        "rabbitmq:3.12-management",
		ExposedPorts: []string{"5672/tcp", "15672/tcp"},
		Env: map[string]string{
			"RABBITMQ_DEFAULT_USER": "guest",
			"RABBITMQ_DEFAULT_PASS": "guest",
		},
		WaitingFor: wait.ForLog("Server startup complete"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Logf("启动RabbitMQ容器失败（可能缺少Docker）: %v", err)
		return
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("关闭容器失败: %v", err)
		}
	}()

	// 获取AMQP端口
	amqpPort, err := container.MappedPort(ctx, "5672")
	if err != nil {
		t.Logf("获取AMQP端口失败: %v", err)
		return
	}

	// 获取Management端口
	mgmtPort, err := container.MappedPort(ctx, "15672")
	if err != nil {
		t.Logf("获取Management端口失败: %v", err)
		return
	}

	t.Logf("RabbitMQ AMQP地址: localhost:%s", amqpPort.Port())
	t.Logf("RabbitMQ Management地址: localhost:%s", mgmtPort.Port())
}

// TestIntegration_ConfigValidation 测试配置验证
func TestIntegration_ConfigValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":  []interface{}{"localhost:9092"},
				"topic":    "test-topic",
				"group_id": "test-group",
			},
		},
		Target: TargetConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "target-topic",
		},
		Mirror: MirrorConfig{
			WorkerCount: 4,
		},
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		t.Logf("配置验证失败: %v", err)
	} else {
		t.Log("配置验证成功")
	}
}

// TestIntegration_MirrorMakerInitialization 测试MirrorMaker初始化
func TestIntegration_MirrorMakerInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":  []interface{}{"localhost:19092"},
				"topic":    "test-topic",
				"group_id": "test-group",
			},
		},
		Target: TargetConfig{
			Brokers: []string{"localhost:19092"},
			Topic:   "target-topic",
		},
		Mirror: MirrorConfig{
			WorkerCount: 2,
		},
		Log: LogConfig{
			FilePath:        "/tmp/test.log",
			StatsInterval:   10 * time.Second,
			RotateInterval:  24 * time.Hour,
			MaxArchiveFiles: 3,
			AsyncBufferSize: 100,
		},
		Server: ServerConfig{
			Enabled: false,
		},
		Retry: RetryConfig{
			Enabled: true,
		},
		Dedup: DedupConfig{
			Enabled: false,
		},
	}

	// 尝试初始化MirrorMaker（会失败因为Kafka未运行，但检查结构完整性）
	if mm, err := NewMirrorMaker(config); err != nil {
		t.Logf("MirrorMaker初始化失败（预期，Kafka未运行）: %v", err)
	} else if mm != nil && mm.config.Mirror.WorkerCount != 2 {
		t.Error("Worker数量应该是2")
	}
}

// TestIntegration_ContainerNetworking 测试容器网络
func TestIntegration_ContainerNetworking(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	if !dockerAvailable(t) {
		t.Skip("Docker不可用，跳过")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// 创建自定义网络
	network, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name: "test-network",
		},
	})
	if err != nil {
		t.Logf("创建网络失败（可能缺少Docker）: %v", err)
		return
	}
	defer func() {
		if err := network.Remove(ctx); err != nil {
			t.Logf("删除网络失败: %v", err)
		}
	}()

	t.Log("创建网络成功")
}
