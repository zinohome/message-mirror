package core

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_WithSecurity(t *testing.T) {
	configContent := `
source:
  type: "kafka"
  brokers:
    - "localhost:9092"
  topic: "test-topic"

target:
  brokers:
    - "localhost:9093"

security:
  tls:
    enabled: true
    min_version: "1.2"
  audit_enabled: true
`
	
	tmpFile, err := os.CreateTemp("", "test-config-security-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	tmpFile.WriteString(configContent)
	tmpFile.Close()
	
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	
	if !config.Security.TLS.Enabled {
		t.Error("期望security.tls.enabled=true")
	}
	
	if config.Security.TLS.MinVersion != "1.2" {
		t.Errorf("期望security.tls.min_version='1.2'，实际'%s'", config.Security.TLS.MinVersion)
	}
	
	if !config.Security.AuditEnabled {
		t.Error("期望security.audit_enabled=true")
	}
}

func TestLoadConfig_WithDedup(t *testing.T) {
	configContent := `
source:
  type: "kafka"
  brokers:
    - "localhost:9092"
  topic: "test-topic"

target:
  brokers:
    - "localhost:9093"

dedup:
  enabled: true
  strategy: "hash"
  ttl: 12h
  max_entries: 500000
  cleanup_interval: 30m
`
	
	tmpFile, err := os.CreateTemp("", "test-config-dedup-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	tmpFile.WriteString(configContent)
	tmpFile.Close()
	
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	
	if !config.Dedup.Enabled {
		t.Error("期望dedup.enabled=true")
	}
	
	if config.Dedup.Strategy != "hash" {
		t.Errorf("期望dedup.strategy='hash'，实际'%s'", config.Dedup.Strategy)
	}
	
	if config.Dedup.TTL != 12*time.Hour {
		t.Errorf("期望dedup.ttl=12h，实际%v", config.Dedup.TTL)
	}
	
	if config.Dedup.MaxEntries != 500000 {
		t.Errorf("期望dedup.max_entries=500000，实际%d", config.Dedup.MaxEntries)
	}
}

func TestLoadConfig_WithRetry(t *testing.T) {
	configContent := `
source:
  type: "kafka"
  brokers:
    - "localhost:9092"
  topic: "test-topic"

target:
  brokers:
    - "localhost:9093"

retry:
  enabled: true
  max_retries: 5
  initial_interval: 200ms
  max_interval: 20s
  multiplier: 3.0
  jitter: false
`
	
	tmpFile, err := os.CreateTemp("", "test-config-retry-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	tmpFile.WriteString(configContent)
	tmpFile.Close()
	
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	
	if !config.Retry.Enabled {
		t.Error("期望retry.enabled=true")
	}
	
	if config.Retry.MaxRetries != 5 {
		t.Errorf("期望retry.max_retries=5，实际%d", config.Retry.MaxRetries)
	}
	
	if config.Retry.InitialInterval != 200*time.Millisecond {
		t.Errorf("期望retry.initial_interval=200ms，实际%v", config.Retry.InitialInterval)
	}
	
	if config.Retry.Multiplier != 3.0 {
		t.Errorf("期望retry.multiplier=3.0，实际%f", config.Retry.Multiplier)
	}
	
	if config.Retry.Jitter {
		t.Error("期望retry.jitter=false")
	}
}

func TestValidateConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "worker_count为负数",
			config: &Config{
				Source: SourceConfig{Type: "kafka"},
				Target: TargetConfig{Brokers: []string{"localhost:9092"}},
				Mirror: MirrorConfig{WorkerCount: -1},
			},
			wantErr: true,
		},
		{
			name: "worker_count为0",
			config: &Config{
				Source: SourceConfig{Type: "kafka"},
				Target: TargetConfig{Brokers: []string{"localhost:9092"}},
				Mirror: MirrorConfig{WorkerCount: 0},
			},
			wantErr: true,
		},
		{
			name: "worker_count为1（最小值）",
			config: &Config{
				Source: SourceConfig{Type: "kafka"},
				Target: TargetConfig{Brokers: []string{"localhost:9092"}},
				Mirror: MirrorConfig{WorkerCount: 1},
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

