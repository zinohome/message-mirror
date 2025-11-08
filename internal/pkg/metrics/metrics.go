package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics 指标管理器
type Metrics struct {
	// 消息计数器
	MessagesConsumedTotal prometheus.Counter
	MessagesProducedTotal prometheus.Counter
	MessagesFailedTotal  prometheus.Counter

	// 字节计数器
	BytesConsumedTotal prometheus.Counter
	BytesProducedTotal prometheus.Counter

	// 延迟直方图
	MessageLatency prometheus.Histogram

	// 速率仪表盘
	MessageRate prometheus.Gauge
	ByteRate    prometheus.Gauge

	// 错误率
	ErrorRate prometheus.Gauge

	// 健康状态
	HealthStatus prometheus.Gauge

	// 内部状态
	mu            sync.RWMutex
	lastUpdate    time.Time
	messageCount  int64
	byteCount     int64
	errorCount    int64
}

// NewMetrics 创建新的指标管理器
func NewMetrics() *Metrics {
	return &Metrics{
		MessagesConsumedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "message_mirror_messages_consumed_total",
			Help: "Total number of messages consumed",
		}),
		MessagesProducedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "message_mirror_messages_produced_total",
			Help: "Total number of messages produced",
		}),
		MessagesFailedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "message_mirror_messages_failed_total",
			Help: "Total number of messages failed",
		}),
		BytesConsumedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "message_mirror_bytes_consumed_total",
			Help: "Total number of bytes consumed",
		}),
		BytesProducedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "message_mirror_bytes_produced_total",
			Help: "Total number of bytes produced",
		}),
		MessageLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "message_mirror_message_latency_seconds",
			Help:    "Message processing latency in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
		}),
		MessageRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "message_mirror_message_rate_per_second",
			Help: "Current message processing rate per second",
		}),
		ByteRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "message_mirror_byte_rate_per_second",
			Help: "Current byte processing rate per second",
		}),
		ErrorRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "message_mirror_error_rate",
			Help: "Current error rate",
		}),
		HealthStatus: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "message_mirror_health_status",
			Help: "Health status (1 = healthy, 0 = unhealthy)",
		}),
		lastUpdate: time.Now(),
	}
}

// Register 注册所有指标
func (m *Metrics) Register() {
	prometheus.MustRegister(
		m.MessagesConsumedTotal,
		m.MessagesProducedTotal,
		m.MessagesFailedTotal,
		m.BytesConsumedTotal,
		m.BytesProducedTotal,
		m.MessageLatency,
		m.MessageRate,
		m.ByteRate,
		m.ErrorRate,
		m.HealthStatus,
	)
}

// RecordMessageConsumed 记录消息消费
func (m *Metrics) RecordMessageConsumed(bytes int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MessagesConsumedTotal.Inc()
	m.BytesConsumedTotal.Add(float64(bytes))
	m.messageCount++
	m.byteCount += int64(bytes)
}

// RecordMessageProduced 记录消息生产
func (m *Metrics) RecordMessageProduced(bytes int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MessagesProducedTotal.Inc()
	m.BytesProducedTotal.Add(float64(bytes))
}

// RecordMessageFailed 记录消息失败
func (m *Metrics) RecordMessageFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MessagesFailedTotal.Inc()
	m.errorCount++
}

// RecordLatency 记录延迟
func (m *Metrics) RecordLatency(duration time.Duration) {
	m.MessageLatency.Observe(duration.Seconds())
}

// UpdateRates 更新速率指标
func (m *Metrics) UpdateRates() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.lastUpdate).Seconds()

	if elapsed > 0 {
		msgRate := float64(m.messageCount) / elapsed
		byteRate := float64(m.byteCount) / elapsed
		errRate := float64(m.errorCount) / elapsed

		m.MessageRate.Set(msgRate)
		m.ByteRate.Set(byteRate)
		m.ErrorRate.Set(errRate)

		// 重置计数器
		m.messageCount = 0
		m.byteCount = 0
		m.errorCount = 0
		m.lastUpdate = now
	}
}

// SetHealthStatus 设置健康状态
func (m *Metrics) SetHealthStatus(healthy bool) {
	if healthy {
		m.HealthStatus.Set(1)
	} else {
		m.HealthStatus.Set(0)
	}
}
