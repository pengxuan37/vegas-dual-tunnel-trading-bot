package trading

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/binance"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/database"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/strategy"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
)

// TradeExecutor 交易执行器
type TradeExecutor struct {
	logger         logger.Logger
	binanceClient  *binance.Client
	db             *database.Database
	tradeRepo      *database.TradeRepository
	positionRepo   *database.PositionRepository
	userConfigRepo *database.UserConfigRepository
	mu             sync.RWMutex
	isRunning      bool
	ctx            context.Context
	cancel         context.CancelFunc
	activeOrders   map[string]*ActiveOrder
	positions      map[string]*Position
}

// ActiveOrder 活跃订单
type ActiveOrder struct {
	ID            string
	UserID        int64
	Symbol        string
	Side          string
	Type          string
	Quantity      decimal.Decimal
	Price         decimal.Decimal
	StopPrice     decimal.Decimal
	Status        string
	StrategyType  string
	SignalType    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Position 持仓信息
type Position struct {
	UserID          int64
	Symbol          string
	Side            string
	Size            decimal.Decimal
	EntryPrice      decimal.Decimal
	MarkPrice       decimal.Decimal
	UnrealizedPnl   decimal.Decimal
	StopLossPrice   decimal.Decimal
	TakeProfitPrice decimal.Decimal
	StrategyType    string
	IsOpen          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TradeRequest 交易请求
type TradeRequest struct {
	UserID       int64
	Symbol       string
	Signal       *strategy.TradingSignal
	Quantity     decimal.Decimal
	StrategyType string
}

// TradeResult 交易结果
type TradeResult struct {
	Success      bool
	OrderID      string
	Symbol       string
	Side         string
	Quantity     decimal.Decimal
	Price        decimal.Decimal
	Status       string
	Message      string
	Error        error
	ExecutedAt   time.Time
}

// NewTradeExecutor 创建新的交易执行器
func NewTradeExecutor(log logger.Logger, client *binance.Client, db *database.Database) *TradeExecutor {
	ctx, cancel := context.WithCancel(context.Background())

	return &TradeExecutor{
		logger:         log,
		binanceClient:  client,
		db:             db,
		tradeRepo:      database.NewTradeRepository(db.GetDB()),
		positionRepo:   database.NewPositionRepository(db.GetDB()),
		userConfigRepo: database.NewUserConfigRepository(db.GetDB()),
		ctx:            ctx,
		cancel:         cancel,
		activeOrders:   make(map[string]*ActiveOrder),
		positions:      make(map[string]*Position),
		isRunning:      false,
	}
}

// Start 启动交易执行器
func (te *TradeExecutor) Start() error {
	te.mu.Lock()
	defer te.mu.Unlock()

	if te.isRunning {
		return fmt.Errorf("trade executor is already running")
	}

	te.isRunning = true
	te.logger.Info("Trade executor started")

	// 启动订单监控
	go te.monitorOrders()

	// 启动持仓监控
	go te.monitorPositions()

	return nil
}

// Stop 停止交易执行器
func (te *TradeExecutor) Stop() {
	te.mu.Lock()
	defer te.mu.Unlock()

	if !te.isRunning {
		return
	}

	te.isRunning = false
	te.cancel()
	te.logger.Info("Trade executor stopped")
}

// ExecuteTrade 执行交易
func (te *TradeExecutor) ExecuteTrade(request *TradeRequest) *TradeResult {
	result := &TradeResult{
		ExecutedAt: time.Now(),
	}

	// 验证用户配置
	userConfig, err := te.userConfigRepo.GetByUserID(request.UserID)
	if err != nil {
		result.Error = fmt.Errorf("failed to get user config: %w", err)
		return result
	}

	if userConfig == nil {
		result.Error = fmt.Errorf("user config not found")
		return result
	}

	if !userConfig.IsActive {
		result.Error = fmt.Errorf("user trading is disabled")
		return result
	}

	// 验证交易信号
	if request.Signal == nil {
		result.Error = fmt.Errorf("trading signal is required")
		return result
	}

	// 计算交易数量
	if request.Quantity.IsZero() {
		quantity, err := te.calculateQuantity(userConfig, request.Symbol, request.Signal.Price)
		if err != nil {
			result.Error = fmt.Errorf("failed to calculate quantity: %w", err)
			return result
		}
		request.Quantity = quantity
	}

	// 执行不同类型的交易
	switch request.Signal.Type {
	case strategy.SignalBuy:
		return te.executeBuyOrder(request)
	case strategy.SignalSell:
		return te.executeSellOrder(request)
	case strategy.SignalStopLoss:
		return te.executeStopLoss(request)
	case strategy.SignalTakeProfit:
		return te.executeTakeProfit(request)
	default:
		result.Error = fmt.Errorf("unsupported signal type: %v", request.Signal.Type)
		return result
	}
}

// executeBuyOrder 执行买入订单
func (te *TradeExecutor) executeBuyOrder(request *TradeRequest) *TradeResult {
	result := &TradeResult{ExecutedAt: time.Now()}

	// 构建订单请求
	orderReq := &binance.OrderRequest{
		Symbol:      request.Symbol,
		Side:        "BUY",
		Type:        "MARKET",
		Quantity:    request.Quantity.String(),
		TimeInForce: "GTC",
	}

	// 发送订单
	orderResp, err := te.binanceClient.PlaceOrder(orderReq)
	if err != nil {
		result.Error = fmt.Errorf("failed to place buy order: %w", err)
		return result
	}

	// 记录交易
	trade := &database.Trade{
		UserID:        request.UserID,
		Symbol:        request.Symbol,
		OrderID:       fmt.Sprintf("%d", orderResp.OrderID),
		ClientOrderID: orderResp.ClientOrderID,
		Side:          "BUY",
		Type:          "MARKET",
		Quantity:      request.Quantity.InexactFloat64(),
		Price:         request.Signal.Price.InexactFloat64(),
		Status:        orderResp.Status,
		StrategyType:  request.StrategyType,
		SignalType:    "entry",
	}

	if err := te.tradeRepo.Create(trade); err != nil {
		te.logger.Errorf("Failed to save trade record: %v", err)
	}

	// 设置止损止盈订单
	if !request.Signal.StopLoss.IsZero() || !request.Signal.TakeProfit.IsZero() {
		go te.setStopLossAndTakeProfit(request, fmt.Sprintf("%d", orderResp.OrderID))
	}

	result.Success = true
	result.OrderID = fmt.Sprintf("%d", orderResp.OrderID)
	result.Message = fmt.Sprintf("Buy order placed successfully: %s", orderResp.OrderID)

	te.logger.Infof("Buy order executed: %s, Quantity: %s, Price: %s", 
		orderResp.OrderID, request.Quantity.String(), request.Signal.Price.String())

	return result
}

// executeSellOrder 执行卖出订单
func (te *TradeExecutor) executeSellOrder(request *TradeRequest) *TradeResult {
	result := &TradeResult{ExecutedAt: time.Now()}

	// 构建订单请求
	orderReq := &binance.OrderRequest{
		Symbol:      request.Symbol,
		Side:        "SELL",
		Type:        "MARKET",
		Quantity:    request.Quantity.String(),
		TimeInForce: "GTC",
	}

	// 发送订单
	orderResp, err := te.binanceClient.PlaceOrder(orderReq)
	if err != nil {
		result.Error = fmt.Errorf("failed to place sell order: %w", err)
		return result
	}

	// 记录交易
	trade := &database.Trade{
		UserID:        request.UserID,
		Symbol:        request.Symbol,
		OrderID:       fmt.Sprintf("%d", orderResp.OrderID),
		ClientOrderID: orderResp.ClientOrderID,
		Side:          "SELL",
		Type:          "MARKET",
		Quantity:      request.Quantity.InexactFloat64(),
		Price:         request.Signal.Price.InexactFloat64(),
		Status:        orderResp.Status,
		StrategyType:  request.StrategyType,
		SignalType:    "entry",
	}

	if err := te.tradeRepo.Create(trade); err != nil {
		te.logger.Errorf("Failed to save trade record: %v", err)
	}

	// 设置止损止盈订单
	if !request.Signal.StopLoss.IsZero() || !request.Signal.TakeProfit.IsZero() {
		go te.setStopLossAndTakeProfit(request, fmt.Sprintf("%d", orderResp.OrderID))
	}

	result.Success = true
	result.OrderID = fmt.Sprintf("%d", orderResp.OrderID)
	result.Message = fmt.Sprintf("Sell order placed successfully: %d", orderResp.OrderID)

	te.logger.Infof("Sell order executed: %d, Quantity: %s, Price: %s", 
		orderResp.OrderID, request.Quantity.String(), request.Signal.Price.String())

	return result
}

// executeStopLoss 执行止损订单
func (te *TradeExecutor) executeStopLoss(request *TradeRequest) *TradeResult {
	result := &TradeResult{ExecutedAt: time.Now()}

	// 构建止损订单请求
	orderReq := &binance.OrderRequest{
		Symbol:      request.Symbol,
		Side:        te.getOppositeSide(request.Signal),
		Type:        "STOP_MARKET",
		Quantity:    request.Quantity.String(),
		StopPrice:   request.Signal.StopLoss.String(),
		TimeInForce: "GTC",
	}

	// 发送订单
	orderResp, err := te.binanceClient.PlaceOrder(orderReq)
	if err != nil {
		result.Error = fmt.Errorf("failed to place stop loss order: %w", err)
		return result
	}

	// 记录交易
	trade := &database.Trade{
		UserID:        request.UserID,
		Symbol:        request.Symbol,
		OrderID:       fmt.Sprintf("%d", orderResp.OrderID),
		ClientOrderID: orderResp.ClientOrderID,
		Side:          orderReq.Side,
		Type:          "STOP_MARKET",
		Quantity:      request.Quantity.InexactFloat64(),
		StopPrice:     request.Signal.StopLoss.InexactFloat64(),
		Status:        orderResp.Status,
		StrategyType:  request.StrategyType,
		SignalType:    "stop_loss",
	}

	if err := te.tradeRepo.Create(trade); err != nil {
		te.logger.Errorf("Failed to save trade record: %v", err)
	}

	result.Success = true
	result.OrderID = fmt.Sprintf("%d", orderResp.OrderID)
	result.Message = fmt.Sprintf("Stop loss order placed successfully: %d", orderResp.OrderID)

	te.logger.Infof("Stop loss order executed: %d, Stop Price: %s", 
		orderResp.OrderID, request.Signal.StopLoss.String())

	return result
}

// executeTakeProfit 执行止盈订单
func (te *TradeExecutor) executeTakeProfit(request *TradeRequest) *TradeResult {
	result := &TradeResult{ExecutedAt: time.Now()}

	// 构建止盈订单请求
	orderReq := &binance.OrderRequest{
		Symbol:      request.Symbol,
		Side:        te.getOppositeSide(request.Signal),
		Type:        "LIMIT",
		Quantity:    request.Quantity.String(),
		Price:       request.Signal.TakeProfit.String(),
		TimeInForce: "GTC",
	}

	// 发送订单
	orderResp, err := te.binanceClient.PlaceOrder(orderReq)
	if err != nil {
		result.Error = fmt.Errorf("failed to place take profit order: %w", err)
		return result
	}

	// 记录交易
	trade := &database.Trade{
		UserID:        request.UserID,
		Symbol:        request.Symbol,
		OrderID:       fmt.Sprintf("%d", orderResp.OrderID),
		ClientOrderID: orderResp.ClientOrderID,
		Side:          orderReq.Side,
		Type:          "LIMIT",
		Quantity:      request.Quantity.InexactFloat64(),
		Price:         request.Signal.TakeProfit.InexactFloat64(),
		Status:        orderResp.Status,
		StrategyType:  request.StrategyType,
		SignalType:    "take_profit",
	}

	if err := te.tradeRepo.Create(trade); err != nil {
		te.logger.Errorf("Failed to save trade record: %v", err)
	}

	result.Success = true
	result.OrderID = fmt.Sprintf("%d", orderResp.OrderID)
	result.Message = fmt.Sprintf("Take profit order placed successfully: %d", orderResp.OrderID)

	te.logger.Infof("Take profit order executed: %d, Price: %s", 
		orderResp.OrderID, request.Signal.TakeProfit.String())

	return result
}

// setStopLossAndTakeProfit 设置止损止盈订单
func (te *TradeExecutor) setStopLossAndTakeProfit(request *TradeRequest, parentOrderID string) {
	// 等待主订单成交
	time.Sleep(2 * time.Second)

	// 设置止损订单
	if !request.Signal.StopLoss.IsZero() {
		stopLossReq := &TradeRequest{
			UserID:       request.UserID,
			Symbol:       request.Symbol,
			Quantity:     request.Quantity,
			StrategyType: request.StrategyType,
			Signal: &strategy.TradingSignal{
				Type:     strategy.SignalStopLoss,
				StopLoss: request.Signal.StopLoss,
			},
		}
		te.ExecuteTrade(stopLossReq)
	}

	// 设置止盈订单
	if !request.Signal.TakeProfit.IsZero() {
		takeProfitReq := &TradeRequest{
			UserID:       request.UserID,
			Symbol:       request.Symbol,
			Quantity:     request.Quantity,
			StrategyType: request.StrategyType,
			Signal: &strategy.TradingSignal{
				Type:       strategy.SignalTakeProfit,
				TakeProfit: request.Signal.TakeProfit,
			},
		}
		te.ExecuteTrade(takeProfitReq)
	}
}

// calculateQuantity 计算交易数量
func (te *TradeExecutor) calculateQuantity(userConfig *database.UserConfig, symbol string, price decimal.Decimal) (decimal.Decimal, error) {
	// 获取账户信息
	accountInfo, err := te.binanceClient.GetAccountInfo()
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get account info: %w", err)
	}

	// 查找USDT余额
	var usdtBalance decimal.Decimal
	for _, asset := range accountInfo.Assets {
		if asset.Asset == "USDT" {
			free, err := decimal.NewFromString(asset.AvailableBalance)
			if err != nil {
				return decimal.Zero, fmt.Errorf("invalid balance format: %w", err)
			}
			usdtBalance = free
			break
		}
	}

	if usdtBalance.IsZero() {
		return decimal.Zero, fmt.Errorf("insufficient USDT balance")
	}

	// 计算风险金额
	riskAmount := usdtBalance.Mul(decimal.NewFromFloat(userConfig.RiskPercentage / 100))

	// 限制最大仓位大小
	maxPositionValue := decimal.NewFromFloat(userConfig.MaxPositionSize)
	if riskAmount.GreaterThan(maxPositionValue) {
		riskAmount = maxPositionValue
	}

	// 计算数量
	quantity := riskAmount.Div(price)

	// 确保数量不为零
	if quantity.LessThan(decimal.NewFromFloat(0.001)) {
		return decimal.Zero, fmt.Errorf("calculated quantity too small")
	}

	return quantity, nil
}

// getOppositeSide 获取相反的交易方向
func (te *TradeExecutor) getOppositeSide(signal *strategy.TradingSignal) string {
	if signal.Type == strategy.SignalBuy {
		return "SELL"
	}
	return "BUY"
}

// monitorOrders 监控订单状态
func (te *TradeExecutor) monitorOrders() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-te.ctx.Done():
			return
		case <-ticker.C:
			te.updateOrderStatus()
		}
	}
}

// monitorPositions 监控持仓状态
func (te *TradeExecutor) monitorPositions() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-te.ctx.Done():
			return
		case <-ticker.C:
			te.updatePositionStatus()
		}
	}
}

// updateOrderStatus 更新订单状态
func (te *TradeExecutor) updateOrderStatus() {
	// TODO: 实现订单状态更新逻辑
	te.logger.Debug("Updating order status...")
}

// updatePositionStatus 更新持仓状态
func (te *TradeExecutor) updatePositionStatus() {
	// TODO: 实现持仓状态更新逻辑
	te.logger.Debug("Updating position status...")
}

// CancelOrder 取消订单
func (te *TradeExecutor) CancelOrder(symbol, orderID string) error {
	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid order ID format: %w", err)
	}
	err = te.binanceClient.CancelOrder(symbol, orderIDInt)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	te.logger.Infof("Order cancelled: %s", orderID)
	return nil
}

// GetActiveOrders 获取活跃订单
func (te *TradeExecutor) GetActiveOrders() map[string]*ActiveOrder {
	te.mu.RLock()
	defer te.mu.RUnlock()

	orders := make(map[string]*ActiveOrder)
	for k, v := range te.activeOrders {
		orders[k] = v
	}
	return orders
}

// GetPositions 获取持仓信息
func (te *TradeExecutor) GetPositions() map[string]*Position {
	te.mu.RLock()
	defer te.mu.RUnlock()

	positions := make(map[string]*Position)
	for k, v := range te.positions {
		positions[k] = v
	}
	return positions
}

// IsRunning 检查是否正在运行
func (te *TradeExecutor) IsRunning() bool {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.isRunning
}