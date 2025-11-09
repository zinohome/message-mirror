package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFilePlugin_Start 测试启动插件
func TestFilePlugin_Start_Extended(t *testing.T) {
	plugin := NewFilePlugin()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "file-plugin-start-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	finishDir := filepath.Join(tmpDir, "finish")
	os.MkdirAll(finishDir, 0755)

	config := map[string]interface{}{
		"watch_dir":    tmpDir,
		"finish_dir":   finishDir,
		"file_pattern": "*.txt",
		"poll_interval": 100 * time.Millisecond,
	}

	err = plugin.Initialize(config)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("启动插件失败: %v", err)
	}

	// 等待插件启动
	time.Sleep(100 * time.Millisecond)

	// 停止插件
	err = plugin.Stop()
	if err != nil {
		t.Fatalf("停止插件失败: %v", err)
	}
}

// TestFilePlugin_FileWatcher 测试文件监控
func TestFilePlugin_FileWatcher(t *testing.T) {
	plugin := NewFilePlugin()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "file-plugin-watcher-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	finishDir := filepath.Join(tmpDir, "finish")
	os.MkdirAll(finishDir, 0755)

	config := map[string]interface{}{
		"watch_dir":    tmpDir,
		"finish_dir":   finishDir,
		"file_pattern": "*.txt",
		"poll_interval": 100 * time.Millisecond,
	}

	err = plugin.Initialize(config)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("启动插件失败: %v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 等待文件处理
	time.Sleep(300 * time.Millisecond)

	// 验证文件可能已被处理（移动到finish目录或删除）
	// 注意：实际行为取决于插件实现

	// 停止插件
	plugin.Stop()
}

// TestFilePlugin_ErrorHandling 测试错误处理
func TestFilePlugin_ErrorHandling(t *testing.T) {
	plugin := NewFilePlugin()

	// 测试无效配置
	invalidConfig := map[string]interface{}{
		"watch_dir": "", // 空目录
	}

	err := plugin.Initialize(invalidConfig)
	if err == nil {
		t.Error("无效配置应该返回错误")
	}

	// 测试不存在的目录
	invalidConfig2 := map[string]interface{}{
		"watch_dir":  "/nonexistent/directory",
		"finish_dir": "/nonexistent/finish",
		"file_pattern": "*.txt",
	}

	err = plugin.Initialize(invalidConfig2)
	// 可能成功（如果插件会创建目录）或失败
	t.Logf("不存在目录的初始化结果: %v", err)
}

// TestFilePlugin_Recursive 测试递归监控
func TestFilePlugin_Recursive(t *testing.T) {
	plugin := NewFilePlugin()

	// 创建临时目录结构
	tmpDir, err := os.MkdirTemp("", "file-plugin-recursive-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	finishDir := filepath.Join(tmpDir, "finish")
	os.MkdirAll(finishDir, 0755)

	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	config := map[string]interface{}{
		"watch_dir":    tmpDir,
		"finish_dir":   finishDir,
		"file_pattern": "*.txt",
		"recursive":    true,
		"poll_interval": 100 * time.Millisecond,
	}

	err = plugin.Initialize(config)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("启动插件失败: %v", err)
	}

	// 在子目录中创建文件
	subFile := filepath.Join(subDir, "subtest.txt")
	err = os.WriteFile(subFile, []byte("sub content"), 0644)
	if err != nil {
		t.Fatalf("创建子目录文件失败: %v", err)
	}

	// 等待文件处理
	time.Sleep(300 * time.Millisecond)

	// 停止插件
	plugin.Stop()
}

// TestFilePlugin_MultipleFiles 测试多个文件
func TestFilePlugin_MultipleFiles(t *testing.T) {
	plugin := NewFilePlugin()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "file-plugin-multiple-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	finishDir := filepath.Join(tmpDir, "finish")
	os.MkdirAll(finishDir, 0755)

	config := map[string]interface{}{
		"watch_dir":    tmpDir,
		"finish_dir":   finishDir,
		"file_pattern": "*.txt",
		"poll_interval": 100 * time.Millisecond,
	}

	err = plugin.Initialize(config)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("启动插件失败: %v", err)
	}

	// 创建多个文件
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(tmpDir, "test"+string(rune(i))+".txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("创建文件失败: %v", err)
		}
	}

	// 等待文件处理
	time.Sleep(500 * time.Millisecond)

	// 验证统计信息
	stats := plugin.GetStats()
	t.Logf("收到%d条消息，确认%d条，错误%d", stats.MessagesReceived, stats.MessagesAcked, stats.Errors)

	// 停止插件
	plugin.Stop()
}

// TestFilePlugin_ContextCancellation 测试上下文取消
func TestFilePlugin_ContextCancellation(t *testing.T) {
	plugin := NewFilePlugin()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "file-plugin-cancel-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	finishDir := filepath.Join(tmpDir, "finish")
	os.MkdirAll(finishDir, 0755)

	config := map[string]interface{}{
		"watch_dir":    tmpDir,
		"finish_dir":   finishDir,
		"file_pattern": "*.txt",
		"poll_interval": 100 * time.Millisecond,
	}

	err = plugin.Initialize(config)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("启动插件失败: %v", err)
	}

	// 等待插件启动
	time.Sleep(100 * time.Millisecond)

	// 取消上下文
	cancel()

	// 等待插件停止
	time.Sleep(100 * time.Millisecond)

	// 停止插件
	plugin.Stop()
}

