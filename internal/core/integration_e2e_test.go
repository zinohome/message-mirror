package core

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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

	// 等待Kafka就绪
	time.Sleep(5 * time.Second)
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
			CompressionType: "none",
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
	partition, _ := consumer.Partitions(targetTopic)
	pc, _ := consumer.ConsumePartition(targetTopic, partition[0], sarama.OffsetOldest)
	defer pc.Close()

consumeLoop:
	for {
		select {
		case msg := <-pc.Messages():
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

func startKafkaContainer(ctx context.Context) (testcontainers.Container, []string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "confluentinc/cp-kafka:7.5.0",
		ExposedPorts: []string{"9092/tcp", "9093/tcp"},
		Env: map[string]string{
			"KAFKA_BROKER_ID":                                "1",
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":           "PLAINTEXT:PLAINTEXT,PLAINTEXT_INTERNAL:PLAINTEXT",
			"KAFKA_ADVERTISED_LISTENERS":                     "PLAINTEXT://localhost:9092,PLAINTEXT_INTERNAL://localhost:9093",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR":         "1",
			"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":            "1",
			"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR": "1",
			"KAFKA_ZOOKEEPER_CONNECT":                        "ignored",
			"KAFKA_PROCESS_ROLES":                            "broker,controller",
			"KAFKA_NODE_ID":                                  "1",
			"KAFKA_CONTROLLER_QUORUM_VOTERS":                 "1@localhost:9094",
			"KAFKA_LISTENERS":                                "PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9094,PLAINTEXT_INTERNAL://0.0.0.0:9093",
			"KAFKA_INTER_BROKER_LISTENER_NAME":               "PLAINTEXT",
			"KAFKA_CONTROLLER_LISTENER_NAMES":                "CONTROLLER",
			"KAFKA_LOG_DIRS":                                 "/tmp/kraft-combined-logs",
			"CLUSTER_ID":                                     "MkU3OEVBNTcwNTJENDM2Qk",
		},
		WaitingFor: wait.ForLog("started (kafka.server.KafkaRaftServer)").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, nil, err
	}

	port, err := container.MappedPort(ctx, "9092")
	if err != nil {
		return nil, nil, err
	}

	brokers := []string{fmt.Sprintf("%s:%s", host, port.Port())}
	return container, brokers, nil
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

	return sarama.NewSyncProducer(brokers, config)
}

func createConsumer(brokers []string, topic string) (sarama.Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Return.Errors = true

	return sarama.NewConsumer(brokers, config)
}

// TestConcurrentConsumers 并发消费者测试
func TestConcurrentConsumers(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过并发测试")
	}

	// TODO: 实现并发消费者测试
	t.Skip("待实现")
}

// TestErrorRecovery 错误恢复测试
func TestErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过错误恢复测试")
	}

	// TODO: 实现错误恢复测试
	t.Skip("待实现")
}

// 辅助函数：格式化JSON输出
func prettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
