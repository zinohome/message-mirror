package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	configPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "message-mirror",
		Short: "Kafka消息镜像工具",
		Long:  "一个用Go语言开发的Kafka消息镜像工具，用于在Kafka集群之间复制消息",
		RunE:  run,
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "配置文件路径")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("执行失败: %v", err)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// 加载配置
	config, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	log.Printf("配置加载成功: 源类型=%s, 目标集群=%v",
		config.Source.Type, config.Target.Brokers)

	// 创建MirrorMaker
	mirrorMaker, err := NewMirrorMaker(config)
	if err != nil {
		return err
	}

	// 启动MirrorMaker
	if err := mirrorMaker.Start(); err != nil {
		return err
	}

	// 启动HTTP服务器（如果启用）
	var httpServer *HTTPServer
	if config.Server.Enabled {
		metrics := mirrorMaker.GetMetrics()
		httpServer = NewHTTPServer(config.Server.Address, metrics, mirrorMaker, context.Background())
		if err := httpServer.Start(); err != nil {
			log.Printf("启动HTTP服务器失败: %v", err)
		}
	}

	// 设置信号处理，实现优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 等待信号
	<-sigChan
	log.Println("收到停止信号，正在关闭...")

	// 停止HTTP服务器
	if httpServer != nil {
		if err := httpServer.Stop(); err != nil {
			log.Printf("停止HTTP服务器失败: %v", err)
		}
	}

	// 停止MirrorMaker
	if err := mirrorMaker.Stop(); err != nil {
		return err
	}

	// 打印最终统计信息
	stats := mirrorMaker.GetStats()
	log.Printf("最终统计信息:")
	log.Printf("  消费消息数: %d", stats.MessagesConsumed)
	log.Printf("  生产消息数: %d", stats.MessagesProduced)
	log.Printf("  错误数: %d", stats.Errors)
	log.Printf("  运行时间: %v", time.Since(stats.StartTime))

	return nil
}
