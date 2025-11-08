package plugins

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FilePlugin 文件监控数据源插件
type FilePlugin struct {
	config     *FilePluginConfig
	watcher    *fsnotify.Watcher
	msgChan    chan *Message
	stats      *FilePluginStats
	statsMu    sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	processedFiles map[string]bool
	processedMu    sync.RWMutex
}

// FilePluginConfig 文件插件配置
type FilePluginConfig struct {
	WatchDir      string
	FinishDir     string
	FilePattern   string // 文件匹配模式，如 "*.txt", "*.json"
	PollInterval  time.Duration
	Recursive     bool // 是否递归监控子目录
}

// FilePluginStats 文件插件统计信息
type FilePluginStats struct {
	MessagesReceived int64
	MessagesAcked    int64
	Errors           int64
	FilesProcessed   int64
	LastMessageTime  time.Time
	StartTime        time.Time
}

// NewFilePlugin 创建文件插件实例
func NewFilePlugin() SourcePlugin {
	return &FilePlugin{
		stats: &FilePluginStats{
			StartTime: time.Now(),
		},
		processedFiles: make(map[string]bool),
	}
}

// Name 返回插件名称
func (p *FilePlugin) Name() string {
	return "file"
}

// Initialize 初始化插件
func (p *FilePlugin) Initialize(config map[string]interface{}) error {
	cfg := &FilePluginConfig{}

	if watchDir, ok := config["watch_dir"].(string); ok {
		cfg.WatchDir = watchDir
	} else {
		return fmt.Errorf("watch_dir配置不能为空")
	}

	if finishDir, ok := config["finish_dir"].(string); ok {
		cfg.FinishDir = finishDir
	} else {
		return fmt.Errorf("finish_dir配置不能为空")
	}

	if filePattern, ok := config["file_pattern"].(string); ok {
		cfg.FilePattern = filePattern
	} else {
		cfg.FilePattern = "*" // 默认匹配所有文件
	}

	if pollInterval, ok := config["poll_interval"].(time.Duration); ok {
		cfg.PollInterval = pollInterval
	} else {
		cfg.PollInterval = 1 * time.Second
	}

	if recursive, ok := config["recursive"].(bool); ok {
		cfg.Recursive = recursive
	} else {
		cfg.Recursive = false
	}

	p.config = cfg
	p.msgChan = make(chan *Message, 100)

	// 确保目录存在
	if err := os.MkdirAll(cfg.WatchDir, 0755); err != nil {
		return fmt.Errorf("创建监控目录失败: %w", err)
	}

	if err := os.MkdirAll(cfg.FinishDir, 0755); err != nil {
		return fmt.Errorf("创建完成目录失败: %w", err)
	}

	return nil
}

// Start 启动插件
func (p *FilePlugin) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// 创建文件监控器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建文件监控器失败: %w", err)
	}
	p.watcher = watcher

	// 添加监控目录
	if p.config.Recursive {
		err = filepath.Walk(p.config.WatchDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return watcher.Add(path)
			}
			return nil
		})
	} else {
		err = watcher.Add(p.config.WatchDir)
	}

	if err != nil {
		watcher.Close()
		return fmt.Errorf("添加监控目录失败: %w", err)
	}

	log.Printf("[文件插件] 开始监控目录: %s (模式: %s)", p.config.WatchDir, p.config.FilePattern)

	// 处理文件系统事件
	p.wg.Add(1)
	go p.watchEvents()

	// 初始扫描已存在的文件
	p.wg.Add(1)
	go p.scanExistingFiles()

	return nil
}

// watchEvents 监控文件系统事件
func (p *FilePlugin) watchEvents() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return

		case event, ok := <-p.watcher.Events:
			if !ok {
				return
			}

			// 只处理文件写入完成事件
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// 检查文件是否匹配模式
				if p.matchesPattern(event.Name) {
					p.processFile(event.Name)
				}
			}

		case err, ok := <-p.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[文件插件] 监控错误: %v", err)
			p.statsMu.Lock()
			p.stats.Errors++
			p.statsMu.Unlock()
		}
	}
}

// scanExistingFiles 扫描已存在的文件
func (p *FilePlugin) scanExistingFiles() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.scanDirectory(p.config.WatchDir)
		}
	}
}

// scanDirectory 扫描目录中的文件
func (p *FilePlugin) scanDirectory(dir string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() && p.matchesPattern(path) {
			p.processFile(path)
		}

		return nil
	})

	if err != nil {
		log.Printf("[文件插件] 扫描目录错误: %v", err)
	}
}

// matchesPattern 检查文件是否匹配模式
func (p *FilePlugin) matchesPattern(filename string) bool {
	if p.config.FilePattern == "*" {
		return true
	}

	matched, err := filepath.Match(p.config.FilePattern, filepath.Base(filename))
	if err != nil {
		return false
	}

	return matched
}

// processFile 处理文件
func (p *FilePlugin) processFile(filePath string) {
	// 检查是否已处理
	p.processedMu.RLock()
	if p.processedFiles[filePath] {
		p.processedMu.RUnlock()
		return
	}
	p.processedMu.RUnlock()

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("[文件插件] 读取文件失败 %s: %v", filePath, err)
		p.statsMu.Lock()
		p.stats.Errors++
		p.statsMu.Unlock()
		return
	}

	// 标记为已处理
	p.processedMu.Lock()
	p.processedFiles[filePath] = true
	p.processedMu.Unlock()

	// 创建消息
	msg := &Message{
		Key:       []byte(filepath.Base(filePath)),
		Value:     content,
		Headers:   make(map[string][]byte),
		Timestamp: time.Now(),
		Source:    "file",
		Metadata: map[string]interface{}{
			"file_path": filePath,
			"file_name": filepath.Base(filePath),
			"file_size": len(content),
		},
	}

	// 更新统计
	p.statsMu.Lock()
	p.stats.MessagesReceived++
	p.stats.LastMessageTime = time.Now()
	p.statsMu.Unlock()

	// 发送消息
	select {
	case p.msgChan <- msg:
	case <-p.ctx.Done():
		return
	}
}

// moveToFinishDir 移动文件到完成目录
func (p *FilePlugin) moveToFinishDir(filePath string) error {
	fileName := filepath.Base(filePath)
	finishPath := filepath.Join(p.config.FinishDir, fileName)

	// 如果目标文件已存在，添加时间戳
	if _, err := os.Stat(finishPath); err == nil {
		ext := filepath.Ext(fileName)
		name := strings.TrimSuffix(fileName, ext)
		finishPath = filepath.Join(p.config.FinishDir, fmt.Sprintf("%s_%d%s", name, time.Now().Unix(), ext))
	}

	err := os.Rename(filePath, finishPath)
	if err != nil {
		return fmt.Errorf("移动文件失败: %w", err)
	}

	p.statsMu.Lock()
	p.stats.FilesProcessed++
	p.statsMu.Unlock()

	log.Printf("[文件插件] 文件已移动到完成目录: %s -> %s", filePath, finishPath)
	return nil
}

// Stop 停止插件
func (p *FilePlugin) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.watcher != nil {
		p.watcher.Close()
	}
	p.wg.Wait()
	if p.msgChan != nil {
		close(p.msgChan)
	}
	log.Println("[文件插件] 已停止")
	return nil
}

// Messages 返回消息通道
func (p *FilePlugin) Messages() <-chan *Message {
	return p.msgChan
}

// Ack 确认消息（处理后移动文件）
func (p *FilePlugin) Ack(msg *Message) error {
	if filePath, ok := msg.Metadata["file_path"].(string); ok {
		err := p.moveToFinishDir(filePath)
		if err != nil {
			p.statsMu.Lock()
			p.stats.Errors++
			p.statsMu.Unlock()
			return err
		}

		// 从已处理列表中移除
		p.processedMu.Lock()
		delete(p.processedFiles, filePath)
		p.processedMu.Unlock()

		p.statsMu.Lock()
		p.stats.MessagesAcked++
		p.statsMu.Unlock()
	}
	return nil
}

// GetStats 获取统计信息
func (p *FilePlugin) GetStats() PluginStats {
	p.statsMu.RLock()
	defer p.statsMu.RUnlock()

	return PluginStats{
		MessagesReceived: p.stats.MessagesReceived,
		MessagesAcked:    p.stats.MessagesAcked,
		Errors:           p.stats.Errors,
		LastMessageTime:  p.stats.LastMessageTime,
		StartTime:        p.stats.StartTime,
	}
}

func init() {
	RegisterPlugin("file", NewFilePlugin)
}
