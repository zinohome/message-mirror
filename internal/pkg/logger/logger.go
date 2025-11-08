package logger

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// LogConfig 日志配置（临时定义，后续会从core包导入）
type LogConfig struct {
	FilePath         string
	StatsInterval    time.Duration
	RotateInterval   time.Duration
	MaxArchiveFiles  int
	AsyncBufferSize  int
}

// Logger 异步日志管理器
type Logger struct {
	config      *LogConfig
	file        *os.File
	filePath    string
	writeChan   chan string
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.RWMutex
	lastRotate  time.Time
}

// NewLogger 创建新的日志管理器
func NewLogger(config *LogConfig, ctx context.Context) (*Logger, error) {
	if config == nil {
		config = &LogConfig{
			FilePath:         "message-mirror.log",
			StatsInterval:    10 * time.Second,
			RotateInterval:   24 * time.Hour,
			MaxArchiveFiles:  7,
			AsyncBufferSize: 1000,
		}
	}

	loggerCtx, cancel := context.WithCancel(ctx)

	logger := &Logger{
		config:     config,
		filePath:   config.FilePath,
		writeChan:  make(chan string, config.AsyncBufferSize),
		ctx:        loggerCtx,
		cancel:     cancel,
		lastRotate: time.Now(),
	}

	// 打开日志文件
	if err := logger.openFile(); err != nil {
		cancel()
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 启动异步写入goroutine
	logger.wg.Add(1)
	go logger.asyncWriter()

	// 启动日志轮转goroutine
	logger.wg.Add(1)
	go logger.rotateWorker()

	return logger, nil
}

// openFile 打开日志文件
func (l *Logger) openFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 确保目录存在
	dir := filepath.Dir(l.filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
	}

	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// 关闭旧文件
	if l.file != nil {
		l.file.Close()
	}

	l.file = file
	return nil
}

// asyncWriter 异步写入goroutine
func (l *Logger) asyncWriter() {
	defer l.wg.Done()

	for {
		select {
		case <-l.ctx.Done():
			// 关闭前写入所有待写入的消息
			l.flush()
			return

		case msg := <-l.writeChan:
			l.mu.RLock()
			file := l.file
			l.mu.RUnlock()

			if file != nil {
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				logLine := fmt.Sprintf("[%s] %s\n", timestamp, msg)
				
				if _, err := file.WriteString(logLine); err != nil {
					log.Printf("写入日志失败: %v", err)
				}
			}
		}
	}
}

// flush 刷新所有待写入的消息
func (l *Logger) flush() {
	for {
		select {
		case msg := <-l.writeChan:
			l.mu.RLock()
			file := l.file
			l.mu.RUnlock()

			if file != nil {
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				logLine := fmt.Sprintf("[%s] %s\n", timestamp, msg)
				file.WriteString(logLine)
			}
		default:
			return
		}
	}
}

// rotateWorker 日志轮转工作goroutine
func (l *Logger) rotateWorker() {
	defer l.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // 每小时检查一次
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return

		case <-ticker.C:
			if time.Since(l.lastRotate) >= l.config.RotateInterval {
				if err := l.rotate(); err != nil {
					log.Printf("日志轮转失败: %v", err)
				}
			}
		}
	}
}

// rotate 执行日志轮转
func (l *Logger) rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 关闭当前文件
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	// 如果日志文件不存在或为空，直接打开新文件
	info, err := os.Stat(l.filePath)
	if err != nil || info.Size() == 0 {
		l.lastRotate = time.Now()
		return l.openFile()
	}

	// 生成归档文件名（带时间戳）
	timestamp := time.Now().Format("20060102-150405")
	archivePath := fmt.Sprintf("%s.%s.tar.gz", l.filePath, timestamp)

	// 压缩当前日志文件
	if err := l.compressLogFile(l.filePath, archivePath); err != nil {
		return fmt.Errorf("压缩日志文件失败: %w", err)
	}

	// 删除原始日志文件
	if err := os.Remove(l.filePath); err != nil {
		return fmt.Errorf("删除原始日志文件失败: %w", err)
	}

	// 清理旧的归档文件
	if err := l.cleanupOldArchives(); err != nil {
		log.Printf("清理旧归档文件失败: %v", err)
	}

	// 打开新的日志文件
	l.lastRotate = time.Now()
	return l.openFile()
}

// compressLogFile 压缩日志文件为tar.gz
func (l *Logger) compressLogFile(srcPath, destPath string) error {
	// 打开源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建目标文件
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 创建gzip writer
	gzWriter := gzip.NewWriter(destFile)
	defer gzWriter.Close()

	// 创建tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 获取文件信息
	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// 创建tar header
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.Base(srcPath)

	// 写入header
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	// 复制文件内容
	if _, err := io.Copy(tarWriter, srcFile); err != nil {
		return err
	}

	return nil
}

// cleanupOldArchives 清理旧的归档文件
func (l *Logger) cleanupOldArchives() error {
	logDir := filepath.Dir(l.filePath)
	if logDir == "." || logDir == "" {
		logDir = "."
	}

	baseName := filepath.Base(l.filePath)
	pattern := fmt.Sprintf("%s.*.tar.gz", baseName)

	// 查找所有匹配的归档文件
	matches, err := filepath.Glob(filepath.Join(logDir, pattern))
	if err != nil {
		return err
	}

	// 如果归档文件数量超过限制，删除最旧的
	if len(matches) > l.config.MaxArchiveFiles {
		// 按文件名排序（文件名包含时间戳，所以可以按文件名排序）
		sort.Strings(matches)

		// 删除最旧的文件
		toDelete := len(matches) - l.config.MaxArchiveFiles
		for i := 0; i < toDelete; i++ {
			if err := os.Remove(matches[i]); err != nil {
				log.Printf("删除归档文件失败 %s: %v", matches[i], err)
			} else {
				log.Printf("删除旧归档文件: %s", matches[i])
			}
		}
	}

	return nil
}

// Write 写入日志（异步）
func (l *Logger) Write(msg string) {
	select {
	case l.writeChan <- msg:
	default:
		// 如果缓冲区满了，直接输出到标准错误（避免阻塞）
		log.Printf("日志缓冲区已满，直接输出: %s", msg)
	}
}

// Printf 格式化写入日志（异步）
func (l *Logger) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.Write(msg)
}

// Println 写入日志行（异步）
func (l *Logger) Println(args ...interface{}) {
	msg := fmt.Sprintln(args...)
	// 移除末尾的换行符（Write会添加）
	msg = msg[:len(msg)-1]
	l.Write(msg)
}

// Stop 停止日志管理器
func (l *Logger) Stop() error {
	l.cancel()
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}

	return nil
}
