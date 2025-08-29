package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/binance"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/config"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/database"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/notification"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/stream"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/strategy"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/telegram"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/trading"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
)

// App 应用主结构
type App struct {
	config            *config.Config
	logger            logger.Logger
	db                *database.Database
	telegramBot       *telegram.Bot
	binanceClient     *binance.Client
	binanceWSClient   *binance.WebSocketClient
	strategyManager   *strategy.StrategyManager
	tradeExecutor     *trading.TradeExecutor
	streamManager     *stream.StreamManager
	notificationMgr   *notification.NotificationManager
	mu                sync.RWMutex
	isRunning         bool
}

// New 创建新的应用实例
func New(cfg *config.Config, log logger.Logger) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	if log == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	app := &App{
		config: cfg,
		logger: log,
	}

	// 初始化数据库
	db, err := database.New(cfg.Database.Path, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	app.db = db

	// 初始化Telegram机器人
	telegramBot, err := telegram.New(&cfg.Telegram, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telegram bot: %w", err)
	}
	app.telegramBot = telegramBot

	// 初始化Binance客户端
	binanceClient, err := binance.New(&cfg.Binance, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize binance client: %w", err)
	}
	app.binanceClient = binanceClient

	// 初始化Binance WebSocket客户端
	binanceWSClient, err := binance.NewWebSocketClient(cfg.GetBinanceWSURL(), log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize binance websocket client: %w", err)
	}
	app.binanceWSClient = binanceWSClient

	// 初始化策略管理器
	strategyManager := strategy.NewStrategyManager(log)
	app.strategyManager = strategyManager

	// 初始化交易执行器
	tradeExecutor := trading.NewTradeExecutor(log, binanceClient, db)
	app.tradeExecutor = tradeExecutor

	// 初始化通知管理器
	notificationMgr := notification.New(cfg, log, telegramBot)
	app.notificationMgr = notificationMgr

	// 初始化流管理器
	streamManager, err := stream.New(cfg, log, strategyManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stream manager: %w", err)
	}
	app.streamManager = streamManager

	return app, nil
}

// Run 运行应用
func (a *App) Run(ctx context.Context) error {
	a.mu.Lock()
	if a.isRunning {
		a.mu.Unlock()
		return fmt.Errorf("application is already running")
	}
	a.isRunning = true
	a.mu.Unlock()

	a.logger.Info("Application starting...")

	// 启动Telegram机器人
	if err := a.telegramBot.Start(ctx); err != nil {
		return fmt.Errorf("failed to start telegram bot: %w", err)
	}
	a.logger.Info("Telegram bot started")

	// 启动策略管理器
	if err := a.strategyManager.Start(); err != nil {
		return fmt.Errorf("failed to start strategy manager: %w", err)
	}
	a.logger.Info("Strategy manager started")

	// 启动交易执行器
	if err := a.tradeExecutor.Start(); err != nil {
		return fmt.Errorf("failed to start trade executor: %w", err)
	}
	a.logger.Info("Trade executor started")

	// 启动通知管理器
	if err := a.notificationMgr.Start(); err != nil {
		return fmt.Errorf("failed to start notification manager: %w", err)
	}
	a.logger.Info("Notification manager started")

	// 启动流管理器
	if err := a.streamManager.Start(); err != nil {
		return fmt.Errorf("failed to start stream manager: %w", err)
	}
	a.logger.Info("Stream manager started")

	// 注册维加斯双隧道策略
	vegasStrategy := strategy.NewVegasTunnelStrategy(a.logger)
	if err := a.strategyManager.RegisterStrategy("vegas_tunnel", vegasStrategy); err != nil {
		a.logger.Errorf("Failed to register vegas tunnel strategy: %v", err)
	} else {
		a.logger.Info("Vegas tunnel strategy registered")
	}

	a.logger.Info("Application started successfully")

	// 等待上下文取消
	<-ctx.Done()

	a.logger.Info("Application shutting down...")

	// 停止所有服务
	a.streamManager.Stop()
	a.logger.Info("Stream manager stopped")

	a.notificationMgr.Stop()
	a.logger.Info("Notification manager stopped")

	a.tradeExecutor.Stop()
	a.logger.Info("Trade executor stopped")

	a.strategyManager.Stop()
	a.logger.Info("Strategy manager stopped")

	a.telegramBot.Stop()
	a.logger.Info("Telegram bot stopped")

	// 关闭数据库连接
	if err := a.db.Close(); err != nil {
		a.logger.Errorf("Failed to close database: %v", err)
	} else {
		a.logger.Info("Database closed")
	}

	a.mu.Lock()
	a.isRunning = false
	a.mu.Unlock()

	a.logger.Info("Application shutdown completed")
	return nil
}