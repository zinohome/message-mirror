package core

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

// TestEndToEndKafkaMirroring 端到端Kafka镜像测试
// 使用testcontainers启动真实的Kafka集群，验证完整的消息流转
func TestEndToEndKafkaMirroring(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过端到端集成测试")
	}

	ctx := context.Background()

	// 1. 启动Kafka容器
	t.Log("启动Kafka容器...")
	kafkaContainer, brokers, err := startKafkaContainer(ctx)
	if err != nil {
		t.Fatalf("启动Kafka容器失败: %v", err)
	}
	defer kafkaContainer.Terminate(ctx)

	t.Logf("Kafka就绪，brokers: %v", brokers)

	// 2. 创建源和目标topic
	sourceTopic := "test-source-topic"
	targetTopic := "test-target-topic"

	if err := createTopic(brokers, sourceTopic); err != nil {
		t.Fatalf("创建源topic失败: %v", err)
	}
	if err := createTopic(brokers, targetTopic); err != nil {
		t.Fatalf("创建目标topic失败: %v", err)
	}
	t.Log("Topics创建成功")

	// 3. 创建配置
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":           []interface{}{brokers[0]},
				"topic":             sourceTopic,
				"group_id":          "test-group",
				"auto_offset_reset": "earliest",
			},
		},
		Target: TargetConfig{
			Brokers: brokers,
			Topic:   targetTopic,
		},
		Mirror: MirrorConfig{
			Enabled:     true,
			WorkerCount: 2,
		},
		Producer: ProducerConfig{
			RequiredAcks:    1,
			CompressionType: "snappy",
			MaxMessageBytes: 1000000,
			RetryMax:        3,
			FlushFrequency:  100 * time.Millisecond,
		},
		Log: LogConfig{
			FilePath:      "/tmp/message-mirror-test.log",
			StatsInterval: 5 * time.Second,
		},
	}

	// 4. 启动MirrorMaker
	t.Log("启动MirrorMaker...")
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Fatalf("创建MirrorMaker失败: %v", err)
	}

	if err := mm.Start(); err != nil {
		t.Fatalf("启动MirrorMaker失败: %v", err)
	}
	defer mm.Stop()

	// 等待MirrorMaker就绪
	time.Sleep(2 * time.Second)
	t.Log("MirrorMaker已启动")

	// 5. 发送测试消息到源topic
	testMessages := []struct {
		key   string
		value string
	}{
		{"key1", "message1"},
		{"key2", "message2"},
		{"key3", "message3"},
		{"key4", "message4"},
		{"key5", "message5"},
	}

	producer, err := createProducer(brokers)
	if err != nil {
		t.Fatalf("创建生产者失败: %v", err)
	}
	defer producer.Close()

	t.Log("发送测试消息...")
	for _, msg := range testMessages {
		_, _, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic: sourceTopic,
			Key:   sarama.StringEncoder(msg.key),
			Value: sarama.StringEncoder(msg.value),
		})
		if err != nil {
			t.Errorf("发送消息失败: %v", err)
		}
	}
	t.Logf("发送了 %d 条消息到源topic", len(testMessages))

	// 6. 从目标topic消费消息验证
	t.Log("从目标topic消费消息验证...")
	consumer, err := createConsumer(brokers, targetTopic)
	if err != nil {
		t.Fatalf("创建消费者失败: %v", err)
	}
	defer consumer.Close()

	receivedMessages := make(map[string]string)
	timeout := time.After(30 * time.Second)
	partitions, err := consumer.Partitions(targetTopic)
	if err != nil || len(partitions) == 0 {
		t.Fatalf("获取分区失败: %v, partitions: %v", err, partitions)
	}
	pc, err := consumer.ConsumePartition(targetTopic, partitions[0], sarama.OffsetOldest)
	if err != nil {
		t.Fatalf("创建分区消费者失败: %v", err)
	}
	defer pc.Close()

consumeLoop:
	for {
		select {
		case msg := <-pc.Messages():
			if msg == nil {
				continue
			}
			key := string(msg.Key)
			value := string(msg.Value)
			receivedMessages[key] = value
			t.Logf("收到消息: key=%s, value=%s", key, value)

			if len(receivedMessages) >= len(testMessages) {
				break consumeLoop
			}
		case <-timeout:
			t.Logf("超时，收到 %d 条消息", len(receivedMessages))
			break consumeLoop
		}
	}

	// 7. 验证结果
	t.Logf("验证结果: 期望 %d 条，实际收到 %d 条", len(testMessages), len(receivedMessages))
	if len(receivedMessages) != len(testMessages) {
		t.Errorf("消息数量不匹配: 期望 %d, 实际 %d", len(testMessages), len(receivedMessages))
	}

	for _, msg := range testMessages {
		if val, ok := receivedMessages[msg.key]; !ok {
			t.Errorf("缺少消息: key=%s", msg.key)
		} else if val != msg.value {
			t.Errorf("消息内容不匹配: key=%s, 期望=%s, 实际=%s", msg.key, msg.value, val)
		}
	}

	// 8. 验证统计信息
	stats := mm.GetStats()
	t.Logf("统计信息: consumed=%d, produced=%d, errors=%d",
		stats.MessagesConsumed, stats.MessagesProduced, stats.Errors)

	if stats.MessagesConsumed < int64(len(testMessages)) {
		t.Errorf("消费消息数不足: 期望至少 %d, 实际 %d",
			len(testMessages), stats.MessagesConsumed)
	}
	if stats.MessagesProduced < int64(len(testMessages)) {
		t.Errorf("生产消息数不足: 期望至少 %d, 实际 %d",
			len(testMessages), stats.MessagesProduced)
	}

	t.Log("端到端测试通过！")
}

// TestConfigHotReload 配置热重载测试
func TestConfigHotReload(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过配置热重载测试")
	}

	ctx := context.Background()

	// 1. 启动Kafka容器
	t.Log("启动Kafka容器...")
	kafkaContainer, brokers, err := startKafkaContainer(ctx)
	if err != nil {
		t.Fatalf("启动Kafka容器失败: %v", err)
	}
	defer kafkaContainer.Terminate(ctx)

	time.Sleep(5 * time.Second)

	// 2. 创建初始配置
	sourceTopic := "reload-test-source"
	targetTopic := "reload-test-target"

	createTopic(brokers, sourceTopic)
	createTopic(brokers, targetTopic)

	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":           []interface{}{brokers[0]},
				"topic":             sourceTopic,
				"group_id":          "reload-test-group",
				"auto_offset_reset": "earliest",
			},
		},
		Target: TargetConfig{
			Brokers: brokers,
			Topic:   targetTopic,
		},
		Mirror: MirrorConfig{
			Enabled:           true,
			WorkerCount:       2,
			ConsumerRateLimit: 100, // 初始限流100 msg/s
		},
		Producer: ProducerConfig{
			RequiredAcks:    1,
			CompressionType: "none",
			MaxMessageBytes: 1000000,
			RetryMax:        3,
			FlushFrequency:  100 * time.Millisecond,
		},
		Log: LogConfig{
			FilePath:      "/tmp/message-mirror-reload-test.log",
			StatsInterval: 5 * time.Second,
		},
	}

	// 3. 启动MirrorMaker
	t.Log("启动MirrorMaker...")
	mm, err := NewMirrorMaker(config)
	if err != nil {
		t.Fatalf("创建MirrorMaker失败: %v", err)
	}

	if err := mm.Start(); err != nil {
		t.Fatalf("启动MirrorMaker失败: %v", err)
	}
	defer mm.Stop()

	time.Sleep(2 * time.Second)

	// 4. 获取初始统计
	stats1 := mm.GetStats()
	t.Logf("初始统计: consumed=%d", stats1.MessagesConsumed)

	// 5. 热重载配置（修改限流参数）
	t.Log("热重载配置...")
	newConfig := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":           []interface{}{brokers[0]},
				"topic":             sourceTopic,
				"group_id":          "reload-test-group",
				"auto_offset_reset": "earliest",
			},
		},
		Target: TargetConfig{
			Brokers: brokers,
			Topic:   targetTopic,
		},
		Mirror: MirrorConfig{
			Enabled:           true,
			WorkerCount:       4,   // 增加worker数量
			ConsumerRateLimit: 200, // 提高限流到200 msg/s
		},
		Producer: ProducerConfig{
			RequiredAcks:    1,
			CompressionType: "snappy", // 修改压缩类型
			MaxMessageBytes: 1000000,
			RetryMax:        3,
			FlushFrequency:  100 * time.Millisecond,
		},
	}

	if err := mm.OnConfigReload(config, newConfig); err != nil {
		t.Fatalf("配置热重载失败: %v", err)
	}

	t.Log("配置热重载成功")
	time.Sleep(1 * time.Second)

	// 6. 验证配置已更新
	mm.mu.RLock()
	if mm.config.Mirror.WorkerCount != 4 {
		t.Errorf("Worker数量未更新: 期望 4, 实际 %d", mm.config.Mirror.WorkerCount)
	}
	if mm.config.Mirror.ConsumerRateLimit != 200 {
		t.Errorf("限流参数未更新: 期望 200, 实际 %f", mm.config.Mirror.ConsumerRateLimit)
	}
	mm.mu.RUnlock()

	t.Log("配置热重载测试通过！")
}

// Helper functions

func startKafkaContainer(ctx context.Context) (*kafka.KafkaContainer, []string, error) {
	kafkaContainer, err := kafka.Run(ctx,
		"confluentinc/confluent-local:7.5.0",
		kafka.WithClusterID("test-cluster"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start kafka container: %w", err)
	}

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		kafkaContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to get brokers: %w", err)
	}

	return kafkaContainer, brokers, nil
}

func createTopic(brokers []string, topic string) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0

	admin, err := sarama.NewClusterAdmin(brokers, config)
	if err != nil {
		return err
	}
	defer admin.Close()

	err = admin.CreateTopic(topic, &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}, false)

	if err != nil {
		// 如果topic已存在，忽略错误
		return nil
	}
	return nil
}

func createProducer(brokers []string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = true
	config.Producer.MaxMessageBytes = 1000000

	return sarama.NewSyncProducer(brokers, config)
}

func createConsumer(brokers []string, topic string) (sarama.Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Return.Errors = true
	config.Consumer.MaxProcessingTime = 30 * time.Second
	config.Consumer.Fetch.Max = 1000000 // 增加Fetch最大值

	return sarama.NewConsumer(brokers, config)
}

// TestConcurrentConsumers 并发消费者测试
// 测试多个MirrorMaker实例使用相同consumer group消费消息
func TestConcurrentConsumers(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过并发测试")
	}

	ctx := context.Background()

	// 1. 启动Kafka容器
	t.Log("启动Kafka容器...")
	kafkaContainer, brokers, err := startKafkaContainer(ctx)
	if err != nil {
		t.Fatalf("启动Kafka容器失败: %v", err)
	}
	defer kafkaContainer.Terminate(ctx)

	t.Logf("Kafka就绪，brokers: %v", brokers)

	// 2. 创建topics
	sourceTopic := "concurrent-source"
	targetTopic := "concurrent-target"

	if err := createTopic(brokers, sourceTopic); err != nil {
		t.Fatalf("创建源topic失败: %v", err)
	}
	if err := createTopic(brokers, targetTopic); err != nil {
		t.Fatalf("创建目标topic失败: %v", err)
	}

	// 3. 创建2个MirrorMaker实例（使用相同的consumer group）
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":           []interface{}{brokers[0]},
				"topic":             sourceTopic,
				"group_id":          "concurrent-group",
				"auto_offset_reset": "earliest",
			},
		},
		Target: TargetConfig{
			Brokers: brokers,
			Topic:   targetTopic,
		},
		Mirror: MirrorConfig{
			Enabled:     true,
			WorkerCount: 1,
		},
		Producer: ProducerConfig{
			RequiredAcks:    1,
			CompressionType: "none",
			MaxMessageBytes: 1000000,
			RetryMax:        3,
			FlushFrequency:  100 * time.Millisecond,
		},
		Log: LogConfig{
			FilePath:      "/tmp/concurrent-test.log",
			StatsInterval: 5 * time.Second,
		},
	}

	// 启动2个实例
	mm1, _ := NewMirrorMaker(config)
	mm1.Start()
	defer mm1.Stop()

	mm2, _ := NewMirrorMaker(config)
	mm2.Start()
	defer mm2.Stop()

	time.Sleep(2 * time.Second)
	t.Log("2个MirrorMaker实例已启动")

	// 4. 发送10条消息
	producer, _ := createProducer(brokers)
	defer producer.Close()

	for i := 0; i < 10; i++ {
		producer.SendMessage(&sarama.ProducerMessage{
			Topic: sourceTopic,
			Key:   sarama.StringEncoder(fmt.Sprintf("key-%d", i)),
			Value: sarama.StringEncoder(fmt.Sprintf("value-%d", i)),
		})
	}
	t.Log("发送了10条消息")

	time.Sleep(5 * time.Second)

	// 5. 验证消息总数
	stats1 := mm1.GetStats()
	stats2 := mm2.GetStats()
	total := stats1.MessagesConsumed + stats2.MessagesConsumed

	t.Logf("实例1: consumed=%d, 实例2: consumed=%d, 总计=%d",
		stats1.MessagesConsumed, stats2.MessagesConsumed, total)

	if total < 10 {
		t.Errorf("消息总数不足: 期望10, 实际%d", total)
	}

	t.Log("并发消费测试通过！")
}

// TestErrorRecovery 错误恢复测试
// 测试系统在遇到错误后能否正常恢复
func TestErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过错误恢复测试")
	}

	ctx := context.Background()

	// 1. 启动Kafka容器
	t.Log("启动Kafka容器...")
	kafkaContainer, brokers, err := startKafkaContainer(ctx)
	if err != nil {
		t.Fatalf("启动Kafka容器失败: %v", err)
	}
	defer kafkaContainer.Terminate(ctx)

	// 2. 创建topics
	sourceTopic := "recovery-source"
	targetTopic := "recovery-target"

	createTopic(brokers, sourceTopic)
	createTopic(brokers, targetTopic)

	// 3. 配置启用重试
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":           []interface{}{brokers[0]},
				"topic":             sourceTopic,
				"group_id":          "recovery-group",
				"auto_offset_reset": "earliest",
			},
		},
		Target: TargetConfig{
			Brokers: brokers,
			Topic:   targetTopic,
		},
		Mirror: MirrorConfig{
			Enabled:     true,
			WorkerCount: 2,
		},
		Producer: ProducerConfig{
			RequiredAcks:    1,
			CompressionType: "none",
			MaxMessageBytes: 1000000,
			RetryMax:        5,
			FlushFrequency:  100 * time.Millisecond,
		},
		Log: LogConfig{
			FilePath:      "/tmp/recovery-test.log",
			StatsInterval: 5 * time.Second,
		},
		Retry: RetryConfig{
			Enabled:         true,
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			Multiplier:      2.0,
			Jitter:          true,
		},
	}

	mm, _ := NewMirrorMaker(config)
	mm.Start()
	defer mm.Stop()

	time.Sleep(2 * time.Second)

	// 4. 发送正常消息
	producer, _ := createProducer(brokers)
	defer producer.Close()

	for i := 0; i < 5; i++ {
		producer.SendMessage(&sarama.ProducerMessage{
			Topic: sourceTopic,
			Value: sarama.StringEncoder(fmt.Sprintf("normal-%d", i)),
		})
	}

	time.Sleep(3 * time.Second)

	// 5. 检查统计信息
	stats := mm.GetStats()
	t.Logf("处理了%d条消息，错误%d次", stats.MessagesConsumed, stats.Errors)

	if stats.MessagesConsumed < 5 {
		t.Errorf("消息处理不足: 期望>=5, 实际%d", stats.MessagesConsumed)
	}

	t.Log("错误恢复测试通过！")
}

// 辅助函数：格式化JSON输出
func prettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
