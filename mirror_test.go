package main

import (
	"testing"
	"time"
)

func TestNewMirrorMaker(t *testing.T) {
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers": []interface{}{"localhost:9092"},
				"topic":   "test-topic",
			},
		},
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
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

	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker（可能需要Kafka） - %v", err)
		return
	}
	defer mm.Stop()

	if mm == nil {
		t.Error("MirrorMaker不应该为nil")
	}

	if mm.config != config {
		t.Error("配置应该被正确设置")
	}
}

func TestMirrorMaker_GetStats(t *testing.T) {
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers": []interface{}{"localhost:9092"},
				"topic":   "test-topic",
			},
		},
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
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

	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker（可能需要Kafka） - %v", err)
		return
	}
	defer mm.Stop()

	stats := mm.GetStats()
	if stats.MessagesConsumed != 0 {
		t.Errorf("初始消息数应该为0，实际%d", stats.MessagesConsumed)
	}
}

func TestStats_Update(t *testing.T) {
	stats := &Stats{
		StartTime: time.Now(),
	}

	stats.mu.Lock()
	stats.MessagesConsumed = 10
	stats.MessagesProduced = 8
	stats.BytesConsumed = 1000
	stats.BytesProduced = 800
	stats.Errors = 2
	stats.mu.Unlock()

	if stats.MessagesConsumed != 10 {
		t.Errorf("期望MessagesConsumed=10，实际%d", stats.MessagesConsumed)
	}

	if stats.MessagesProduced != 8 {
		t.Errorf("期望MessagesProduced=8，实际%d", stats.MessagesProduced)
	}

	if stats.Errors != 2 {
		t.Errorf("期望Errors=2，实际%d", stats.Errors)
	}
}

func TestMirrorMaker_GetMetrics(t *testing.T) {
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers": []interface{}{"localhost:9092"},
				"topic":   "test-topic",
			},
		},
		Target: TargetConfig{
			Brokers: []string{"localhost:9093"},
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

	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Skipf("跳过测试：无法创建MirrorMaker（可能需要Kafka） - %v", err)
		return
	}
	defer mm.Stop()

	metrics := mm.GetMetrics()
	if metrics == nil {
		t.Error("Metrics不应该为nil")
	}
}

