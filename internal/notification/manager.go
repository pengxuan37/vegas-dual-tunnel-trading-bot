package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/config"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/strategy"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/telegram"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/trading"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
	"github.com/shopspring/decimal"
)

// NotificationManager 通知管理器
type NotificationManager struct {
	config      *config.Config
	logger      logger.Logger
	telegramBot *telegram.Bot
	mu          sync.RWMutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	queue       chan *Notification
	workers     int
}

// NotificationType 通知类型
type NotificationType int

const (
	NotificationInfo NotificationType = iota
	NotificationWarning
	NotificationError
	NotificationTrade
	NotificationSignal
	NotificationSystem
)

// NotificationPriority 通知优先级
type NotificationPriority int

const (
	PriorityLow NotificationPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// Notification 通知消息
type Notification struct {
	Type      NotificationType
	Priority  NotificationPriority
	Title     string
	Message   string
	Data      interface{}
	Timestamp time.Time
	ChatIDs   []int64 // 指定发送的聊天ID，为空则发送给所有配置的聊天
}

// TradeNotificationData 交易通知数据
type TradeNotificationData struct {
	Symbol      string
	Side        string
	Quantity    decimal.Decimal
	Price       decimal.Decimal
	OrderID     string
	Status      string
	Profit      decimal.Decimal
	ProfitRate  decimal.Decimal
}

// SignalNotificationData 信号通知数据
type SignalNotificationData struct {
	Symbol     string
	SignalType string
	Price      decimal.Decimal
	Confidence float64
	Reason     string
}

// New 创建新的通知管理器
func New(cfg *config.Config, log logger.Logger, bot *telegram.Bot) *NotificationManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &NotificationManager{
		config:      cfg,
		logger:      log,
		telegramBot: bot,
		ctx:         ctx,
		cancel:      cancel,
		queue:       make(chan *Notification, 1000), // 缓冲队列
		workers:     3,                              // 工作协程数量
	}
}

// Start 启动通知管理器
func (nm *NotificationManager) Start() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.running {
		return fmt.Errorf("notification manager is already running")
	}

	// 启动工作协程
	for i := 0; i < nm.workers; i++ {
		go nm.worker(i)
	}

	nm.running = true
	nm.logger.Info("Notification manager started successfully")

	return nil
}

// Stop 停止通知管理器
func (nm *NotificationManager) Stop() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.running {
		return nil
	}

	// 取消上下文
	nm.cancel()

	// 关闭队列
	close(nm.queue)

	nm.running = false
	nm.logger.Info("Notification manager stopped")

	return nil
}

// SendNotification 发送通知
func (nm *NotificationManager) SendNotification(notification *Notification) error {
	nm.mu.RLock()
	running := nm.running
	nm.mu.RUnlock()

	if !running {
		return fmt.Errorf("notification manager is not running")
	}

	notification.Timestamp = time.Now()

	select {
	case nm.queue <- notification:
		return nil
	default:
		nm.logger.Warn("Notification queue is full, dropping message")
		return fmt.Errorf("notification queue is full")
	}
}

// SendTradeNotification 发送交易通知
func (nm *NotificationManager) SendTradeNotification(trade *trading.TradeResult) error {
	data := &TradeNotificationData{
		Symbol:   trade.Symbol,
		Side:     trade.Side,
		Quantity: trade.Quantity,
		Price:    trade.Price,
		OrderID:  trade.OrderID,
		Status:   trade.Status,
	}

	var title, message string
	var priority NotificationPriority

	switch trade.Status {
	case "FILLED":
		title = "🎯 交易执行成功"
		message = nm.formatTradeMessage(data)
		priority = PriorityHigh
	case "PARTIALLY_FILLED":
		title = "⏳ 交易部分成交"
		message = nm.formatTradeMessage(data)
		priority = PriorityNormal
	case "CANCELED":
		title = "❌ 交易已取消"
		message = nm.formatTradeMessage(data)
		priority = PriorityNormal
	case "REJECTED":
		title = "🚫 交易被拒绝"
		message = nm.formatTradeMessage(data)
		priority = PriorityHigh
	default:
		title = "📊 交易状态更新"
		message = nm.formatTradeMessage(data)
		priority = PriorityNormal
	}

	notification := &Notification{
		Type:     NotificationTrade,
		Priority: priority,
		Title:    title,
		Message:  message,
		Data:     data,
	}

	return nm.SendNotification(notification)
}

// SendSignalNotification 发送信号通知
func (nm *NotificationManager) SendSignalNotification(signal *strategy.TradingSignal) error {
	data := &SignalNotificationData{
		Symbol:     signal.Symbol,
		SignalType: nm.signalTypeToString(signal.Type),
		Price:      signal.Price,
		Confidence: signal.Confidence,
		Reason:     signal.Reason,
	}

	var title string
	var priority NotificationPriority

	switch signal.Type {
	case strategy.SignalBuy:
		title = "📈 买入信号"
		priority = PriorityHigh
	case strategy.SignalSell:
		title = "📉 卖出信号"
		priority = PriorityHigh
	case strategy.SignalStopLoss:
		title = "🛑 止损信号"
		priority = PriorityCritical
	case strategy.SignalTakeProfit:
		title = "💰 止盈信号"
		priority = PriorityHigh
	default:
		title = "📊 交易信号"
		priority = PriorityNormal
	}

	message := nm.formatSignalMessage(data)

	notification := &Notification{
		Type:     NotificationSignal,
		Priority: priority,
		Title:    title,
		Message:  message,
		Data:     data,
	}

	return nm.SendNotification(notification)
}

// SendSystemNotification 发送系统通知
func (nm *NotificationManager) SendSystemNotification(level string, title, message string) error {
	var notificationType NotificationType
	var priority NotificationPriority

	switch level {
	case "info":
		notificationType = NotificationInfo
		priority = PriorityLow
	case "warning":
		notificationType = NotificationWarning
		priority = PriorityNormal
	case "error":
		notificationType = NotificationError
		priority = PriorityHigh
	default:
		notificationType = NotificationSystem
		priority = PriorityNormal
	}

	notification := &Notification{
		Type:     notificationType,
		Priority: priority,
		Title:    title,
		Message:  message,
	}

	return nm.SendNotification(notification)
}

// worker 工作协程
func (nm *NotificationManager) worker(id int) {
	nm.logger.Debugf("Notification worker %d started", id)

	for {
		select {
		case <-nm.ctx.Done():
			nm.logger.Debugf("Notification worker %d stopped", id)
			return
		case notification, ok := <-nm.queue:
			if !ok {
				nm.logger.Debugf("Notification worker %d stopped (queue closed)", id)
				return
			}

			if err := nm.processNotification(notification); err != nil {
				nm.logger.Errorf("Worker %d failed to process notification: %v", id, err)
			}
		}
	}
}

// processNotification 处理通知
func (nm *NotificationManager) processNotification(notification *Notification) error {
	// 格式化消息
	fullMessage := nm.formatNotificationMessage(notification)

	// 确定发送目标
	chatIDs := notification.ChatIDs
	if len(chatIDs) == 0 {
		chatIDs = nm.config.Telegram.ChatIDs
	}

	// 发送到所有目标聊天
	for _, chatID := range chatIDs {
		if err := nm.telegramBot.SendMessageToChat(chatID, fullMessage); err != nil {
			nm.logger.Errorf("Failed to send notification to chat %d: %v", chatID, err)
			continue
		}
	}

	nm.logger.Debugf("Notification sent: %s", notification.Title)
	return nil
}

// formatNotificationMessage 格式化通知消息
func (nm *NotificationManager) formatNotificationMessage(notification *Notification) string {
	timeStr := notification.Timestamp.Format("2006-01-02 15:04:05")
	priorityIcon := nm.getPriorityIcon(notification.Priority)

	return fmt.Sprintf("%s %s\n\n%s\n\n⏰ %s",
		priorityIcon,
		notification.Title,
		notification.Message,
		timeStr,
	)
}

// formatTradeMessage 格式化交易消息
func (nm *NotificationManager) formatTradeMessage(data *TradeNotificationData) string {
	message := fmt.Sprintf("交易对: %s\n", data.Symbol)
	message += fmt.Sprintf("方向: %s\n", data.Side)
	message += fmt.Sprintf("数量: %s\n", data.Quantity.String())
	message += fmt.Sprintf("价格: %s\n", data.Price.String())
	message += fmt.Sprintf("订单ID: %s\n", data.OrderID)
	message += fmt.Sprintf("状态: %s", data.Status)

	if !data.Profit.IsZero() {
		message += fmt.Sprintf("\n盈亏: %s USDT", data.Profit.String())
	}

	if !data.ProfitRate.IsZero() {
		message += fmt.Sprintf("\n盈亏率: %s%%", data.ProfitRate.String())
	}

	return message
}

// formatTradeNotification 格式化交易通知
func (nm *NotificationManager) formatTradeNotification(trade *trading.TradeResult) string {
	var status string
	if trade.Success {
		status = "✅ 成功"
	} else {
		status = "❌ 失败"
	}

	return fmt.Sprintf(
		"🔔 *交易通知*\n\n"+
			"交易对: %s\n"+
			"方向: %s\n"+
			"数量: %s\n"+
			"价格: %s\n"+
			"状态: %s\n"+
			"时间: %s",
		trade.Symbol,
		trade.Side,
		trade.Quantity.String(),
		trade.Price.String(),
		status,
		trade.ExecutedAt.Format("2006-01-02 15:04:05"),
	)
}

// formatSignalMessage 格式化信号消息
func (nm *NotificationManager) formatSignalMessage(data *SignalNotificationData) string {
	message := fmt.Sprintf("交易对: %s\n", data.Symbol)
	message += fmt.Sprintf("信号类型: %s\n", data.SignalType)
	message += fmt.Sprintf("价格: %s\n", data.Price.String())
	message += fmt.Sprintf("置信度: %.2f%%\n", data.Confidence*100)
	message += fmt.Sprintf("原因: %s", data.Reason)

	return message
}

// getPriorityIcon 获取优先级图标
func (nm *NotificationManager) getPriorityIcon(priority NotificationPriority) string {
	switch priority {
	case PriorityLow:
		return "ℹ️"
	case PriorityNormal:
		return "📢"
	case PriorityHigh:
		return "⚠️"
	case PriorityCritical:
		return "🚨"
	default:
		return "📢"
	}
}

// signalTypeToString 信号类型转字符串
func (nm *NotificationManager) signalTypeToString(signalType strategy.SignalType) string {
	switch signalType {
	case strategy.SignalBuy:
		return "买入"
	case strategy.SignalSell:
		return "卖出"
	case strategy.SignalStopLoss:
		return "止损"
	case strategy.SignalTakeProfit:
		return "止盈"
	default:
		return "未知"
	}
}

// IsRunning 检查是否正在运行
func (nm *NotificationManager) IsRunning() bool {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.running
}

// GetQueueSize 获取队列大小
func (nm *NotificationManager) GetQueueSize() int {
	return len(nm.queue)
}