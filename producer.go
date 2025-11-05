package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/IBM/sarama"
)

// MirrorProducer 镜像生产者
type MirrorProducer struct {
	config   *Config
	producer sarama.AsyncProducer
	ctx      context.Context
	wg       sync.WaitGroup
}

// NewMirrorProducer 创建新的镜像生产者
func NewMirrorProducer(config *Config) (*MirrorProducer, error) {
	// 创建Sarama配置
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_0_0

	// 生产者配置
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = sarama.RequiredAcks(config.Producer.RequiredAcks)
	saramaConfig.Producer.Retry.Max = config.Producer.RetryMax
	saramaConfig.Producer.Idempotent = config.Producer.Idempotent
	saramaConfig.Producer.MaxMessageBytes = config.Producer.MaxMessageBytes
	saramaConfig.Producer.Flush.Frequency = config.Producer.FlushFrequency

	// 压缩类型
	switch config.Producer.CompressionType {
	case "gzip":
		saramaConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		saramaConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		saramaConfig.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		saramaConfig.Producer.Compression = sarama.CompressionZSTD
	default:
		saramaConfig.Producer.Compression = sarama.CompressionNone
	}

	// TLS配置
	if config.Security.TLS.Enabled {
		tlsConfig, err := NewTLSConfig(&config.Security.TLS)
		if err != nil {
			return nil, fmt.Errorf("创建TLS配置失败: %w", err)
		}
		if tlsConfig != nil {
			saramaConfig.Net.TLS.Enable = true
			saramaConfig.Net.TLS.Config = tlsConfig
		}
	}

	// SASL配置
	if config.Target.SecurityProtocol == "SASL_PLAINTEXT" || config.Target.SecurityProtocol == "SASL_SSL" {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		saramaConfig.Net.SASL.User = config.Target.SASLUsername
		saramaConfig.Net.SASL.Password = config.Target.SASLPassword
	}

	// 如果启用幂等性，需要设置事务ID
	if config.Producer.Idempotent {
		saramaConfig.Producer.Transaction.ID = "mirror-maker-producer"
	}

	// 创建异步生产者
	producer, err := sarama.NewAsyncProducer(config.Target.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("创建生产者失败: %w", err)
	}

	return &MirrorProducer{
		config:   config,
		producer: producer,
	}, nil
}

// Start 启动生产者
func (mp *MirrorProducer) Start(ctx context.Context) error {
	mp.ctx = ctx

	// 启动成功消息处理
	mp.wg.Add(1)
	go mp.handleSuccesses()

	// 启动错误消息处理
	mp.wg.Add(1)
	go mp.handleErrors()

	log.Println("生产者已启动")
	return nil
}

// Stop 停止生产者
func (mp *MirrorProducer) Stop() error {
	log.Println("正在停止生产者...")

	// 关闭生产者
	if mp.producer != nil {
		if err := mp.producer.Close(); err != nil {
			return fmt.Errorf("关闭生产者失败: %w", err)
		}
	}

	// 等待所有goroutine完成
	mp.wg.Wait()

	log.Println("生产者已停止")
	return nil
}

// Send 发送消息
func (mp *MirrorProducer) Send(message *sarama.ProducerMessage) error {
	select {
	case <-mp.ctx.Done():
		return mp.ctx.Err()
	case mp.producer.Input() <- message:
		return nil
	}
}

// handleSuccesses 处理成功发送的消息
func (mp *MirrorProducer) handleSuccesses() {
	defer mp.wg.Done()

	for {
		select {
		case <-mp.ctx.Done():
			return
		case msg := <-mp.producer.Successes():
			if msg != nil {
				// 可以在这里记录成功的消息
				// log.Printf("消息发送成功: topic=%s, partition=%d, offset=%d",
				//     msg.Topic, msg.Partition, msg.Offset)
			}
		}
	}
}

// handleErrors 处理发送失败的消息
func (mp *MirrorProducer) handleErrors() {
	defer mp.wg.Done()

	for {
		select {
		case <-mp.ctx.Done():
			return
		case err := <-mp.producer.Errors():
			if err != nil {
				log.Printf("消息发送失败: topic=%s, partition=%d, error=%v",
					err.Msg.Topic, err.Msg.Partition, err.Err)
			}
		}
	}
}
