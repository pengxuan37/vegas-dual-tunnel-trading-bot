package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/app"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/config"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
)

func main() {
	// 初始化日志
	logger := logger.NewLogger()
	logger.Info("Starting Vegas Dual Tunnel Trading Bot...")

	// 加载配置
	cfg, err := config.Load("config.json")
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// 创建应用实例
	app, err := app.New(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create app: %v", err)
	}

	// 创建上下文和等待组
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// 启动应用
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.Run(ctx); err != nil {
			logger.Errorf("App run error: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Received shutdown signal, gracefully shutting down...")

	// 取消上下文
	cancel()

	// 等待所有goroutine完成，最多等待30秒
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Graceful shutdown completed")
	case <-time.After(30 * time.Second):
		logger.Warn("Shutdown timeout, forcing exit")
	}

	fmt.Println("Vegas Dual Tunnel Trading Bot stopped")
}