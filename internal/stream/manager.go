package stream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/binance"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/config"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/strategy"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
)

// StreamManager WebSocket数据流管理器
type StreamManager struct {
	config          *config.Config
	logger          logger.Logger
	binanceWS       *binance.WebSocketClient
	strategyManager *strategy.StrategyManager
	subscriptions   map[string]*Subscription
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         bool
}

// Subscription 订阅信息
type Subscription struct {
	Symbol    string
	Interval  string
	Active    bool
	CreatedAt time.Time
	LastData  time.Time
	Handlers  []DataHandler
}

// DataHandler 数据处理器接口
type DataHandler interface {
	HandleKlineData(data *binance.KlineStreamData) error
	HandleTickerData(data *binance.TickerStreamData) error
	GetName() string
}

// StrategyHandler 策略数据处理器
type StrategyHandler struct {
	strategyManager *strategy.StrategyManager
	logger          logger.Logger
}

// New 创建新的流管理器
func New(cfg *config.Config, log logger.Logger, strategyMgr *strategy.StrategyManager) (*StreamManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建币安WebSocket客户端
	binanceWS, err := binance.NewWebSocketClient(cfg.GetBinanceWSURL(), log)
	if err != nil {
		return nil, fmt.Errorf("failed to create binance websocket client: %w", err)
	}

	return &StreamManager{
		config:          cfg,
		logger:          log,
		binanceWS:       binanceWS,
		strategyManager: strategyMgr,
		subscriptions:   make(map[string]*Subscription),
		ctx:             ctx,
		cancel:          cancel,
		running:         false,
	}, nil
}

// Start 启动流管理器
func (sm *StreamManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("stream manager is already running")
	}

	// 设置数据处理器
	strategyHandler := &StrategyHandler{
		strategyManager: sm.strategyManager,
		logger:          sm.logger,
	}
	sm.binanceWS.SetStreamHandler(strategyHandler)

	// 启动WebSocket客户端
	if err := sm.binanceWS.Start(); err != nil {
		return fmt.Errorf("failed to start websocket client: %w", err)
	}

	// 启动监控协程
	sm.wg.Add(1)
	go sm.monitorConnections()

	sm.running = true
	sm.logger.Info("Stream manager started successfully")

	return nil
}

// Stop 停止流管理器
func (sm *StreamManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	// 取消上下文
	sm.cancel()

	// 停止WebSocket客户端
	sm.binanceWS.Stop()

	// 等待所有协程结束
	sm.wg.Wait()

	sm.running = false
	sm.logger.Info("Stream manager stopped")

	return nil
}

// Subscribe 订阅数据流
func (sm *StreamManager) Subscribe(symbol, interval string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := fmt.Sprintf("%s_%s", symbol, interval)

	// 检查是否已经订阅
	if sub, exists := sm.subscriptions[key]; exists {
		if sub.Active {
			return fmt.Errorf("already subscribed to %s %s", symbol, interval)
		}
		sub.Active = true
		sub.LastData = time.Now()
	} else {
		// 创建新订阅
		sm.subscriptions[key] = &Subscription{
			Symbol:    symbol,
			Interval:  interval,
			Active:    true,
			CreatedAt: time.Now(),
			LastData:  time.Now(),
			Handlers:  []DataHandler{},
		}
	}

	// 订阅K线数据
	if err := sm.binanceWS.SubscribeKline(symbol, interval); err != nil {
		return fmt.Errorf("failed to subscribe kline: %w", err)
	}

	// 订阅价格数据
	if err := sm.binanceWS.SubscribeTicker(symbol); err != nil {
		return fmt.Errorf("failed to subscribe ticker: %w", err)
	}

	sm.logger.Infof("Subscribed to %s %s", symbol, interval)
	return nil
}

// Unsubscribe 取消订阅
func (sm *StreamManager) Unsubscribe(symbol, interval string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := fmt.Sprintf("%s_%s", symbol, interval)

	if sub, exists := sm.subscriptions[key]; exists {
		sub.Active = false
		// 取消订阅K线数据
		if err := sm.binanceWS.UnsubscribeKline(symbol, interval); err != nil {
			sm.logger.Errorf("Failed to unsubscribe kline: %v", err)
		}
		// 取消订阅价格数据
		if err := sm.binanceWS.UnsubscribeTicker(symbol); err != nil {
			sm.logger.Errorf("Failed to unsubscribe ticker: %v", err)
		}
		sm.logger.Infof("Unsubscribed from %s %s", symbol, interval)
	}

	return nil
}

// GetSubscriptions 获取所有订阅
func (sm *StreamManager) GetSubscriptions() map[string]*Subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 创建副本
	result := make(map[string]*Subscription)
	for k, v := range sm.subscriptions {
		result[k] = &Subscription{
			Symbol:    v.Symbol,
			Interval:  v.Interval,
			Active:    v.Active,
			CreatedAt: v.CreatedAt,
			LastData:  v.LastData,
		}
	}

	return result
}

// IsRunning 检查是否正在运行
func (sm *StreamManager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.running
}

// monitorConnections 监控连接状态
func (sm *StreamManager) monitorConnections() {
	defer sm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.checkSubscriptionHealth()
		}
	}
}

// checkSubscriptionHealth 检查订阅健康状态
func (sm *StreamManager) checkSubscriptionHealth() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for key, sub := range sm.subscriptions {
		if sub.Active && now.Sub(sub.LastData) > 2*time.Minute {
			sm.logger.Warnf("No data received for %s in %v, attempting reconnection", key, now.Sub(sub.LastData))
			// 尝试重新订阅
			go func(symbol, interval string) {
				if err := sm.binanceWS.SubscribeKline(symbol, interval); err != nil {
					sm.logger.Errorf("Failed to resubscribe kline %s %s: %v", symbol, interval, err)
				}
				if err := sm.binanceWS.SubscribeTicker(symbol); err != nil {
					sm.logger.Errorf("Failed to resubscribe ticker %s: %v", symbol, err)
				}
			}(sub.Symbol, sub.Interval)
		}
	}
}

// updateLastDataTime 更新最后数据时间
func (sm *StreamManager) updateLastDataTime(symbol, interval string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := fmt.Sprintf("%s_%s", symbol, interval)
	if sub, exists := sm.subscriptions[key]; exists {
		sub.LastData = time.Now()
	}
}

// StrategyHandler 实现 StreamHandler 接口

// HandleKlineData 处理K线数据
func (sh *StrategyHandler) HandleKlineData(data *binance.KlineStreamData) error {
	if data == nil {
		return fmt.Errorf("received nil kline data")
	}

	// 转换价格字符串为 decimal.Decimal
	open, _ := decimal.NewFromString(data.Data.Kline.Open)
	high, _ := decimal.NewFromString(data.Data.Kline.High)
	low, _ := decimal.NewFromString(data.Data.Kline.Low)
	close, _ := decimal.NewFromString(data.Data.Kline.Close)
	volume, _ := decimal.NewFromString(data.Data.Kline.Volume)

	// 转换为策略所需的K线数据格式
	klineData := &strategy.KlineData{
		Symbol:    data.Data.Symbol,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
		Timestamp: time.Unix(data.Data.Kline.StartTime/1000, 0),
	}

	// 只处理已关闭的K线
	if !data.Data.Kline.IsClosed {
		return nil
	}

	// 执行策略分析
	if err := sh.strategyManager.ProcessKlineData(klineData); err != nil {
		sh.logger.Errorf("Failed to process kline data for %s: %v", data.Data.Symbol, err)
		return err
	}

	sh.logger.Debugf("Processed kline data for %s: %s", data.Data.Symbol, klineData.Close.String())
	return nil
}

// HandleTickerData 处理价格数据
func (sh *StrategyHandler) HandleTickerData(data *binance.TickerStreamData) error {
	if data == nil {
		return fmt.Errorf("received nil ticker data")
	}

	sh.logger.Debugf("Received ticker data for %s: %s", data.Data.Symbol, data.Data.LastPrice)
	return nil
}

// GetName 获取处理器名称
func (sh *StrategyHandler) GetName() string {
	return "StrategyHandler"
}