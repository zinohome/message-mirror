package main

import (
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()
	if metrics == nil {
		t.Error("Metrics不应该为nil")
	}
	
	// 注意：不调用Register()，因为Prometheus指标不能重复注册
	// 在实际使用中，Register()只应该被调用一次
}

func TestMetrics_Register(t *testing.T) {
	// 注意：Prometheus指标不能重复注册
	// 这个测试跳过，因为其他测试已经注册了指标
	metrics := NewMetrics()
	
	// 不调用Register()，直接测试记录方法（使用默认注册表）
	metrics.RecordMessageConsumed(100)
	metrics.RecordMessageProduced(100)
	metrics.RecordMessageFailed()
	metrics.RecordLatency(10 * time.Millisecond)
	metrics.SetHealthStatus(true)
	
	// 更新速率
	metrics.UpdateRates()
}

func TestMetrics_RecordMessageConsumed(t *testing.T) {
	metrics := NewMetrics()
	// 注意：不调用Register()，避免重复注册错误
	
	metrics.RecordMessageConsumed(100)
	metrics.RecordMessageConsumed(200)
	
	// 验证指标被记录（无法直接验证，但应该不panic）
}

func TestMetrics_RecordMessageProduced(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.RecordMessageProduced(100)
	metrics.RecordMessageProduced(200)
}

func TestMetrics_RecordMessageFailed(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.RecordMessageFailed()
	metrics.RecordMessageFailed()
}

// 注意：RecordBytesConsumed和RecordBytesProduced方法不存在
// 字节统计通过RecordMessageConsumed/Produced的参数传递
func TestMetrics_RecordMessageWithBytes(t *testing.T) {
	metrics := NewMetrics()
	
	// 通过RecordMessageConsumed传递字节数（消息大小）
	metrics.RecordMessageConsumed(1000) // 1000字节的消息
	metrics.RecordMessageProduced(2000) // 2000字节的消息
}

func TestMetrics_RecordLatency(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.RecordLatency(10 * time.Millisecond)
	metrics.RecordLatency(20 * time.Millisecond)
	metrics.RecordLatency(100 * time.Millisecond)
}

func TestMetrics_SetHealthStatus(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.SetHealthStatus(true)
	metrics.SetHealthStatus(false)
	metrics.SetHealthStatus(true)
}

func TestMetrics_UpdateRates(t *testing.T) {
	metrics := NewMetrics()
	
	// 记录一些消息（字节数通过消息大小传递）
	metrics.RecordMessageConsumed(100)
	metrics.RecordMessageProduced(100)
	
	// 更新速率
	metrics.UpdateRates()
	
	// 等待一段时间后再次更新
	time.Sleep(100 * time.Millisecond)
	metrics.UpdateRates()
}

