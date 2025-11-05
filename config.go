package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config 应用程序配置
type Config struct {
	Source      SourceConfig `mapstructure:"source"`
	Target      TargetConfig `mapstructure:"target"`
	Mirror      MirrorConfig `mapstructure:"mirror"`
	Producer    ProducerConfig `mapstructure:"producer"`
	Log         LogConfig `mapstructure:"log"`
	Server      ServerConfig `mapstructure:"server"`
	Retry       RetryConfig `mapstructure:"retry"`
	Dedup       DedupConfig `mapstructure:"dedup"`
	Security    SecurityConfig `mapstructure:"security"`
}

// SourceConfig 数据源配置（支持插件）
type SourceConfig struct {
	Type   string                 `mapstructure:"type"` // kafka, rabbitmq, file
	Config map[string]interface{} `mapstructure:",remain"` // 插件特定的配置
}

// TargetConfig 目标配置（目前只支持Kafka）
type TargetConfig struct {
	Brokers          []string `mapstructure:"brokers"`
	Topic            string   `mapstructure:"topic"`
	SecurityProtocol string   `mapstructure:"security_protocol"`
	SASLMechanism    string   `mapstructure:"sasl_mechanism"`
	SASLUsername     string   `mapstructure:"sasl_username"`
	SASLPassword     string   `mapstructure:"sasl_password"`
}

// KafkaConfig Kafka集群配置（保留用于向后兼容）
type KafkaConfig struct {
	Brokers          []string `mapstructure:"brokers"`
	Topic            string   `mapstructure:"topic"`
	Topics           []string `mapstructure:"topics"`
	SecurityProtocol string   `mapstructure:"security_protocol"`
	SASLMechanism    string   `mapstructure:"sasl_mechanism"`
	SASLUsername     string   `mapstructure:"sasl_username"`
	SASLPassword      string   `mapstructure:"sasl_password"`
}

// ConsumerConfig 消费者配置（保留用于向后兼容）
type ConsumerConfig struct {
	GroupID           string        `mapstructure:"group_id"`
	AutoOffsetReset   string        `mapstructure:"auto_offset_reset"`
	SessionTimeout    time.Duration `mapstructure:"session_timeout"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	MaxProcessingTime time.Duration `mapstructure:"max_processing_time"`
	BatchSize         int           `mapstructure:"batch_size"`
}

// MirrorConfig 镜像配置
type MirrorConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	TopicPattern      string   `mapstructure:"topic_pattern"`
	ExcludeTopics     []string `mapstructure:"exclude_topics"`
	PreservePartition bool     `mapstructure:"preserve_partition"`
	OffsetSync        bool     `mapstructure:"offset_sync"`
	WorkerCount       int      `mapstructure:"worker_count"`
	// 流控配置
	ConsumerRateLimit  float64 `mapstructure:"consumer_rate_limit"`  // 消费速率限制（消息/秒，0表示不限制）
	ConsumerBurstSize  int     `mapstructure:"consumer_burst_size"`  // 消费突发大小
	ProducerRateLimit  float64 `mapstructure:"producer_rate_limit"`  // 生产速率限制（消息/秒，0表示不限制）
	ProducerBurstSize  int     `mapstructure:"producer_burst_size"`  // 生产突发大小
	BytesRateLimit     float64 `mapstructure:"bytes_rate_limit"`    // 字节速率限制（字节/秒，0表示不限制，优先于消息速率限制）
	BytesBurstSize     int     `mapstructure:"bytes_burst_size"`     // 字节突发大小
	// 批处理配置
	BatchEnabled       bool          `mapstructure:"batch_enabled"`        // 是否启用批处理
	BatchSize          int           `mapstructure:"batch_size"`           // 批处理大小（消息数）
	BatchTimeout       time.Duration `mapstructure:"batch_timeout"`        // 批处理超时时间
}

// ProducerConfig 生产者配置
type ProducerConfig struct {
	MaxMessageBytes    int           `mapstructure:"max_message_bytes"`
	CompressionType    string        `mapstructure:"compression_type"`
	RequiredAcks       int16         `mapstructure:"required_acks"`
	FlushFrequency    time.Duration `mapstructure:"flush_frequency"`
	RetryMax           int           `mapstructure:"retry_max"`
	Idempotent         bool          `mapstructure:"idempotent"`
}

// LogConfig 日志配置
type LogConfig struct {
	FilePath            string        `mapstructure:"file_path"`
	StatsInterval      time.Duration `mapstructure:"stats_interval"`
	RotateInterval     time.Duration `mapstructure:"rotate_interval"`     // 日志轮转间隔
	MaxArchiveFiles    int           `mapstructure:"max_archive_files"` // 保留的归档文件数量
	AsyncBufferSize    int           `mapstructure:"async_buffer_size"` // 异步缓冲区大小
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Address string `mapstructure:"address"` // HTTP服务器地址
}

// RetryConfig 重试配置
type RetryConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	MaxRetries      int           `mapstructure:"max_retries"`
	InitialInterval time.Duration `mapstructure:"initial_interval"`
	MaxInterval     time.Duration `mapstructure:"max_interval"`
	Multiplier      float64       `mapstructure:"multiplier"`
	Jitter          bool          `mapstructure:"jitter"`
}

// DedupConfig 去重配置
type DedupConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	Strategy       string        `mapstructure:"strategy"`        // 去重策略：key, value, key_value, hash
	TTL            time.Duration `mapstructure:"ttl"`             // 去重记录的TTL
	MaxEntries     int64         `mapstructure:"max_entries"`    // 最大去重记录数
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"` // 清理过期记录的间隔
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	TLS            TLSConfig `mapstructure:"tls"`
	AuditEnabled   bool      `mapstructure:"audit_enabled"`
	ConfigEncryption bool    `mapstructure:"config_encryption"`
	EncryptionKey  string    `mapstructure:"encryption_key"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaults()

	// 环境变量支持
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("配置文件未找到: %s", configPath)
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 将source配置转换为map[string]interface{}格式供插件使用
	if config.Source.Config == nil {
		config.Source.Config = make(map[string]interface{})
		// 从viper中提取source配置（排除type字段）
		sourceConfig := viper.GetStringMap("source")
		for k, v := range sourceConfig {
			if k != "type" {
				config.Source.Config[k] = v
			}
		}
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// Mirror配置默认值
	viper.SetDefault("mirror.enabled", true)
	viper.SetDefault("mirror.preserve_partition", true)
	viper.SetDefault("mirror.offset_sync", false)
	viper.SetDefault("mirror.worker_count", 4)
	// 流控默认值（0表示不限制）
	viper.SetDefault("mirror.consumer_rate_limit", 0.0)
	viper.SetDefault("mirror.consumer_burst_size", 100)
	viper.SetDefault("mirror.producer_rate_limit", 0.0)
	viper.SetDefault("mirror.producer_burst_size", 100)
	viper.SetDefault("mirror.bytes_rate_limit", 0.0)
	// 批处理默认值
	viper.SetDefault("mirror.batch_enabled", false)
	viper.SetDefault("mirror.batch_size", 100)
	viper.SetDefault("mirror.batch_timeout", 100*time.Millisecond)
	viper.SetDefault("mirror.bytes_burst_size", 10485760) // 10MB

	// Consumer配置默认值
	viper.SetDefault("consumer.group_id", "message-mirror-group")
	viper.SetDefault("consumer.auto_offset_reset", "earliest")
	viper.SetDefault("consumer.session_timeout", 30*time.Second)
	viper.SetDefault("consumer.heartbeat_interval", 3*time.Second)
	viper.SetDefault("consumer.max_processing_time", 5*time.Minute)
	viper.SetDefault("consumer.batch_size", 100)

	// Producer配置默认值
	viper.SetDefault("producer.max_message_bytes", 1000000)
	viper.SetDefault("producer.compression_type", "snappy")
	viper.SetDefault("producer.required_acks", 1)
	viper.SetDefault("producer.flush_frequency", 100*time.Millisecond)
	viper.SetDefault("producer.retry_max", 3)
	viper.SetDefault("producer.idempotent", true)

	// Log配置默认值
	viper.SetDefault("log.file_path", "message-mirror.log")
	viper.SetDefault("log.stats_interval", 10*time.Second)
	viper.SetDefault("log.rotate_interval", 24*time.Hour)
	viper.SetDefault("log.max_archive_files", 7)
	viper.SetDefault("log.async_buffer_size", 1000)

	// Server配置默认值
	viper.SetDefault("server.enabled", true)
	viper.SetDefault("server.address", ":8080")

	// Retry配置默认值
	viper.SetDefault("retry.enabled", true)
	viper.SetDefault("retry.max_retries", 3)
	viper.SetDefault("retry.initial_interval", 100*time.Millisecond)
	viper.SetDefault("retry.max_interval", 10*time.Second)
	viper.SetDefault("retry.multiplier", 2.0)
	viper.SetDefault("retry.jitter", true)

	// Dedup配置默认值
	viper.SetDefault("dedup.enabled", false)
	viper.SetDefault("dedup.strategy", "key_value")
	viper.SetDefault("dedup.ttl", 24*time.Hour)
	viper.SetDefault("dedup.max_entries", 1000000)
	viper.SetDefault("dedup.cleanup_interval", 1*time.Hour)

	// Security配置默认值
	viper.SetDefault("security.tls.enabled", false)
	viper.SetDefault("security.tls.insecure_skip_verify", false)
	viper.SetDefault("security.tls.min_version", "1.2")
	viper.SetDefault("security.audit_enabled", false)
	viper.SetDefault("security.config_encryption", false)
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证数据源类型
	if config.Source.Type == "" {
		return fmt.Errorf("数据源类型不能为空，支持的类型: kafka, rabbitmq, file")
	}

	validTypes := map[string]bool{
		"kafka":    true,
		"rabbitmq": true,
		"file":     true,
	}
	if !validTypes[config.Source.Type] {
		return fmt.Errorf("不支持的数据源类型: %s，支持的类型: kafka, rabbitmq, file", config.Source.Type)
	}

	// 验证目标集群
	if len(config.Target.Brokers) == 0 {
		return fmt.Errorf("目标集群brokers不能为空")
	}

	if config.Mirror.WorkerCount <= 0 {
		return fmt.Errorf("worker数量必须大于0")
	}

	return nil
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return "config.yaml"
}
