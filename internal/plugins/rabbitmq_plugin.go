package plugins

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

// RabbitMQPlugin RabbitMQ数据源插件
type RabbitMQPlugin struct {
	config     *RabbitMQPluginConfig
	conn       *amqp.Connection
	channel    *amqp.Channel
	msgChan    chan *Message
	stats      *RabbitMQPluginStats
	statsMu    sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// RabbitMQPluginConfig RabbitMQ插件配置
type RabbitMQPluginConfig struct {
	URL      string
	Exchange string
	Queue    string
	QueueOptions *RabbitMQQueueOptions
}

// RabbitMQQueueOptions 队列选项
type RabbitMQQueueOptions struct {
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
}

// RabbitMQPluginStats RabbitMQ插件统计信息
type RabbitMQPluginStats struct {
	MessagesReceived int64
	MessagesAcked    int64
	Errors           int64
	LastMessageTime  time.Time
	StartTime        time.Time
}

// NewRabbitMQPlugin 创建RabbitMQ插件实例
func NewRabbitMQPlugin() SourcePlugin {
	return &RabbitMQPlugin{
		stats: &RabbitMQPluginStats{
			StartTime: time.Now(),
		},
	}
}

// Name 返回插件名称
func (p *RabbitMQPlugin) Name() string {
	return "rabbitmq"
}

// Initialize 初始化插件
func (p *RabbitMQPlugin) Initialize(config map[string]interface{}) error {
	cfg := &RabbitMQPluginConfig{}

	if url, ok := config["url"].(string); ok {
		cfg.URL = url
	} else {
		cfg.URL = "amqp://guest:guest@localhost:5672/"
	}

	if exchange, ok := config["exchange"].(string); ok {
		cfg.Exchange = exchange
	}

	if queue, ok := config["queue"].(string); ok {
		cfg.Queue = queue
	} else {
		return fmt.Errorf("queue配置不能为空")
	}

	// 队列选项
	cfg.QueueOptions = &RabbitMQQueueOptions{
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
	}

	if queueOpts, ok := config["queue_options"].(map[string]interface{}); ok {
		if durable, ok := queueOpts["durable"].(bool); ok {
			cfg.QueueOptions.Durable = durable
		}
		if autoDelete, ok := queueOpts["auto_delete"].(bool); ok {
			cfg.QueueOptions.AutoDelete = autoDelete
		}
		if exclusive, ok := queueOpts["exclusive"].(bool); ok {
			cfg.QueueOptions.Exclusive = exclusive
		}
	}

	p.config = cfg
	p.msgChan = make(chan *Message, 100)

	return nil
}

// Start 启动插件
func (p *RabbitMQPlugin) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// 连接RabbitMQ
	var err error
	p.conn, err = amqp.Dial(p.config.URL)
	if err != nil {
		return fmt.Errorf("连接RabbitMQ失败: %w", err)
	}

	p.channel, err = p.conn.Channel()
	if err != nil {
		p.conn.Close()
		return fmt.Errorf("创建RabbitMQ通道失败: %w", err)
	}

	// 声明队列
	_, err = p.channel.QueueDeclare(
		p.config.Queue,
		p.config.QueueOptions.Durable,
		p.config.QueueOptions.AutoDelete,
		p.config.QueueOptions.Exclusive,
		p.config.QueueOptions.NoWait,
		nil,
	)
	if err != nil {
		p.channel.Close()
		p.conn.Close()
		return fmt.Errorf("声明队列失败: %w", err)
	}

	// 如果有exchange，绑定队列
	if p.config.Exchange != "" {
		err = p.channel.QueueBind(
			p.config.Queue,
			"", // routing key
			p.config.Exchange,
			false,
			nil,
		)
		if err != nil {
			p.channel.Close()
			p.conn.Close()
			return fmt.Errorf("绑定队列到exchange失败: %w", err)
		}
	}

	// 设置QoS
	err = p.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		log.Printf("[RabbitMQ插件] 设置QoS失败: %v", err)
	}

	log.Printf("[RabbitMQ插件] 开始消费队列: %s", p.config.Queue)

	// 开始消费消息
	deliveries, err := p.channel.Consume(
		p.config.Queue,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		p.channel.Close()
		p.conn.Close()
		return fmt.Errorf("开始消费失败: %w", err)
	}

	// 处理消息
	go func() {
		defer p.cancel()
		for {
			select {
			case <-p.ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					log.Printf("[RabbitMQ插件] 消息通道已关闭")
					return
				}

				// 转换为统一消息格式
				msg := &Message{
					Key:       []byte(delivery.MessageId),
					Value:     delivery.Body,
					Headers:   convertRabbitMQHeaders(delivery.Headers),
					Timestamp: delivery.Timestamp,
					Source:    "rabbitmq",
					Metadata: map[string]interface{}{
						"exchange":     delivery.Exchange,
						"routing_key":  delivery.RoutingKey,
						"delivery_tag": delivery.DeliveryTag,
						"delivery":     delivery, // 保存原始delivery用于ack
					},
				}

				// 更新统计
				p.statsMu.Lock()
				p.stats.MessagesReceived++
				p.stats.LastMessageTime = time.Now()
				p.statsMu.Unlock()

				// 发送消息
				select {
				case p.msgChan <- msg:
				case <-p.ctx.Done():
					return
				}
			}
		}
	}()

	return nil
}

// Stop 停止插件
func (p *RabbitMQPlugin) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	if p.msgChan != nil {
		close(p.msgChan)
	}
	log.Println("[RabbitMQ插件] 已停止")
	return nil
}

// Messages 返回消息通道
func (p *RabbitMQPlugin) Messages() <-chan *Message {
	return p.msgChan
}

// Ack 确认消息
func (p *RabbitMQPlugin) Ack(msg *Message) error {
	if delivery, ok := msg.Metadata["delivery"].(amqp.Delivery); ok {
		err := delivery.Ack(false)
		if err != nil {
			p.statsMu.Lock()
			p.stats.Errors++
			p.statsMu.Unlock()
			return fmt.Errorf("确认消息失败: %w", err)
		}

		p.statsMu.Lock()
		p.stats.MessagesAcked++
		p.statsMu.Unlock()
	}
	return nil
}

// GetStats 获取统计信息
func (p *RabbitMQPlugin) GetStats() PluginStats {
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

// convertRabbitMQHeaders 转换RabbitMQ消息头
func convertRabbitMQHeaders(headers amqp.Table) map[string][]byte {
	if len(headers) == 0 {
		return nil
	}

	result := make(map[string][]byte)
	for k, v := range headers {
		switch val := v.(type) {
		case string:
			result[k] = []byte(val)
		case []byte:
			result[k] = val
		case []interface{}:
			// 处理数组类型的header
			bytes := make([]byte, 0)
			for _, item := range val {
				if b, ok := item.([]byte); ok {
					bytes = append(bytes, b...)
				}
			}
			result[k] = bytes
		}
	}
	return result
}

func init() {
	RegisterPlugin("rabbitmq", NewRabbitMQPlugin)
}
