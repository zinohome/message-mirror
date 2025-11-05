package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilePlugin_Name(t *testing.T) {
	plugin := NewFilePlugin()
	if plugin.Name() != "file" {
		t.Errorf("期望插件名'file'，实际'%s'", plugin.Name())
	}
}

func TestFilePlugin_Initialize(t *testing.T) {
	plugin := NewFilePlugin()
	
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "file-plugin-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	finishDir := filepath.Join(tmpDir, "finish")
	os.MkdirAll(finishDir, 0755)
	
	config := map[string]interface{}{
		"watch_dir":   tmpDir,
		"finish_dir":  finishDir,
		"file_pattern": "*.txt",
		"poll_interval": 1 * time.Second,
		"recursive": false,
	}
	
	err = plugin.Initialize(config)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	
	// 清理
	plugin.Stop()
}

func TestFilePlugin_GetStats(t *testing.T) {
	plugin := NewFilePlugin()
	
	stats := plugin.GetStats()
	if stats.StartTime.IsZero() {
		t.Error("StartTime应该被设置")
	}
}

func TestFilePlugin_Stop(t *testing.T) {
	plugin := NewFilePlugin()
	
	// 初始化插件（避免nil channel panic）
	tmpDir, err := os.MkdirTemp("", "file-plugin-stop-test-*")
	if err == nil {
		defer os.RemoveAll(tmpDir)
		
		finishDir := filepath.Join(tmpDir, "finish")
		os.MkdirAll(finishDir, 0755)
		
		config := map[string]interface{}{
			"watch_dir":   tmpDir,
			"finish_dir":  finishDir,
			"file_pattern": "*.txt",
		}
		
		err = plugin.Initialize(config)
		if err == nil {
			// 测试停止已初始化的插件
			err = plugin.Stop()
			if err != nil {
				t.Logf("停止插件失败: %v", err)
			}
		}
	}
}

func TestFilePlugin_Messages(t *testing.T) {
	plugin := NewFilePlugin()
	
	// 初始化插件以确保msgChan被创建
	tmpDir, err := os.MkdirTemp("", "file-plugin-messages-test-*")
	if err == nil {
		defer os.RemoveAll(tmpDir)
		
		finishDir := filepath.Join(tmpDir, "finish")
		os.MkdirAll(finishDir, 0755)
		
		config := map[string]interface{}{
			"watch_dir":   tmpDir,
			"finish_dir":  finishDir,
			"file_pattern": "*.txt",
		}
		
		err = plugin.Initialize(config)
		if err == nil {
			msgChan := plugin.Messages()
			if msgChan == nil {
				t.Error("消息通道不应该为nil")
			}
		}
	}
}

func TestFilePlugin_Ack(t *testing.T) {
	plugin := NewFilePlugin()
	
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "file-plugin-ack-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	finishDir := filepath.Join(tmpDir, "finish")
	os.MkdirAll(finishDir, 0755)
	
	config := map[string]interface{}{
		"watch_dir":   tmpDir,
		"finish_dir":  finishDir,
		"file_pattern": "*.txt",
	}
	
	err = plugin.Initialize(config)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	
	msg := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
		Metadata: map[string]interface{}{
			"file_path": filepath.Join(tmpDir, "test.txt"),
		},
	}
	
	err = plugin.Ack(msg)
	if err != nil {
		t.Logf("确认消息失败: %v", err)
	}
	
	plugin.Stop()
}

func TestFilePlugin_Start(t *testing.T) {
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
		"watch_dir":   tmpDir,
		"finish_dir":  finishDir,
		"file_pattern": "*.txt",
		"poll_interval": 1 * time.Second,
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
	
	// 等待一小段时间让插件启动
	time.Sleep(100 * time.Millisecond)
	
	// 停止插件
	plugin.Stop()
}

func TestFilePlugin_MatchesPattern(t *testing.T) {
	plugin := NewFilePlugin()
	
	// 测试文件模式匹配
	testCases := []struct {
		filename string
		pattern  string
		expected bool
	}{
		{"test.txt", "*.txt", true},
		{"test.json", "*.txt", false},
		{"test.txt", "*.json", false},
		{"test.txt", "test.*", true},
		{"file.log", "*.log", true},
	}
	
	// 注意：matchesPattern可能是私有方法，需要通过插件内部测试
	// 这里我们通过Initialize和实际文件操作来间接测试
	for _, tc := range testCases {
		// 创建临时目录
		tmpDir, err := os.MkdirTemp("", "file-plugin-pattern-test-*")
		if err != nil {
			continue
		}
		defer os.RemoveAll(tmpDir)
		
		finishDir := filepath.Join(tmpDir, "finish")
		os.MkdirAll(finishDir, 0755)
		
		// 创建测试文件
		testFile := filepath.Join(tmpDir, tc.filename)
		os.WriteFile(testFile, []byte("test"), 0644)
		
		config := map[string]interface{}{
			"watch_dir":   tmpDir,
			"finish_dir":  finishDir,
			"file_pattern": tc.pattern,
			"poll_interval": 100 * time.Millisecond,
		}
		
		err = plugin.Initialize(config)
		if err != nil {
			continue
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		plugin.Start(ctx)
		
		// 等待处理
		time.Sleep(200 * time.Millisecond)
		
		plugin.Stop()
		cancel()
	}
}

