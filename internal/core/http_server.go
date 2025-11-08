package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"message-mirror/internal/pkg/metrics"
	"message-mirror/web"
)

// HTTPServer HTTP服务器（提供健康检查和指标）
type HTTPServer struct {
	server        *http.Server
	metrics       *metrics.Metrics
	mirror        *MirrorMaker
	configManager *ConfigManager
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewHTTPServer 创建新的HTTP服务器
func NewHTTPServer(addr string, metricsInstance *metrics.Metrics, mirror *MirrorMaker, configManager *ConfigManager, ctx context.Context) *HTTPServer {
	serverCtx, cancel := context.WithCancel(ctx)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	httpServer := &HTTPServer{
		server:        server,
		metrics:       metricsInstance,
		mirror:        mirror,
		configManager: configManager,
		ctx:           serverCtx,
		cancel:        cancel,
	}

	// 注册路由
	mux.HandleFunc("/health", httpServer.healthHandler)
	mux.HandleFunc("/ready", httpServer.readyHandler)
	mux.Handle("/metrics", promhttp.Handler())
	
	// 配置API
	mux.HandleFunc("/api/config", httpServer.configHandler)
	mux.HandleFunc("/api/config/reload", httpServer.configReloadHandler)
	mux.HandleFunc("/api/stats", httpServer.statsHandler)
	
	// Web UI静态文件
	mux.HandleFunc("/", httpServer.webUIHandler)
	mux.HandleFunc("/ui", httpServer.webUIHandler)

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

// configHandler 配置处理器
func (s *HTTPServer) configHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.configManager == nil {
		http.Error(w, `{"error":"配置管理器未初始化"}`, http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "GET":
		// 获取配置
		configJSON, err := s.configManager.GetConfigJSON()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"获取配置失败: %v"}`, err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(configJSON)

	case "POST", "PUT":
		// 更新配置
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"读取请求体失败: %v"}`, err), http.StatusBadRequest)
			return
		}

		if err := s.configManager.UpdateConfigFromJSON(body); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"更新配置失败: %v"}`, err), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","message":"配置已更新"}`))

	default:
		http.Error(w, `{"error":"不支持的HTTP方法"}`, http.StatusMethodNotAllowed)
	}
}

// configReloadHandler 配置重载处理器
func (s *HTTPServer) configReloadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.configManager == nil {
		http.Error(w, `{"error":"配置管理器未初始化"}`, http.StatusInternalServerError)
		return
	}

	if r.Method != "POST" {
		http.Error(w, `{"error":"只支持POST方法"}`, http.StatusMethodNotAllowed)
		return
	}

	if err := s.configManager.ReloadFromFile(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"重载配置失败: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"配置已重载"}`))
}

// statsHandler 统计信息处理器
func (s *HTTPServer) statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.mirror == nil {
		http.Error(w, `{"error":"MirrorMaker未初始化"}`, http.StatusInternalServerError)
		return
	}

	stats := s.mirror.GetStats()
	response := map[string]interface{}{
		"messages_consumed": stats.MessagesConsumed,
		"messages_produced": stats.MessagesProduced,
		"bytes_consumed":    stats.BytesConsumed,
		"bytes_produced":    stats.BytesProduced,
		"errors":            stats.Errors,
		"last_message_time": stats.LastMessageTime.Format(time.RFC3339),
		"start_time":        stats.StartTime.Format(time.RFC3339),
		"uptime_seconds":    time.Since(stats.StartTime).Seconds(),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"序列化统计信息失败: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// webUIHandler Web UI处理器
func (s *HTTPServer) webUIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(web.GetWebUIHTML()))
}

