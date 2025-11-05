package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

// HTTPServer HTTP服务器（提供健康检查和指标）
type HTTPServer struct {
	server   *http.Server
	metrics  *Metrics
	mirror   *MirrorMaker
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewHTTPServer 创建新的HTTP服务器
func NewHTTPServer(addr string, metrics *Metrics, mirror *MirrorMaker, ctx context.Context) *HTTPServer {
	serverCtx, cancel := context.WithCancel(ctx)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	httpServer := &HTTPServer{
		server:  server,
		metrics: metrics,
		mirror:  mirror,
		ctx:     serverCtx,
		cancel:  cancel,
	}

	// 注册路由
	mux.HandleFunc("/health", httpServer.healthHandler)
	mux.HandleFunc("/ready", httpServer.readyHandler)
	mux.Handle("/metrics", promhttp.Handler())

	return httpServer
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start() error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// 启动速率更新goroutine
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		go func() {
			for {
				select {
				case <-s.ctx.Done():
					return
				case <-ticker.C:
					if s.metrics != nil {
						s.metrics.UpdateRates()
					}
				}
			}
		}()

		// 启动HTTP服务器
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP服务器错误: %v", err)
		}
	}()

	log.Printf("HTTP服务器已启动，监听地址: %s", s.server.Addr)
	return nil
}

// Stop 停止HTTP服务器
func (s *HTTPServer) Stop() error {
	s.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	s.wg.Wait()
	log.Println("HTTP服务器已停止")
	return nil
}

// healthHandler 健康检查处理器
func (s *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	// 检查基本状态
	healthy := true
	status := "healthy"

	if s.mirror == nil {
		healthy = false
		status = "unhealthy"
	} else {
		// 可以添加更多健康检查逻辑
		stats := s.mirror.GetStats()
		// 如果最近没有消息处理且运行时间超过1分钟，可能是异常
		if time.Since(stats.LastMessageTime) > 5*time.Minute && time.Since(stats.StartTime) > 1*time.Minute {
			healthy = false
			status = "degraded"
		}
	}

	if s.metrics != nil {
		s.metrics.SetHealthStatus(healthy)
	}

	w.Header().Set("Content-Type", "application/json")
	if healthy {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"` + status + `"}`))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"` + status + `"}`))
	}
}

// readyHandler 就绪检查处理器
func (s *HTTPServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	ready := s.mirror != nil

	w.Header().Set("Content-Type", "application/json")
	if ready {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"not ready"}`))
	}
}

