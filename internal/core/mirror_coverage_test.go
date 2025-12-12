package core

import (
	"testing"
	"time"

	"message-mirror/internal/plugins"
)

// TestMirrorMaker_ApplyConsumerRateLimit 测试消费速率限制
func TestMirrorMaker_ApplyConsumerRateLimit(t *testing.T) {
	tests := []struct {
		name           string
		rateLimit      float64
		bytesLimit     float64
		messageSize    int
		expectBlocking bool
	}{
		{
			name:           "无速率限制",
			rateLimit:      0,
			bytesLimit:     0,
			messageSize:    100,
			expectBlocking: false,
		},
		{
			name:           "字节速率限制优先",
			rateLimit:      100,
			bytesLimit:     1000,
			messageSize:    100,
			expectBlocking: false,
		},
		{
			name:           "仅消息速率限制",
			rateLimit:      10,
			bytesLimit:     0,
			messageSize:    100,
			expectBlocking: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Source: SourceConfig{
					Type: "kafka",
					Config: map[string]interface{}{
						"brokers": []string{"localhost:9092"},
						"topic":   "test",
					},
				},
				Target: TargetConfig{
					Brokers: []string{"localhost:9092"},
					Topic:   "test-target",
				},
				Mirror: MirrorConfig{
					WorkerCount:       1,
					ConsumerRateLimit: tt.rateLimit,
					ConsumerBurstSize: 100,
					BytesRateLimit:    tt.bytesLimit,
					BytesBurstSize:    10000,
				},
				Log: LogConfig{
					FilePath:      "test.log",
					StatsInterval: 10 * time.Second,
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

			mm, err := NewMirrorMaker(config)
			if err != nil {
				t.Skipf("无法创建MirrorMaker（可能需要Kafka）: %v", err)
				return
			}
			defer mm.Stop()

			msg := &plugins.Message{
				Key:   []byte("test-key"),
				Value: make([]byte, tt.messageSize),
			}

			start := time.Now()
			// 使用mm.ctx而不是创建新的context
			err = mm.applyConsumerRateLimit(msg)
			duration := time.Since(start)

			if err == mm.ctx.Err() && !tt.expectBlocking {
				t.Errorf("期望不阻塞，但超时了")
			}

			if tt.expectBlocking && duration < 50*time.Millisecond {
				t.Logf("警告: 期望阻塞但快速返回 (%v)", duration)
			}
		})
	}
}

// TestMirrorMaker_GetTargetTopic 测试目标topic获取逻辑
func TestMirrorMaker_GetTargetTopic(t *testing.T) {
	tests := []struct {
		name            string
		configTopic     string
		messageSource   string
		messageMetadata map[string]interface{}
		expectedTopic   string
	}{
		{
			name:          "使用配置的topic",
			configTopic:   "configured-topic",
			messageSource: "kafka",
			expectedTopic: "configured-topic",
		},
		{
			name:          "从Kafka消息metadata获取",
			configTopic:   "",
			messageSource: "kafka",
			messageMetadata: map[string]interface{}{
				"topic": "source-topic",
			},
			expectedTopic: "source-topic",
		},
		{
			name:            "默认topic名称",
			configTopic:     "",
			messageSource:   "rabbitmq",
			messageMetadata: map[string]interface{}{},
			expectedTopic:   "mirrored-messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Source: SourceConfig{
					Type: "kafka",
					Config: map[string]interface{}{
						"brokers": []string{"localhost:9092"},
						"topic":   "test",
					},
				},
				Target: TargetConfig{
					Brokers: []string{"localhost:9092"},
					Topic:   tt.configTopic,
				},
				Mirror: MirrorConfig{
					WorkerCount: 1,
				},
				Log: LogConfig{
					FilePath:      "test.log",
					StatsInterval: 10 * time.Second,
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

			mm, err := NewMirrorMaker(config)
			if err != nil {
				t.Skipf("无法创建MirrorMaker（可能需要Kafka）: %v", err)
				return
			}
			defer mm.Stop()

			msg := &plugins.Message{
				Source:   tt.messageSource,
				Metadata: tt.messageMetadata,
			}

			topic := mm.getTargetTopic(msg)
			if topic != tt.expectedTopic {
				t.Errorf("期望topic=%s, 得到=%s", tt.expectedTopic, topic)
			}
		})
	}
}

// TestMirrorMaker_Stats 测试统计信息更新
func TestMirrorMaker_Stats(t *testing.T) {
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers": []string{"localhost:9092"},
				"topic":   "test",
			},
		},
		Target: TargetConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "test-target",
		},
		Mirror: MirrorConfig{
			WorkerCount: 1,
		},
		Log: LogConfig{
			FilePath:      "test.log",
			StatsInterval: 10 * time.Second,
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

	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("无法创建MirrorMaker（可能需要Kafka）: %v", err)
		return
	}
	defer mm.Stop()

	// 验证初始统计
	stats := mm.GetStats()
	if stats.MessagesConsumed != 0 {
		t.Errorf("初始MessagesConsumed应该是0, 得到=%d", stats.MessagesConsumed)
	}
	if stats.MessagesProduced != 0 {
		t.Errorf("初始MessagesProduced应该是0, 得到=%d", stats.MessagesProduced)
	}

	// 手动更新统计（模拟消息处理）
	mm.stats.mu.Lock()
	mm.stats.MessagesConsumed = 10
	mm.stats.BytesConsumed = 1000
	mm.stats.MessagesProduced = 9
	mm.stats.BytesProduced = 900
	mm.stats.Errors = 1
	mm.stats.mu.Unlock()

	// 验证更新后的统计
	stats = mm.GetStats()
	if stats.MessagesConsumed != 10 {
		t.Errorf("期望MessagesConsumed=10, 得到=%d", stats.MessagesConsumed)
	}
	if stats.MessagesProduced != 9 {
		t.Errorf("期望MessagesProduced=9, 得到=%d", stats.MessagesProduced)
	}
	if stats.Errors != 1 {
		t.Errorf("期望Errors=1, 得到=%d", stats.Errors)
	}
}

// TestMirrorMaker_ConvertHeaders 测试消息头转换
func TestMirrorMaker_ConvertHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string][]byte
		expect  int
	}{
		{
			name:    "空headers",
			headers: nil,
			expect:  0,
		},
		{
			name: "单个header",
			headers: map[string][]byte{
				"key1": []byte("value1"),
			},
			expect: 1,
		},
		{
			name: "多个headers",
			headers: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
				"key3": []byte("value3"),
			},
			expect: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Source: SourceConfig{
					Type: "kafka",
					Config: map[string]interface{}{
						"brokers": []string{"localhost:9092"},
						"topic":   "test",
					},
				},
				Target: TargetConfig{
					Brokers: []string{"localhost:9092"},
					Topic:   "test-target",
				},
				Mirror: MirrorConfig{
					WorkerCount: 1,
				},
				Log: LogConfig{
					FilePath:      "test.log",
					StatsInterval: 10 * time.Second,
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

			mm, err := NewMirrorMaker(config)
			if err != nil {
				t.Skipf("无法创建MirrorMaker（可能需要Kafka）: %v", err)
				return
			}
			defer mm.Stop()

			headers := mm.convertHeaders(tt.headers)
			if len(headers) != tt.expect {
				t.Errorf("期望headers数量=%d, 得到=%d", tt.expect, len(headers))
			}

			// 验证header内容
			for k, v := range tt.headers {
				found := false
				for _, h := range headers {
					if string(h.Key) == k && string(h.Value) == string(v) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("header %s=%s 未找到", k, string(v))
				}
			}
		})
	}
}
