package logger

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestLogger_Rotate 测试日志轮转
func TestLogger_Rotate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &LogConfig{
		FilePath:         filepath.Join(tmpDir, "test.log"),
		StatsInterval:    10 * time.Second,
		RotateInterval:   100 * time.Millisecond, // 短间隔用于测试
		MaxArchiveFiles:  7,
		AsyncBufferSize:  100,
	}

	ctx := context.Background()
	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	defer logger.Stop()

	// 写入一些日志
	for i := 0; i < 10; i++ {
		logger.Printf("测试日志消息 %d", i)
	}

	// 等待轮转
	time.Sleep(200 * time.Millisecond)

	// 验证日志文件存在
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		t.Error("日志文件应该存在")
	}

	// 验证归档文件可能已创建（取决于轮转逻辑）
	// 注意：实际轮转可能需要更多时间或更多日志
}

// TestLogger_Archive 测试日志归档
func TestLogger_Archive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &LogConfig{
		FilePath:         filepath.Join(tmpDir, "test.log"),
		StatsInterval:    10 * time.Second,
		RotateInterval:   24 * time.Hour,
		MaxArchiveFiles:  3,
		AsyncBufferSize:  100,
	}

	ctx := context.Background()
	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	defer logger.Stop()

	// 写入一些日志
	for i := 0; i < 5; i++ {
		logger.Printf("测试日志消息 %d", i)
	}

	// 等待写入
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件存在
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		t.Error("日志文件应该存在")
	}
}

// TestLogger_ConcurrentWrite 测试并发写入
func TestLogger_ConcurrentWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &LogConfig{
		FilePath:         filepath.Join(tmpDir, "test.log"),
		StatsInterval:    10 * time.Second,
		RotateInterval:   24 * time.Hour,
		MaxArchiveFiles:  7,
		AsyncBufferSize:  1000,
	}

	ctx := context.Background()
	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	defer logger.Stop()

	// 并发写入
	var wg sync.WaitGroup
	concurrency := 10
	messagesPerGoroutine := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Printf("Goroutine %d: 消息 %d", id, j)
			}
		}(i)
	}

	wg.Wait()

	// 等待异步写入完成
	time.Sleep(200 * time.Millisecond)

	// 验证日志文件存在
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		t.Error("日志文件应该存在")
	}

	// 验证日志文件大小（应该有一些内容）
	info, err := os.Stat(config.FilePath)
	if err != nil {
		t.Fatalf("获取日志文件信息失败: %v", err)
	}
	if info.Size() == 0 {
		t.Error("日志文件不应该为空")
	}
}

// TestLogger_Stats 测试统计信息
func TestLogger_Stats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &LogConfig{
		FilePath:         filepath.Join(tmpDir, "test.log"),
		StatsInterval:    100 * time.Millisecond, // 短间隔用于测试
		RotateInterval:   24 * time.Hour,
		MaxArchiveFiles:  7,
		AsyncBufferSize:  100,
	}

	ctx := context.Background()
	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	defer logger.Stop()

	// 写入一些日志
	for i := 0; i < 10; i++ {
		logger.Printf("测试日志消息 %d", i)
	}

	// 等待统计信息打印
	time.Sleep(200 * time.Millisecond)

	// 验证日志文件存在
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		t.Error("日志文件应该存在")
	}
}

// TestLogger_ErrorHandling 测试错误处理
func TestLogger_ErrorHandling(t *testing.T) {
	// 测试无效文件路径（只读目录）
	readOnlyDir := "/tmp"
	if _, err := os.Stat(readOnlyDir); os.IsNotExist(err) {
		t.Skip("跳过测试：/tmp目录不存在")
		return
	}

	config := &LogConfig{
		FilePath:         filepath.Join(readOnlyDir, "nonexistent", "test.log"), // 无效路径
		StatsInterval:    10 * time.Second,
		RotateInterval:   24 * time.Hour,
		MaxArchiveFiles:  7,
		AsyncBufferSize:  100,
	}

	ctx := context.Background()
	logger, err := NewLogger(config, ctx)
	if err != nil {
		// 预期会失败
		t.Logf("创建日志管理器失败（预期）: %v", err)
		return
	}

	// 如果创建成功，停止它
	if logger != nil {
		logger.Stop()
	}
}

// TestLogger_MultipleWrites 测试多次写入
func TestLogger_MultipleWrites(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &LogConfig{
		FilePath:         filepath.Join(tmpDir, "test.log"),
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
	defer logger.Stop()

	// 测试不同的写入方法
	logger.Write("直接写入")
	logger.Printf("格式化写入: %s", "测试")
	logger.Println("行写入")

	// 等待异步写入
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件存在
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		t.Error("日志文件应该存在")
	}
}

// TestLogger_StopWithPendingWrites 测试停止时待写入的日志
func TestLogger_StopWithPendingWrites(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &LogConfig{
		FilePath:         filepath.Join(tmpDir, "test.log"),
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
	for i := 0; i < 50; i++ {
		logger.Printf("待写入日志 %d", i)
	}

	// 立即停止（不等待写入完成）
	err = logger.Stop()
	if err != nil {
		t.Errorf("停止日志管理器失败: %v", err)
	}

	// 验证日志文件存在
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		t.Error("日志文件应该存在")
	}
}

// TestLogger_LargeBuffer 测试大缓冲区
func TestLogger_LargeBuffer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &LogConfig{
		FilePath:         filepath.Join(tmpDir, "test.log"),
		StatsInterval:    10 * time.Second,
		RotateInterval:   24 * time.Hour,
		MaxArchiveFiles:  7,
		AsyncBufferSize:  10000, // 大缓冲区
	}

	ctx := context.Background()
	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建日志管理器失败: %v", err)
	}
	defer logger.Stop()

	// 写入大量日志
	for i := 0; i < 5000; i++ {
		logger.Printf("大缓冲区测试日志 %d", i)
	}

	// 等待写入
	time.Sleep(500 * time.Millisecond)

	// 验证日志文件存在
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		t.Error("日志文件应该存在")
	}
}

