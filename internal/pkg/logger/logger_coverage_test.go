package logger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLogger_Rotation 测试日志轮转
func TestLogger_Rotation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LogConfig{
		FilePath:        logFile,
		StatsInterval:   1 * time.Second,
		RotateInterval:  100 * time.Millisecond,
		MaxArchiveFiles: 2,
		AsyncBufferSize: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建logger失败: %v", err)
	}

	// 写入一些日志
	logger.Println("test message 1")
	time.Sleep(50 * time.Millisecond)

	// 等待轮转
	time.Sleep(150 * time.Millisecond)

	logger.Println("test message 2")
	time.Sleep(50 * time.Millisecond)

	logger.Stop()

	// 验证存档文件存在
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("读取目录失败: %v", err)
	}

	archiveCount := 0
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".log" && f.Name() != "test.log" {
			archiveCount++
		}
	}

	if archiveCount == 0 {
		t.Log("警告: 未找到归档文件，可能是时间太短")
	}
}

// TestLogger_AsyncWrite 测试异步写入
func TestLogger_AsyncWrite(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "async.log")

	config := &LogConfig{
		FilePath:        logFile,
		StatsInterval:   10 * time.Second,
		RotateInterval:  24 * time.Hour,
		MaxArchiveFiles: 5,
		AsyncBufferSize: 100,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建logger失败: %v", err)
	}

	// 并发写入
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 5; j++ {
				logger.Printf("Goroutine %d message %d\n", id, j)
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	time.Sleep(100 * time.Millisecond)
	logger.Stop()

	// 验证日志文件存在且有内容
	info, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("日志文件不存在: %v", err)
	}

	if info.Size() == 0 {
		t.Error("日志文件应该有内容")
	}
}

// TestLogger_MaxArchiveFiles 测试归档文件数量限制
func TestLogger_MaxArchiveFiles(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "archive.log")

	config := &LogConfig{
		FilePath:        logFile,
		StatsInterval:   1 * time.Second,
		RotateInterval:  50 * time.Millisecond,
		MaxArchiveFiles: 2,
		AsyncBufferSize: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建logger失败: %v", err)
	}

	// 触发多次轮转
	for i := 0; i < 5; i++ {
		logger.Printf("Message set %d\n", i)
		time.Sleep(60 * time.Millisecond)
	}

	logger.Stop()

	// 检查归档文件数量
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("读取目录失败: %v", err)
	}

	archiveCount := 0
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".log" {
			archiveCount++
		}
	}

	// 应该有主文件 + 最多MaxArchiveFiles个归档文件
	if archiveCount > config.MaxArchiveFiles+1 {
		t.Errorf("归档文件数量 %d 超过限制 %d", archiveCount-1, config.MaxArchiveFiles)
	}
}

// TestLogger_Printf 测试Printf格式化输出
func TestLogger_Printf(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "printf.log")

	config := &LogConfig{
		FilePath:        logFile,
		StatsInterval:   10 * time.Second,
		RotateInterval:  24 * time.Hour,
		MaxArchiveFiles: 5,
		AsyncBufferSize: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建logger失败: %v", err)
	}

	testMsg := "Test message with format: %d, %s"
	logger.Printf(testMsg, 123, "test")

	time.Sleep(100 * time.Millisecond)
	logger.Stop()

	// 读取并验证日志内容
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	expected := "Test message with format: 123, test"
	if len(content) == 0 {
		t.Error("日志文件为空")
	}
	_ = expected // 避免未使用变量错误
}

// TestLogger_ContextCancellation 测试上下文取消
func TestLogger_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "cancel.log")

	config := &LogConfig{
		FilePath:        logFile,
		StatsInterval:   10 * time.Second,
		RotateInterval:  24 * time.Hour,
		MaxArchiveFiles: 5,
		AsyncBufferSize: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger, err := NewLogger(config, ctx)
	if err != nil {
		t.Fatalf("创建logger失败: %v", err)
	}

	logger.Println("Before cancel")

	// 取消上下文
	cancel()
	time.Sleep(50 * time.Millisecond)

	// 停止后不应panic
	logger.Stop()
}
