package logger

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLogger_Write(t *testing.T) {
	config := &LogConfig{
		FilePath:         "test.log",
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
		os.Remove("test.log")
	}()
	
	// 写入日志
	logger.Write("测试日志消息")
	logger.Printf("格式化日志: %s", "测试")
	logger.Println("日志行")
	
	// 等待异步写入
	time.Sleep(100 * time.Millisecond)
	
	// 验证日志文件是否存在
	if _, err := os.Stat("test.log"); os.IsNotExist(err) {
		t.Error("日志文件应该被创建")
	}
}

func TestLogger_Stop(t *testing.T) {
	config := &LogConfig{
		FilePath:         "test-stop.log",
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
	
	// 写入一些日志
	for i := 0; i < 10; i++ {
		logger.Write("测试日志消息")
	}
	
	// 停止日志管理器
	if err := logger.Stop(); err != nil {
		t.Errorf("停止日志管理器失败: %v", err)
	}
	
	// 清理
	os.Remove("test-stop.log")
}

func TestLogger_DefaultConfig(t *testing.T) {
	ctx := context.Background()
	logger, err := NewLogger(nil, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	defer func() {
		logger.Stop()
		os.Remove("message-mirror.log")
	}()
	
	// 默认配置应该工作
	logger.Write("测试默认配置")
	time.Sleep(100 * time.Millisecond)
	
	if _, err := os.Stat("message-mirror.log"); os.IsNotExist(err) {
		t.Error("默认日志文件应该被创建")
	}
}

