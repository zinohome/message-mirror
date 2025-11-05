package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLogger_Rotate(t *testing.T) {
	config := &LogConfig{
		FilePath:         "test-rotate.log",
		StatsInterval:    10 * time.Second,
		RotateInterval:   100 * time.Millisecond, // 短间隔用于测试
		MaxArchiveFiles:  3,
		AsyncBufferSize:  100,
	}
	
	ctx := context.Background()
	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	
	// 写入一些日志
	for i := 0; i < 10; i++ {
		logger.Write("测试日志消息")
	}
	
	// 等待轮转
	time.Sleep(200 * time.Millisecond)
	
	// 停止日志管理器
	if err := logger.Stop(); err != nil {
		t.Errorf("停止日志管理器失败: %v", err)
	}
	
	// 清理
	os.Remove("test-rotate.log")
	os.Remove("test-rotate.log.tar.gz")
}

func TestLogger_Printf(t *testing.T) {
	config := &LogConfig{
		FilePath:         "test-printf.log",
		StatsInterval:    10 * time.Second,
		RotateInterval:   24 * time.Hour,
		MaxArchiveFiles:  7,
		AsyncBufferSize:  100,
	}
	
	ctx := context.Background()
	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	defer func() {
		logger.Stop()
		os.Remove("test-printf.log")
	}()
	
	logger.Printf("格式化日志: %s, %d", "测试", 123)
	
	// 等待异步写入
	time.Sleep(100 * time.Millisecond)
	
	if _, err := os.Stat("test-printf.log"); os.IsNotExist(err) {
		t.Error("日志文件应该被创建")
	}
}

func TestLogger_Println(t *testing.T) {
	config := &LogConfig{
		FilePath:         "test-println.log",
		StatsInterval:    10 * time.Second,
		RotateInterval:   24 * time.Hour,
		MaxArchiveFiles:  7,
		AsyncBufferSize:  100,
	}
	
	ctx := context.Background()
	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	defer func() {
		logger.Stop()
		os.Remove("test-println.log")
	}()
	
	logger.Println("日志行测试")
	
	// 等待异步写入
	time.Sleep(100 * time.Millisecond)
	
	if _, err := os.Stat("test-println.log"); os.IsNotExist(err) {
		t.Error("日志文件应该被创建")
	}
}

