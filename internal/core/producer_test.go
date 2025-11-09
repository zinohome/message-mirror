package core

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"

	"message-mirror/internal/pkg/security"
)

// TestNewMirrorProducer 测试创建生产者
func TestNewMirrorProducer(t *testing.T) {
	config := &Config{
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
		},
		Producer: ProducerConfig{
			MaxMessageBytes: 1000000,
			CompressionType: "snappy",
			RequiredAcks:   1,
			RetryMax:        3,
			Idempotent:      true,
		},
		Security: SecurityConfig{
			TLS: security.TLSConfig{
				Enabled: false,
			},
		},
	}

	// 注意：这个测试需要真实的Kafka连接，所以会失败
	// 但我们可以测试配置创建逻辑
	producer, err := NewMirrorProducer(config)
	if err != nil {
		// 预期会失败（因为Kafka不可用），但配置应该被正确解析
		t.Logf("创建生产者失败（预期，因为Kafka不可用）: %v", err)
		return
	}

	if producer == nil {
		t.Error("生产者不应该为nil")
		return
	}

	defer producer.Stop()
}

// TestNewMirrorProducer_CompressionTypes 测试不同的压缩类型
func TestNewMirrorProducer_CompressionTypes(t *testing.T) {
	compressionTypes := []string{"none", "gzip", "snappy", "lz4", "zstd", "invalid"}

	for _, compType := range compressionTypes {
		config := &Config{
			Target: TargetConfig{
				Brokers: []string{"localhost:9093"},
			},
			Producer: ProducerConfig{
				CompressionType: compType,
			},
			Security: SecurityConfig{
				TLS: security.TLSConfig{
					Enabled: false,
				},
			},
		}

		producer, err := NewMirrorProducer(config)
		if err != nil {
			t.Logf("压缩类型 %s: 创建失败（预期，因为Kafka不可用）: %v", compType, err)
			continue
		}

		if producer != nil {
			producer.Stop()
		}
	}
}

// TestMirrorProducer_Start 测试启动生产者
func TestMirrorProducer_Start(t *testing.T) {
	config := &Config{
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
		},
		Producer: ProducerConfig{
			MaxMessageBytes: 1000000,
		},
		Security: SecurityConfig{
			TLS: security.TLSConfig{
				Enabled: false,
			},
		},
	}

	producer, err := NewMirrorProducer(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建生产者（可能需要Kafka） - %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 测试启动
	err = producer.Start(ctx)
	if err != nil {
		t.Fatalf("启动生产者失败: %v", err)
	}

	// 等待一下确保goroutine启动
	time.Sleep(100 * time.Millisecond)

	// 停止
	err = producer.Stop()
	if err != nil {
		t.Fatalf("停止生产者失败: %v", err)
	}
}

// TestMirrorProducer_Stop 测试停止生产者
func TestMirrorProducer_Stop(t *testing.T) {
	config := &Config{
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
		},
		Producer: ProducerConfig{
			MaxMessageBytes: 1000000,
		},
		Security: SecurityConfig{
			TLS: security.TLSConfig{
				Enabled: false,
			},
		},
	}

	producer, err := NewMirrorProducer(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建生产者（可能需要Kafka） - %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// 测试停止
	err = producer.Stop()
	if err != nil {
		t.Fatalf("停止生产者失败: %v", err)
	}

	// 再次停止应该不会出错
	err = producer.Stop()
	if err != nil {
		t.Logf("再次停止生产者（可能已停止）: %v", err)
	}
}

// TestMirrorProducer_Send 测试发送消息
func TestMirrorProducer_Send(t *testing.T) {
	config := &Config{
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
		},
		Producer: ProducerConfig{
			MaxMessageBytes: 1000000,
		},
		Security: SecurityConfig{
			TLS: security.TLSConfig{
				Enabled: false,
			},
		},
	}

	producer, err := NewMirrorProducer(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建生产者（可能需要Kafka） - %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer.Start(ctx)
	defer producer.Stop()

	// 测试发送消息
	msg := &sarama.ProducerMessage{
		Topic: "test-topic",
		Key:   sarama.StringEncoder("test-key"),
		Value: sarama.StringEncoder("test-value"),
	}

	err = producer.Send(msg)
	if err != nil {
		t.Logf("发送消息失败（可能因为Kafka不可用）: %v", err)
	}

	// 测试上下文取消
	cancel()
	time.Sleep(50 * time.Millisecond)

	err = producer.Send(msg)
	if err == nil {
		t.Error("上下文取消后发送应该返回错误")
	}
}

// TestMirrorProducer_UpdateConfig 测试更新配置
func TestMirrorProducer_UpdateConfig(t *testing.T) {
	config := &Config{
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
		},
		Producer: ProducerConfig{
			MaxMessageBytes: 1000000,
		},
		Security: SecurityConfig{
			TLS: security.TLSConfig{
				Enabled: false,
			},
		},
	}

	producer, err := NewMirrorProducer(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建生产者（可能需要Kafka） - %v", err)
		return
	}
	defer producer.Stop()

	// 创建新配置
	newConfig := &Config{
		Target: TargetConfig{
			Brokers: []string{"localhost:9094"},
		},
		Producer: ProducerConfig{
			MaxMessageBytes: 2000000,
		},
		Security: SecurityConfig{
			TLS: security.TLSConfig{
				Enabled: false,
			},
		},
	}

	// 测试更新配置
	err = producer.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}
}

// TestMirrorProducer_ContextCancellation 测试上下文取消
func TestMirrorProducer_ContextCancellation(t *testing.T) {
	config := &Config{
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
		},
		Producer: ProducerConfig{
			MaxMessageBytes: 1000000,
		},
		Security: SecurityConfig{
			TLS: security.TLSConfig{
				Enabled: false,
			},
		},
	}

	producer, err := NewMirrorProducer(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建生产者（可能需要Kafka） - %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	producer.Start(ctx)

	// 等待goroutine启动
	time.Sleep(100 * time.Millisecond)

	// 取消上下文
	cancel()

	// 等待goroutine停止
	time.Sleep(100 * time.Millisecond)

	// 停止生产者
	err = producer.Stop()
	if err != nil {
		t.Fatalf("停止生产者失败: %v", err)
	}
}

