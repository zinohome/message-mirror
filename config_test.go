package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// 创建临时配置文件
	configContent := `
source:
  type: "kafka"
  brokers:
    - "localhost:9092"
  topic: "test-topic"
  group_id: "test-group"

target:
  brokers:
    - "localhost:9093"
  topic: "target-topic"

mirror:
  worker_count: 4
  consumer_rate_limit: 1000
  producer_rate_limit: 500

log:
  file_path: "test.log"
  stats_interval: 10s

server:
  enabled: true
  address: ":8080"

retry:
  enabled: true
  max_retries: 3

dedup:
  enabled: true
  strategy: "key_value"
`
	
	// 写入临时文件
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}
	tmpFile.Close()
	
	// 加载配置
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	
	// 验证配置
	if config.Source.Type != "kafka" {
		t.Errorf("期望source.type='kafka'，实际'%s'", config.Source.Type)
	}
	
	if config.Mirror.WorkerCount != 4 {
		t.Errorf("期望worker_count=4，实际%d", config.Mirror.WorkerCount)
	}
	
	if config.Mirror.ConsumerRateLimit != 1000 {
		t.Errorf("期望consumer_rate_limit=1000，实际%f", config.Mirror.ConsumerRateLimit)
	}
	
	if !config.Retry.Enabled {
		t.Error("期望retry.enabled=true")
	}
	
	if !config.Dedup.Enabled {
		t.Error("期望dedup.enabled=true")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "有效配置",
			config: &Config{
				Source: SourceConfig{
					Type: "kafka",
				},
				Target: TargetConfig{
					Brokers: []string{"localhost:9092"},
				},
				Mirror: MirrorConfig{
					WorkerCount: 4,
				},
			},
			wantErr: false,
		},
		{
			name: "缺少数据源类型",
			config: &Config{
				Source: SourceConfig{
					Type: "",
				},
				Target: TargetConfig{
					Brokers: []string{"localhost:9092"},
				},
				Mirror: MirrorConfig{
					WorkerCount: 4,
				},
			},
			wantErr: true,
		},
		{
			name: "无效数据源类型",
			config: &Config{
				Source: SourceConfig{
					Type: "invalid",
				},
				Target: TargetConfig{
					Brokers: []string{"localhost:9092"},
				},
				Mirror: MirrorConfig{
					WorkerCount: 4,
				},
			},
			wantErr: true,
		},
		{
			name: "缺少目标brokers",
			config: &Config{
				Source: SourceConfig{
					Type: "kafka",
				},
				Target: TargetConfig{
					Brokers: []string{},
				},
				Mirror: MirrorConfig{
					WorkerCount: 4,
				},
			},
			wantErr: true,
		},
		{
			name: "无效worker数量",
			config: &Config{
				Source: SourceConfig{
					Type: "kafka",
				},
				Target: TargetConfig{
					Brokers: []string{"localhost:9092"},
				},
				Mirror: MirrorConfig{
					WorkerCount: 0,
				},
			},
			wantErr: true,
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

func TestGetConfigPath(t *testing.T) {
	// 测试默认路径
	path := GetConfigPath()
	if path != "config.yaml" {
		t.Errorf("期望默认路径'config.yaml'，实际'%s'", path)
	}
	
	// 测试环境变量
	os.Setenv("CONFIG_PATH", "/custom/path/config.yaml")
	defer os.Unsetenv("CONFIG_PATH")
	
	path = GetConfigPath()
	if path != "/custom/path/config.yaml" {
		t.Errorf("期望环境变量路径'/custom/path/config.yaml'，实际'%s'", path)
	}
}

func TestSetDefaults(t *testing.T) {
	// 测试默认值设置
	// 这个函数会设置viper的默认值
	// 我们通过LoadConfig来验证默认值是否正确
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	configContent := `
source:
  type: "kafka"
  brokers:
    - "localhost:9092"
  topic: "test-topic"

target:
  brokers:
    - "localhost:9093"
`
	tmpFile.WriteString(configContent)
	tmpFile.Close()
	
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	
	// 验证默认值
	if config.Mirror.WorkerCount != 4 {
		t.Errorf("期望默认worker_count=4，实际%d", config.Mirror.WorkerCount)
	}
	
	if config.Log.StatsInterval != 10*time.Second {
		t.Errorf("期望默认stats_interval=10s，实际%v", config.Log.StatsInterval)
	}
	
	if config.Server.Address != ":8080" {
		t.Errorf("期望默认server.address=':8080'，实际'%s'", config.Server.Address)
	}
}

