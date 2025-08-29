package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
)

// Strategy 策略接口
type Strategy interface {
	GenerateSignal(klines []KlineData) *TradingSignal
	GetStrategyInfo() map[string]interface{}
	ValidateParameters() error
}

// StrategyManager 策略管理器
type StrategyManager struct {
	logger     logger.Logger
	strategies map[string]Strategy
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	isRunning  bool
}

// StrategyResult 策略执行结果
type StrategyResult struct {
	StrategyName string
	Symbol       string
	Signal       *TradingSignal
	Error        error
	Timestamp    time.Time
}

// NewStrategyManager 创建新的策略管理器
func NewStrategyManager(log logger.Logger) *StrategyManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &StrategyManager{
		logger:     log,
		strategies: make(map[string]Strategy),
		ctx:        ctx,
		cancel:     cancel,
		isRunning:  false,
	}
}

// RegisterStrategy 注册策略
func (sm *StrategyManager) RegisterStrategy(name string, strategy Strategy) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.strategies[name]; exists {
		return fmt.Errorf("strategy %s already registered", name)
	}

	// 验证策略参数
	if err := strategy.ValidateParameters(); err != nil {
		return fmt.Errorf("strategy validation failed: %w", err)
	}

	sm.strategies[name] = strategy
	sm.logger.Infof("Strategy registered: %s", name)
	return nil
}

// UnregisterStrategy 注销策略
func (sm *StrategyManager) UnregisterStrategy(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.strategies[name]; !exists {
		return fmt.Errorf("strategy %s not found", name)
	}

	delete(sm.strategies, name)
	sm.logger.Infof("Strategy unregistered: %s", name)
	return nil
}

// GetStrategy 获取策略
func (sm *StrategyManager) GetStrategy(name string) (Strategy, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	strategy, exists := sm.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", name)
	}

	return strategy, nil
}

// ListStrategies 列出所有策略
func (sm *StrategyManager) ListStrategies() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	names := make([]string, 0, len(sm.strategies))
	for name := range sm.strategies {
		names = append(names, name)
	}

	return names
}

// ExecuteStrategy 执行单个策略
func (sm *StrategyManager) ExecuteStrategy(strategyName string, symbol string, klines []KlineData) *StrategyResult {
	result := &StrategyResult{
		StrategyName: strategyName,
		Symbol:       symbol,
		Timestamp:    time.Now(),
	}

	strategy, err := sm.GetStrategy(strategyName)
	if err != nil {
		result.Error = err
		return result
	}

	// 执行策略
	signal := strategy.GenerateSignal(klines)
	result.Signal = signal

	if signal != nil {
		sm.logger.Infof("Strategy %s generated signal for %s: %s at %.4f", 
			strategyName, symbol, sm.signalTypeToString(signal.Type), signal.Price)
	} else {
		sm.logger.Debugf("Strategy %s: no signal for %s", strategyName, symbol)
	}

	return result
}

// ExecuteAllStrategies 执行所有策略
func (sm *StrategyManager) ExecuteAllStrategies(symbol string, klines []KlineData) []*StrategyResult {
	sm.mu.RLock()
	strategyNames := make([]string, 0, len(sm.strategies))
	for name := range sm.strategies {
		strategyNames = append(strategyNames, name)
	}
	sm.mu.RUnlock()

	results := make([]*StrategyResult, 0, len(strategyNames))
	for _, name := range strategyNames {
		result := sm.ExecuteStrategy(name, symbol, klines)
		results = append(results, result)
	}

	return results
}

// ExecuteStrategiesAsync 异步执行所有策略
func (sm *StrategyManager) ExecuteStrategiesAsync(symbol string, klines []KlineData) <-chan *StrategyResult {
	resultChan := make(chan *StrategyResult, 10)

	go func() {
		defer close(resultChan)

		sm.mu.RLock()
		strategyNames := make([]string, 0, len(sm.strategies))
		for name := range sm.strategies {
			strategyNames = append(strategyNames, name)
		}
		sm.mu.RUnlock()

		var wg sync.WaitGroup
		for _, name := range strategyNames {
			wg.Add(1)
			go func(strategyName string) {
				defer wg.Done()
				result := sm.ExecuteStrategy(strategyName, symbol, klines)
				select {
				case resultChan <- result:
				case <-sm.ctx.Done():
					return
				}
			}(name)
		}
		wg.Wait()
	}()

	return resultChan
}

// GetStrategyInfo 获取策略信息
func (sm *StrategyManager) GetStrategyInfo(name string) (map[string]interface{}, error) {
	strategy, err := sm.GetStrategy(name)
	if err != nil {
		return nil, err
	}

	return strategy.GetStrategyInfo(), nil
}

// ProcessKlineData 处理K线数据
func (sm *StrategyManager) ProcessKlineData(klineData *KlineData) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 验证K线数据
	if err := sm.ValidateKlineData([]KlineData{*klineData}); err != nil {
		return fmt.Errorf("invalid kline data: %w", err)
	}

	// 对所有注册的策略执行分析
	for name, strategy := range sm.strategies {
		go func(strategyName string, s Strategy, data *KlineData) {
			if signal := s.GenerateSignal([]KlineData{*data}); signal != nil {
				sm.logger.Infof("Strategy %s generated signal: %s for %s", 
					strategyName, sm.signalTypeToString(signal.Type), data.Symbol)
				// 这里可以添加信号处理逻辑，比如发送到交易执行器
			}
		}(name, strategy, klineData)
	}

	return nil
}

// GetAllStrategyInfo 获取所有策略信息
func (sm *StrategyManager) GetAllStrategyInfo() map[string]map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	info := make(map[string]map[string]interface{})
	for name, strategy := range sm.strategies {
		info[name] = strategy.GetStrategyInfo()
	}

	return info
}

// Start 启动策略管理器
func (sm *StrategyManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning {
		return fmt.Errorf("strategy manager is already running")
	}

	sm.isRunning = true
	sm.logger.Info("Strategy manager started")
	return nil
}

// Stop 停止策略管理器
func (sm *StrategyManager) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isRunning {
		return
	}

	sm.isRunning = false
	sm.cancel()
	sm.logger.Info("Strategy manager stopped")
}

// IsRunning 检查是否正在运行
func (sm *StrategyManager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isRunning
}

// signalTypeToString 将信号类型转换为字符串
func (sm *StrategyManager) signalTypeToString(signalType SignalType) string {
	switch signalType {
	case SignalBuy:
		return "BUY"
	case SignalSell:
		return "SELL"
	case SignalStopLoss:
		return "STOP_LOSS"
	case SignalTakeProfit:
		return "TAKE_PROFIT"
	default:
		return "NONE"
	}
}

// FilterSignalsByConfidence 根据置信度过滤信号
func (sm *StrategyManager) FilterSignalsByConfidence(results []*StrategyResult, minConfidence float64) []*StrategyResult {
	filtered := make([]*StrategyResult, 0)
	for _, result := range results {
		if result.Signal != nil && result.Signal.Confidence >= minConfidence {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// GetBestSignal 获取最佳信号（置信度最高）
func (sm *StrategyManager) GetBestSignal(results []*StrategyResult) *StrategyResult {
	var bestResult *StrategyResult
	var maxConfidence float64

	for _, result := range results {
		if result.Signal != nil && result.Signal.Confidence > maxConfidence {
			maxConfidence = result.Signal.Confidence
			bestResult = result
		}
	}

	return bestResult
}

// ValidateKlineData 验证K线数据
func (sm *StrategyManager) ValidateKlineData(klines []KlineData) error {
	if len(klines) == 0 {
		return fmt.Errorf("empty kline data")
	}

	for i, kline := range klines {
		if kline.Open.IsZero() || kline.High.IsZero() || kline.Low.IsZero() || kline.Close.IsZero() {
			return fmt.Errorf("invalid price data at index %d", i)
		}

		if kline.High.LessThan(kline.Low) {
			return fmt.Errorf("high price less than low price at index %d", i)
		}

		if kline.High.LessThan(kline.Open) || kline.High.LessThan(kline.Close) {
			return fmt.Errorf("high price inconsistent at index %d", i)
		}

		if kline.Low.GreaterThan(kline.Open) || kline.Low.GreaterThan(kline.Close) {
			return fmt.Errorf("low price inconsistent at index %d", i)
		}
	}

	return nil
}