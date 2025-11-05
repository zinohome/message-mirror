package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

// KafkaPlugin Kafka数据源插件
type KafkaPlugin struct {
	config     *KafkaPluginConfig
	consumer   sarama.ConsumerGroup
	msgChan    chan *Message
	handler    *kafkaConsumerGroupHandler
	stats      *KafkaPluginStats
	statsMu    sync.RWMutex
}

// KafkaPluginConfig Kafka插件配置
type KafkaPluginConfig struct {
	Brokers          []string
	Topic            string
	Topics           []string
	GroupID          string
	AutoOffsetReset  string
	SessionTimeout   time.Duration
	BatchSize        int
	SecurityProtocol string
	SASLUsername     string
	SASLPassword     string
}

// KafkaPluginStats Kafka插件统计信息
type KafkaPluginStats struct {
	MessagesReceived int64
	MessagesAcked    int64
	Errors           int64
	LastMessageTime  time.Time
	StartTime        time.Time
}

// kafkaConsumerGroupHandler Kafka消费者组处理器
type kafkaConsumerGroupHandler struct {
	msgChan chan *Message
	stats   *KafkaPluginStats
	statsMu *sync.RWMutex
}

func (h *kafkaConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *kafkaConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *kafkaConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// 转换为统一消息格式
			msg := &Message{
				Key:       message.Key,
				Value:     message.Value,
				Headers:   convertKafkaHeaders(message.Headers),
				Timestamp: message.Timestamp,
				Source:    "kafka",
				Metadata: map[string]interface{}{
					"topic":     message.Topic,
					"partition": message.Partition,
					"offset":    message.Offset,
				},
			}

			// 更新统计
			h.statsMu.Lock()
			h.stats.MessagesReceived++
			h.stats.LastMessageTime = time.Now()
			h.statsMu.Unlock()

			// 发送消息
			select {
			case h.msgChan <- msg:
				session.MarkMessage(message, "")
			case <-session.Context().Done():
				return nil
			}

		case <-session.Context().Done():
			return nil
		}
	}
}

// convertKafkaHeaders 转换Kafka消息头
func convertKafkaHeaders(headers []*sarama.RecordHeader) map[string][]byte {
	if len(headers) == 0 {
		return nil
	}

	result := make(map[string][]byte, len(headers))
	for _, h := range headers {
		result[string(h.Key)] = h.Value
	}
	return result
}

// NewKafkaPlugin 创建Kafka插件实例
func NewKafkaPlugin() SourcePlugin {
	return &KafkaPlugin{
		stats: &KafkaPluginStats{
			StartTime: time.Now(),
		},
	}
}

// Name 返回插件名称
func (p *KafkaPlugin) Name() string {
	return "kafka"
}

// Initialize 初始化插件
func (p *KafkaPlugin) Initialize(config map[string]interface{}) error {
	cfg := &KafkaPluginConfig{}

	// 解析配置
	if brokers, ok := config["brokers"].([]interface{}); ok {
		cfg.Brokers = make([]string, len(brokers))
		for i, b := range brokers {
			if s, ok := b.(string); ok {
				cfg.Brokers[i] = s
			}
		}
	}

	if topic, ok := config["topic"].(string); ok {
		cfg.Topic = topic
	}

	if topics, ok := config["topics"].([]interface{}); ok {
		cfg.Topics = make([]string, 0, len(topics))
		for _, t := range topics {
			if s, ok := t.(string); ok {
				cfg.Topics = append(cfg.Topics, s)
			}
		}
	}

	if groupID, ok := config["group_id"].(string); ok {
		cfg.GroupID = groupID
	} else {
		cfg.GroupID = "message-mirror-group"
	}

	if autoOffsetReset, ok := config["auto_offset_reset"].(string); ok {
		cfg.AutoOffsetReset = autoOffsetReset
	} else {
		cfg.AutoOffsetReset = "earliest"
	}

	if sessionTimeout, ok := config["session_timeout"].(time.Duration); ok {
		cfg.SessionTimeout = sessionTimeout
	} else {
		cfg.SessionTimeout = 30 * time.Second
	}

	if batchSize, ok := config["batch_size"].(int); ok {
		cfg.BatchSize = batchSize
	} else {
		cfg.BatchSize = 100
	}

	if securityProtocol, ok := config["security_protocol"].(string); ok {
		cfg.SecurityProtocol = securityProtocol
	}

	if username, ok := config["sasl_username"].(string); ok {
		cfg.SASLUsername = username
	}

	if password, ok := config["sasl_password"].(string); ok {
		cfg.SASLPassword = password
	}

	p.config = cfg

	// 创建Sarama配置
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_0_0
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()

	if cfg.AutoOffsetReset == "latest" {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	saramaConfig.Consumer.Group.Session.Timeout = cfg.SessionTimeout
	saramaConfig.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
	saramaConfig.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second

	// TLS配置（从source配置中读取）
	if tlsConfig, ok := config["tls"].(map[string]interface{}); ok {
		if enabled, ok := tlsConfig["enabled"].(bool); ok && enabled {
			tlsCfg := &TLSConfig{
				Enabled:            enabled,
				InsecureSkipVerify: false,
			}
			
			if certFile, ok := tlsConfig["cert_file"].(string); ok {
				tlsCfg.CertFile = certFile
			}
			if keyFile, ok := tlsConfig["key_file"].(string); ok {
				tlsCfg.KeyFile = keyFile
			}
			if caFile, ok := tlsConfig["ca_file"].(string); ok {
				tlsCfg.CAFile = caFile
			}
			if skipVerify, ok := tlsConfig["insecure_skip_verify"].(bool); ok {
				tlsCfg.InsecureSkipVerify = skipVerify
			}
			if minVersion, ok := tlsConfig["min_version"].(string); ok {
				tlsCfg.MinVersion = minVersion
			}
			if maxVersion, ok := tlsConfig["max_version"].(string); ok {
				tlsCfg.MaxVersion = maxVersion
			}
			
			tlsConfigObj, err := NewTLSConfig(tlsCfg)
			if err != nil {
				return fmt.Errorf("创建TLS配置失败: %w", err)
			}
			if tlsConfigObj != nil {
				saramaConfig.Net.TLS.Enable = true
				saramaConfig.Net.TLS.Config = tlsConfigObj
			}
		}
	}
	
	// SASL配置
	if cfg.SecurityProtocol == "SASL_PLAINTEXT" || cfg.SecurityProtocol == "SASL_SSL" {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		saramaConfig.Net.SASL.User = cfg.SASLUsername
		saramaConfig.Net.SASL.Password = cfg.SASLPassword
	}

	// 创建消费者组
	consumer, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, saramaConfig)
	if err != nil {
		return fmt.Errorf("创建Kafka消费者组失败: %w", err)
	}

	p.consumer = consumer
	p.msgChan = make(chan *Message, cfg.BatchSize*2)
	p.handler = &kafkaConsumerGroupHandler{
		msgChan: p.msgChan,
		stats:   p.stats,
		statsMu: &p.statsMu,
	}

	return nil
}

// Start 启动插件
func (p *KafkaPlugin) Start(ctx context.Context) error {
	topics := p.getTopics()
	if len(topics) == 0 {
		return fmt.Errorf("没有配置要消费的topics")
	}

	log.Printf("[Kafka插件] 开始消费topics: %v", topics)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := p.consumer.Consume(ctx, topics, p.handler); err != nil {
					p.statsMu.Lock()
					p.stats.Errors++
					p.statsMu.Unlock()
					log.Printf("[Kafka插件] 消费错误: %v", err)
					time.Sleep(1 * time.Second)
				}
			}
		}
	}()

	return nil
}

// Stop 停止插件
func (p *KafkaPlugin) Stop() error {
	if p.consumer != nil {
		if err := p.consumer.Close(); err != nil {
			return err
		}
	}
	if p.msgChan != nil {
		close(p.msgChan)
	}
	return nil
}

// Messages 返回消息通道
func (p *KafkaPlugin) Messages() <-chan *Message {
	return p.msgChan
}

// Ack 确认消息（Kafka通过自动提交offset实现）
func (p *KafkaPlugin) Ack(msg *Message) error {
	p.statsMu.Lock()
	p.stats.MessagesAcked++
	p.statsMu.Unlock()
	return nil
}

// GetStats 获取统计信息
func (p *KafkaPlugin) GetStats() PluginStats {
	p.statsMu.RLock()
	defer p.statsMu.RUnlock()

	return PluginStats{
		MessagesReceived: p.stats.MessagesReceived,
		MessagesAcked:    p.stats.MessagesAcked,
		Errors:           p.stats.Errors,
		LastMessageTime:  p.stats.LastMessageTime,
		StartTime:        p.stats.StartTime,
	}
}

// getTopics 获取要消费的topics列表
func (p *KafkaPlugin) getTopics() []string {
	if p.config.Topic != "" {
		return []string{p.config.Topic}
	}
	if len(p.config.Topics) > 0 {
		return p.config.Topics
	}
	return []string{}
}

func init() {
	RegisterPlugin("kafka", NewKafkaPlugin)
}
