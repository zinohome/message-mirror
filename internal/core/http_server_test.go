package core

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"message-mirror/internal/pkg/metrics"
)

// TestNewHTTPServer 测试创建HTTP服务器
func TestNewHTTPServer(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	mirror := &MirrorMaker{}
	configManager := &ConfigManager{}
	ctx := context.Background()

	server := NewHTTPServer(":8080", metricsInstance, mirror, configManager, ctx)
	if server == nil {
		t.Fatal("HTTPServer不应该为nil")
	}
	if server.server == nil {
		t.Error("HTTP server不应该为nil")
	}
	if server.metrics != metricsInstance {
		t.Error("metrics应该被正确设置")
	}
	if server.mirror != mirror {
		t.Error("mirror应该被正确设置")
	}
	if server.configManager != configManager {
		t.Error("configManager应该被正确设置")
	}
}

// TestHTTPServer_Start 测试启动服务器
func TestHTTPServer_Start(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	mirror := &MirrorMaker{}
	configManager := &ConfigManager{}
	ctx := context.Background()

	server := NewHTTPServer(":0", metricsInstance, mirror, configManager, ctx) // 使用:0让系统自动分配端口
	
	err := server.Start()
	if err != nil {
		t.Fatalf("启动HTTP服务器失败: %v", err)
	}
	
	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)
	
	// 停止服务器
	err = server.Stop()
	if err != nil {
		t.Fatalf("停止HTTP服务器失败: %v", err)
	}
}

// TestHTTPServer_Stop 测试停止服务器
func TestHTTPServer_Stop(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	mirror := &MirrorMaker{}
	configManager := &ConfigManager{}
	ctx := context.Background()

	server := NewHTTPServer(":0", metricsInstance, mirror, configManager, ctx)
	server.Start()
	
	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)
	
	// 停止服务器
	err := server.Stop()
	if err != nil {
		t.Fatalf("停止HTTP服务器失败: %v", err)
	}
	
	// 再次停止应该不会出错
	err = server.Stop()
	if err != nil {
		t.Logf("再次停止服务器（可能已停止）: %v", err)
	}
}

// TestHTTPServer_healthHandler 测试健康检查端点
func TestHTTPServer_healthHandler(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	ctx := context.Background()

	// 测试健康状态
	mirror := &MirrorMaker{
		stats: &Stats{
			StartTime:       time.Now(),
			LastMessageTime: time.Now(), // 最近有消息
		},
	}
	configManager := &ConfigManager{}
	server := NewHTTPServer(":0", metricsInstance, mirror, configManager, ctx)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w.Code)
	}
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if response["status"] != "healthy" {
		t.Errorf("期望status='healthy'，实际'%s'", response["status"])
	}

	// 测试不健康状态（mirror为nil）
	server2 := NewHTTPServer(":0", metricsInstance, nil, configManager, ctx)
	req2 := httptest.NewRequest("GET", "/health", nil)
	w2 := httptest.NewRecorder()
	server2.healthHandler(w2, req2)

	if w2.Code != http.StatusServiceUnavailable {
		t.Errorf("期望状态码503，实际%d", w2.Code)
	}

	// 测试降级状态（长时间没有消息）
	mirror3 := &MirrorMaker{
		stats: &Stats{
			StartTime:       time.Now().Add(-10 * time.Minute),
			LastMessageTime: time.Now().Add(-10 * time.Minute), // 10分钟前有消息
		},
	}
	server3 := NewHTTPServer(":0", metricsInstance, mirror3, configManager, ctx)
	req3 := httptest.NewRequest("GET", "/health", nil)
	w3 := httptest.NewRecorder()
	server3.healthHandler(w3, req3)

	if w3.Code != http.StatusServiceUnavailable {
		t.Errorf("期望状态码503（降级），实际%d", w3.Code)
	}
}

// TestHTTPServer_readyHandler 测试就绪检查端点
func TestHTTPServer_readyHandler(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	mirror := &MirrorMaker{}
	configManager := &ConfigManager{}
	ctx := context.Background()

	// 测试就绪状态
	server := NewHTTPServer(":0", metricsInstance, mirror, configManager, ctx)
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	server.readyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w.Code)
	}
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if response["status"] != "ready" {
		t.Errorf("期望status='ready'，实际'%s'", response["status"])
	}

	// 测试不就绪状态（mirror为nil）
	server2 := NewHTTPServer(":0", metricsInstance, nil, configManager, ctx)
	req2 := httptest.NewRequest("GET", "/ready", nil)
	w2 := httptest.NewRecorder()
	server2.readyHandler(w2, req2)

	if w2.Code != http.StatusServiceUnavailable {
		t.Errorf("期望状态码503，实际%d", w2.Code)
	}
}

// TestHTTPServer_configHandler 测试配置API
func TestHTTPServer_configHandler(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	mirror := &MirrorMaker{}
	ctx := context.Background()

	// 创建临时配置文件
	tmpFile, err := createTempConfigFileForHTTPTest(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configManager, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	server := NewHTTPServer(":0", metricsInstance, mirror, configManager, ctx)

	// 测试GET请求
	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	server.configHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w.Code)
	}
	
	var config map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &config)
	if err != nil {
		t.Fatalf("解析配置失败: %v", err)
	}

	// 测试POST请求
	currentConfig := configManager.GetConfig()
	currentConfig.Mirror.WorkerCount = 8
	configJSON, err := json.Marshal(currentConfig)
	if err != nil {
		t.Fatalf("序列化配置失败: %v", err)
	}

	req2 := httptest.NewRequest("POST", "/api/config", bytes.NewReader(configJSON))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	server.configHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w2.Code)
	}

	// 验证配置已更新
	updatedConfig := configManager.GetConfig()
	if updatedConfig.Mirror.WorkerCount != 8 {
		t.Errorf("期望worker_count=8，实际%d", updatedConfig.Mirror.WorkerCount)
	}

	// 测试PUT请求
	currentConfig.Mirror.WorkerCount = 12
	configJSON, err = json.Marshal(currentConfig)
	if err != nil {
		t.Fatalf("序列化配置失败: %v", err)
	}

	req3 := httptest.NewRequest("PUT", "/api/config", bytes.NewReader(configJSON))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	server.configHandler(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w3.Code)
	}

	// 测试OPTIONS请求
	req4 := httptest.NewRequest("OPTIONS", "/api/config", nil)
	w4 := httptest.NewRecorder()
	server.configHandler(w4, req4)

	if w4.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w4.Code)
	}

	// 测试无效请求（DELETE方法）
	req5 := httptest.NewRequest("DELETE", "/api/config", nil)
	w5 := httptest.NewRecorder()
	server.configHandler(w5, req5)

	if w5.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态码405，实际%d", w5.Code)
	}

	// 测试configManager为nil的情况
	server2 := NewHTTPServer(":0", metricsInstance, mirror, nil, ctx)
	req6 := httptest.NewRequest("GET", "/api/config", nil)
	w6 := httptest.NewRecorder()
	server2.configHandler(w6, req6)

	if w6.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码500，实际%d", w6.Code)
	}

	// 测试无效JSON
	req7 := httptest.NewRequest("POST", "/api/config", bytes.NewReader([]byte("{invalid json}")))
	req7.Header.Set("Content-Type", "application/json")
	w7 := httptest.NewRecorder()
	server.configHandler(w7, req7)

	if w7.Code != http.StatusBadRequest {
		t.Errorf("期望状态码400，实际%d", w7.Code)
	}
}

// TestHTTPServer_configReloadHandler 测试配置重载端点
func TestHTTPServer_configReloadHandler(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	mirror := &MirrorMaker{}
	ctx := context.Background()

	// 创建临时配置文件
	tmpFile, err := createTempConfigFileForHTTPTest(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configManager, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	server := NewHTTPServer(":0", metricsInstance, mirror, configManager, ctx)

	// 测试POST请求
	req := httptest.NewRequest("POST", "/api/config/reload", nil)
	w := httptest.NewRecorder()
	server.configReloadHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w.Code)
	}

	// 测试非POST请求
	req2 := httptest.NewRequest("GET", "/api/config/reload", nil)
	w2 := httptest.NewRecorder()
	server.configReloadHandler(w2, req2)

	if w2.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态码405，实际%d", w2.Code)
	}

	// 测试configManager为nil的情况
	server2 := NewHTTPServer(":0", metricsInstance, mirror, nil, ctx)
	req3 := httptest.NewRequest("POST", "/api/config/reload", nil)
	w3 := httptest.NewRecorder()
	server2.configReloadHandler(w3, req3)

	if w3.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码500，实际%d", w3.Code)
	}
}

// TestHTTPServer_statsHandler 测试统计信息端点
func TestHTTPServer_statsHandler(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	ctx := context.Background()

	// 创建带统计信息的mirror
	mirror := &MirrorMaker{
		stats: &Stats{
			MessagesConsumed: 100,
			MessagesProduced: 95,
			BytesConsumed:    10000,
			BytesProduced:    9500,
			Errors:           5,
			StartTime:        time.Now().Add(-10 * time.Minute),
			LastMessageTime:  time.Now(),
		},
	}
	configManager := &ConfigManager{}
	server := NewHTTPServer(":0", metricsInstance, mirror, configManager, ctx)

	req := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()
	server.statsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w.Code)
	}

	var stats map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	if err != nil {
		t.Fatalf("解析统计信息失败: %v", err)
	}

	if stats["messages_consumed"].(float64) != 100 {
		t.Errorf("期望messages_consumed=100，实际%v", stats["messages_consumed"])
	}
	if stats["messages_produced"].(float64) != 95 {
		t.Errorf("期望messages_produced=95，实际%v", stats["messages_produced"])
	}
	if stats["errors"].(float64) != 5 {
		t.Errorf("期望errors=5，实际%v", stats["errors"])
	}

	// 测试mirror为nil的情况
	server2 := NewHTTPServer(":0", metricsInstance, nil, configManager, ctx)
	req2 := httptest.NewRequest("GET", "/api/stats", nil)
	w2 := httptest.NewRecorder()
	server2.statsHandler(w2, req2)

	if w2.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码500，实际%d", w2.Code)
	}
}

// TestHTTPServer_webUIHandler 测试Web UI端点
func TestHTTPServer_webUIHandler(t *testing.T) {
	metricsInstance := metrics.NewMetrics()
	mirror := &MirrorMaker{}
	configManager := &ConfigManager{}
	ctx := context.Background()

	server := NewHTTPServer(":0", metricsInstance, mirror, configManager, ctx)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.webUIHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("期望Content-Type='text/html; charset=utf-8'，实际'%s'", w.Header().Get("Content-Type"))
	}

	if len(w.Body.Bytes()) == 0 {
		t.Error("响应体不应该为空")
	}

	// 测试/ui端点
	req2 := httptest.NewRequest("GET", "/ui", nil)
	w2 := httptest.NewRecorder()
	server.webUIHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w2.Code)
	}
}

// createTempConfigFileForHTTPTest 为HTTP测试创建临时配置文件
func createTempConfigFileForHTTPTest(t *testing.T) (*os.File, error) {
	configContent := `
source:
  type: "kafka"
  brokers:
    - "localhost:9092"
  topic: "test-topic"

target:
  brokers:
    - "localhost:9093"
  topic: "target-topic"

mirror:
  worker_count: 4
`
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		return nil, err
	}
	
	if _, err := tmpFile.WriteString(configContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, err
	}
	tmpFile.Close()
	
	return tmpFile, nil
}

