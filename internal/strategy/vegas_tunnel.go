package strategy

import (
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
)

// VegasTunnelStrategy 维加斯双隧道策略
type VegasTunnelStrategy struct {
	logger           logger.Logger
	// 核心EMA指标
	shortEMAPeriod   int     // 短期动能线EMA，默认12
	midTunnel1Period int     // 中期隧道1 EMA，默认144
	midTunnel2Period int     // 中期隧道2 EMA，默认169
	longTunnel1Period int    // 长期隧道1 EMA，默认288
	longTunnel2Period int    // 长期隧道2 EMA，默认338
	// 策略参数
	minTunnelPeriod  int     // 最小隧道持续周期，默认3
	volumeFactor     float64 // 成交量确认因子，默认1.5
	riskRewardRatio  float64 // 风险收益比，默认2:1
	stopLossPercent  float64 // 止损百分比，默认2%
	takeProfitPercent float64 // 止盈百分比，默认4%
	// 多时间周期数据缓存
	kline15MData     []KlineData // 15分钟K线数据
	kline4HData      []KlineData // 4小时K线数据
}

// KlineData K线数据结构
type KlineData struct {
	Symbol    string
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
	Timestamp time.Time
}

// TunnelData 隧道数据
type TunnelData struct {
	EMA12       decimal.Decimal
	EMA144      decimal.Decimal
	EMA169      decimal.Decimal
	EMA288      decimal.Decimal
	EMA338      decimal.Decimal
	MidTunnelUpper  decimal.Decimal // EMA144和EMA169的上边界
	MidTunnelLower  decimal.Decimal // EMA144和EMA169的下边界
	LongTunnelUpper decimal.Decimal // EMA288和EMA338的上边界
	LongTunnelLower decimal.Decimal // EMA288和EMA338的下边界
	TrendDirection  TrendDirection
}

// TrendDirection 趋势方向
type TrendDirection int

const (
	TrendNone TrendDirection = iota
	TrendBullish  // 多头趋势
	TrendBearish  // 空头趋势
	TrendSideways // 震荡趋势
)

// SignalType 信号类型
type SignalType int

const (
	SignalNone SignalType = iota
	SignalBuy
	SignalSell
	SignalStopLoss
	SignalTakeProfit
)

// TradingSignal 交易信号
type TradingSignal struct {
	Symbol      string
	Type        SignalType
	Price       decimal.Decimal
	StopLoss    decimal.Decimal
	TakeProfit  decimal.Decimal
	Confidence  float64
	Reason      string
	Timestamp   time.Time
	Timeframe   string // "15M" 或 "4H"
}

// NewVegasTunnelStrategy 创建新的维加斯隧道策略实例
func NewVegasTunnelStrategy(log logger.Logger) *VegasTunnelStrategy {
	return &VegasTunnelStrategy{
		logger:            log,
		shortEMAPeriod:    12,
		midTunnel1Period:  144,
		midTunnel2Period:  169,
		longTunnel1Period: 288,
		longTunnel2Period: 338,
		minTunnelPeriod:   3,
		volumeFactor:      1.5,
		riskRewardRatio:   2.0,
		stopLossPercent:   0.02, // 2%
		takeProfitPercent: 0.04, // 4%
		kline15MData:      make([]KlineData, 0),
		kline4HData:       make([]KlineData, 0),
	}
}

// SetParameters 设置策略参数
func (v *VegasTunnelStrategy) SetParameters(shortEMA, midTunnel1, midTunnel2, longTunnel1, longTunnel2 int, stopLoss, takeProfit float64) {
	v.shortEMAPeriod = shortEMA
	v.midTunnel1Period = midTunnel1
	v.midTunnel2Period = midTunnel2
	v.longTunnel1Period = longTunnel1
	v.longTunnel2Period = longTunnel2
	v.stopLossPercent = stopLoss
	v.takeProfitPercent = takeProfit
}

// UpdateKlineData 更新K线数据
func (v *VegasTunnelStrategy) UpdateKlineData(kline KlineData, timeframe string) {
	switch timeframe {
	case "15m":
		v.kline15MData = append(v.kline15MData, kline)
		// 保持最近1000根K线
		if len(v.kline15MData) > 1000 {
			v.kline15MData = v.kline15MData[1:]
		}
	case "4h":
		v.kline4HData = append(v.kline4HData, kline)
		// 保持最近500根K线
		if len(v.kline4HData) > 500 {
			v.kline4HData = v.kline4HData[1:]
		}
	}
}

// CalculateEMA 计算指数移动平均线
func (v *VegasTunnelStrategy) CalculateEMA(prices []decimal.Decimal, period int) []decimal.Decimal {
	if len(prices) < period {
		return nil
	}

	result := make([]decimal.Decimal, len(prices))
	alpha := decimal.NewFromFloat(2.0 / float64(period+1))
	one := decimal.NewFromInt(1)

	// 第一个EMA值使用SMA
	sum := decimal.Zero
	for i := 0; i < period; i++ {
		sum = sum.Add(prices[i])
	}
	result[period-1] = sum.Div(decimal.NewFromInt(int64(period)))

	// 后续EMA值
	for i := period; i < len(prices); i++ {
		result[i] = prices[i].Mul(alpha).Add(result[i-1].Mul(one.Sub(alpha)))
	}

	return result
}

// CalculateTunnelData 计算隧道数据
func (v *VegasTunnelStrategy) CalculateTunnelData(klines []KlineData) []TunnelData {
	if len(klines) < v.longTunnel2Period {
		return nil
	}

	// 提取收盘价
	prices := make([]decimal.Decimal, len(klines))
	for i, kline := range klines {
		prices[i] = kline.Close
	}

	// 计算所有EMA
	ema12 := v.CalculateEMA(prices, v.shortEMAPeriod)
	ema144 := v.CalculateEMA(prices, v.midTunnel1Period)
	ema169 := v.CalculateEMA(prices, v.midTunnel2Period)
	ema288 := v.CalculateEMA(prices, v.longTunnel1Period)
	ema338 := v.CalculateEMA(prices, v.longTunnel2Period)

	if ema12 == nil || ema144 == nil || ema169 == nil || ema288 == nil || ema338 == nil {
		return nil
	}

	tunnelData := make([]TunnelData, len(klines))

	for i := v.longTunnel2Period - 1; i < len(klines); i++ {
		tunnel := &tunnelData[i]
		tunnel.EMA12 = ema12[i]
		tunnel.EMA144 = ema144[i]
		tunnel.EMA169 = ema169[i]
		tunnel.EMA288 = ema288[i]
		tunnel.EMA338 = ema338[i]

		// 计算隧道边界
		if tunnel.EMA144.GreaterThan(tunnel.EMA169) {
			tunnel.MidTunnelUpper = tunnel.EMA144
			tunnel.MidTunnelLower = tunnel.EMA169
		} else {
			tunnel.MidTunnelUpper = tunnel.EMA169
			tunnel.MidTunnelLower = tunnel.EMA144
		}

		if tunnel.EMA288.GreaterThan(tunnel.EMA338) {
			tunnel.LongTunnelUpper = tunnel.EMA288
			tunnel.LongTunnelLower = tunnel.EMA338
		} else {
			tunnel.LongTunnelUpper = tunnel.EMA338
			tunnel.LongTunnelLower = tunnel.EMA288
		}

		// 判断趋势方向
		tunnel.TrendDirection = v.determineTrendDirection(*tunnel)
	}

	return tunnelData
}

// determineTrendDirection 判断趋势方向
func (v *VegasTunnelStrategy) determineTrendDirection(tunnel TunnelData) TrendDirection {
	// 多头排列：价格 > 中期隧道 > 长期隧道
	if tunnel.MidTunnelLower.GreaterThan(tunnel.LongTunnelUpper) {
		return TrendBullish
	}
	// 空头排列：价格 < 中期隧道 < 长期隧道
	if tunnel.MidTunnelUpper.LessThan(tunnel.LongTunnelLower) {
		return TrendBearish
	}
	// 震荡：隧道交叉
	return TrendSideways
}

// GenerateSignal 生成交易信号
func (v *VegasTunnelStrategy) GenerateSignal(klines []KlineData) *TradingSignal {
	if len(klines) == 0 {
		return nil
	}
	
	// 获取symbol
	symbol := klines[0].Symbol
	
	// 更新K线数据（假设输入的是15M数据）
	for _, kline := range klines {
		v.UpdateKlineData(kline, "15M")
	}
	// 检查数据充足性
	if len(v.kline15MData) < v.longTunnel2Period {
		v.logger.Debugf("Insufficient 15M data for signal generation: %d", len(v.kline15MData))
		return nil
	}
	
	// 如果没有4H数据，使用15M数据模拟
	if len(v.kline4HData) < v.longTunnel2Period {
		v.logger.Debugf("Insufficient 4H data, using 15M data: %d", len(v.kline4HData))
		// 可以考虑从15M数据中提取4H数据或使用其他逻辑
		return nil
	}

	// 计算4H隧道数据（宏观趋势确认）
	tunnel4H := v.CalculateTunnelData(v.kline4HData)
	if tunnel4H == nil {
		return nil
	}
	current4H := tunnel4H[len(tunnel4H)-1]

	// 计算15M隧道数据（战术入场点）
	tunnel15M := v.CalculateTunnelData(v.kline15MData)
	if tunnel15M == nil {
		return nil
	}
	current15M := tunnel15M[len(tunnel15M)-1]
	currentKline := v.kline15MData[len(v.kline15MData)-1]

	// 检查多头入场信号
	if signal := v.checkLongSignal(current4H, current15M, currentKline, symbol); signal != nil {
		return signal
	}

	// 检查空头入场信号
	if signal := v.checkShortSignal(current4H, current15M, currentKline, symbol); signal != nil {
		return signal
	}

	return nil
}

// checkLongSignal 检查多头入场信号
func (v *VegasTunnelStrategy) checkLongSignal(tunnel4H, tunnel15M TunnelData, kline KlineData, symbol string) *TradingSignal {
	// 1. 4H宏观确认：多头排列
	if tunnel4H.TrendDirection != TrendBullish {
		return nil
	}

	// 2. 4H价格位置确认：现价 > EMA144/169隧道
	if kline.Close.LessThanOrEqual(tunnel4H.MidTunnelLower) {
		return nil
	}

	// 3. 15M战术回调：价格回调至隧道区域获得支撑
	if !v.isPriceNearTunnel(kline.Close, tunnel15M, true) {
		return nil
	}

	// 4. 15M动能触发：收盘价站上EMA12
	if kline.Close.LessThanOrEqual(tunnel15M.EMA12) {
		return nil
	}

	// 生成多头信号
	signal := &TradingSignal{
		Symbol:    symbol,
		Type:      SignalBuy,
		Price:     kline.Close,
		Confidence: v.calculateSignalConfidence(tunnel4H, tunnel15M, true),
		Reason:    "4H多头排列，15M回调至隧道获支撑后站上EMA12",
		Timestamp: kline.Timestamp,
		Timeframe: "15M",
	}

	// 计算止损止盈
	v.calculateStopLossAndTakeProfit(signal, tunnel15M, true)

	return signal
}

// checkShortSignal 检查空头入场信号
func (v *VegasTunnelStrategy) checkShortSignal(tunnel4H, tunnel15M TunnelData, kline KlineData, symbol string) *TradingSignal {
	// 1. 4H宏观确认：空头排列
	if tunnel4H.TrendDirection != TrendBearish {
		return nil
	}

	// 2. 4H价格位置确认：现价 < EMA144/169隧道
	if kline.Close.GreaterThanOrEqual(tunnel4H.MidTunnelUpper) {
		return nil
	}

	// 3. 15M战术反弹：价格反弹至隧道区域受到压制
	if !v.isPriceNearTunnel(kline.Close, tunnel15M, false) {
		return nil
	}

	// 4. 15M动能触发：收盘价跌破EMA12
	if kline.Close.GreaterThanOrEqual(tunnel15M.EMA12) {
		return nil
	}

	// 生成空头信号
	signal := &TradingSignal{
		Symbol:    symbol,
		Type:      SignalSell,
		Price:     kline.Close,
		Confidence: v.calculateSignalConfidence(tunnel4H, tunnel15M, false),
		Reason:    "4H空头排列，15M反弹至隧道受压制后跌破EMA12",
		Timestamp: kline.Timestamp,
		Timeframe: "15M",
	}

	// 计算止损止盈
	v.calculateStopLossAndTakeProfit(signal, tunnel15M, false)

	return signal
}

// isPriceNearTunnel 判断价格是否接近隧道区域
func (v *VegasTunnelStrategy) isPriceNearTunnel(price decimal.Decimal, tunnel TunnelData, isLong bool) bool {
	tolerance := decimal.NewFromFloat(0.002) // 0.2%的容差

	if isLong {
		// 多头：检查是否在中期或长期隧道附近获得支撑
		// 价格在中期隧道范围内或略低于下边界
		if price.GreaterThanOrEqual(tunnel.MidTunnelLower.Mul(decimal.NewFromInt(1).Sub(tolerance))) &&
		   price.LessThanOrEqual(tunnel.MidTunnelUpper.Mul(decimal.NewFromInt(1).Add(tolerance))) {
			return true
		}
		
		// 价格在长期隧道范围内或略低于下边界
		if price.GreaterThanOrEqual(tunnel.LongTunnelLower.Mul(decimal.NewFromInt(1).Sub(tolerance))) &&
		   price.LessThanOrEqual(tunnel.LongTunnelUpper.Mul(decimal.NewFromInt(1).Add(tolerance))) {
			return true
		}
	} else {
		// 空头：检查是否在中期或长期隧道附近受到压制
		// 价格在中期隧道范围内或略高于上边界
		if price.GreaterThanOrEqual(tunnel.MidTunnelLower.Mul(decimal.NewFromInt(1).Sub(tolerance))) &&
		   price.LessThanOrEqual(tunnel.MidTunnelUpper.Mul(decimal.NewFromInt(1).Add(tolerance))) {
			return true
		}
		
		// 价格在长期隧道范围内或略高于上边界
		if price.GreaterThanOrEqual(tunnel.LongTunnelLower.Mul(decimal.NewFromInt(1).Sub(tolerance))) &&
		   price.LessThanOrEqual(tunnel.LongTunnelUpper.Mul(decimal.NewFromInt(1).Add(tolerance))) {
			return true
		}
	}

	return false
}

// calculateSignalConfidence 计算信号置信度
func (v *VegasTunnelStrategy) calculateSignalConfidence(tunnel4H, tunnel15M TunnelData, isLong bool) float64 {
	confidence := 0.6 // 基础置信度

	// 4H趋势强度加分
	if isLong {
		if tunnel4H.MidTunnelLower.GreaterThan(tunnel4H.LongTunnelUpper) {
			confidence += 0.2 // 明显多头排列
		}
	} else {
		if tunnel4H.MidTunnelUpper.LessThan(tunnel4H.LongTunnelLower) {
			confidence += 0.2 // 明显空头排列
		}
	}

	// EMA12动能强度加分
	if isLong {
		if tunnel15M.EMA12.GreaterThan(tunnel15M.MidTunnelLower) {
			confidence += 0.1
		}
	} else {
		if tunnel15M.EMA12.LessThan(tunnel15M.MidTunnelUpper) {
			confidence += 0.1
		}
	}

	return math.Min(confidence, 1.0)
}

// calculateStopLossAndTakeProfit 计算止损止盈
func (v *VegasTunnelStrategy) calculateStopLossAndTakeProfit(signal *TradingSignal, tunnel TunnelData, isLong bool) {
	if isLong {
		// 多头止损：设置在支撑隧道下方
		stopLossLevel := tunnel.MidTunnelLower
		if tunnel.LongTunnelUpper.LessThan(tunnel.MidTunnelLower) {
			stopLossLevel = tunnel.LongTunnelUpper
		}
		signal.StopLoss = stopLossLevel.Mul(decimal.NewFromFloat(0.998)) // 0.2%缓冲
		
		// 第一止盈目标：2R
		riskAmount := signal.Price.Sub(signal.StopLoss)
		signal.TakeProfit = signal.Price.Add(riskAmount.Mul(decimal.NewFromFloat(v.riskRewardRatio)))
	} else {
		// 空头止损：设置在阻力隧道上方
		stopLossLevel := tunnel.MidTunnelUpper
		if tunnel.LongTunnelLower.GreaterThan(tunnel.MidTunnelUpper) {
			stopLossLevel = tunnel.LongTunnelLower
		}
		signal.StopLoss = stopLossLevel.Mul(decimal.NewFromFloat(1.002)) // 0.2%缓冲
		
		// 第一止盈目标：2R
		riskAmount := signal.StopLoss.Sub(signal.Price)
		signal.TakeProfit = signal.Price.Sub(riskAmount.Mul(decimal.NewFromFloat(v.riskRewardRatio)))
	}
}

// GetStrategyInfo 获取策略信息
func (v *VegasTunnelStrategy) GetStrategyInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":                "Vegas Dual Tunnel Strategy",
		"short_ema_period":    v.shortEMAPeriod,
		"mid_tunnel1_period":  v.midTunnel1Period,
		"mid_tunnel2_period":  v.midTunnel2Period,
		"long_tunnel1_period": v.longTunnel1Period,
		"long_tunnel2_period": v.longTunnel2Period,
		"min_tunnel_period":   v.minTunnelPeriod,
		"volume_factor":       v.volumeFactor,
		"risk_reward_ratio":   v.riskRewardRatio,
		"stop_loss_percent":   v.stopLossPercent,
		"take_profit_percent":  v.takeProfitPercent,
		"15m_data_count":      len(v.kline15MData),
		"4h_data_count":       len(v.kline4HData),
	}
}

// ValidateParameters 验证策略参数
func (v *VegasTunnelStrategy) ValidateParameters() error {
	if v.shortEMAPeriod <= 0 {
		return fmt.Errorf("short EMA period must be positive")
	}

	if v.midTunnel1Period <= v.shortEMAPeriod || v.midTunnel2Period <= v.shortEMAPeriod {
		return fmt.Errorf("mid tunnel periods must be greater than short EMA period")
	}

	if v.longTunnel1Period <= v.midTunnel2Period || v.longTunnel2Period <= v.midTunnel2Period {
		return fmt.Errorf("long tunnel periods must be greater than mid tunnel periods")
	}

	if v.stopLossPercent <= 0 || v.stopLossPercent > 0.1 {
		return fmt.Errorf("stop loss percent must be between 0 and 0.1 (10%%)")
	}

	if v.takeProfitPercent <= 0 || v.takeProfitPercent > 0.2 {
		return fmt.Errorf("take profit percent must be between 0 and 0.2 (20%%)")
	}

	if v.riskRewardRatio <= 1.0 {
		return fmt.Errorf("risk reward ratio must be greater than 1.0")
	}

	return nil
}

// CheckEMA12Exit 检查EMA12移动止盈出场信号
func (v *VegasTunnelStrategy) CheckEMA12Exit(symbol string, isLong bool) *TradingSignal {
	if len(v.kline15MData) < 2 {
		return nil
	}

	tunnel15M := v.CalculateTunnelData(v.kline15MData)
	if tunnel15M == nil {
		return nil
	}

	currentKline := v.kline15MData[len(v.kline15MData)-1]
	currentTunnel := tunnel15M[len(tunnel15M)-1]

	var shouldExit bool
	var reason string

	if isLong {
		// 多单：收盘价跌破EMA12
		shouldExit = currentKline.Close.LessThan(currentTunnel.EMA12)
		reason = "15M收盘价跌破EMA12移动止盈线"
	} else {
		// 空单：收盘价突破EMA12
		shouldExit = currentKline.Close.GreaterThan(currentTunnel.EMA12)
		reason = "15M收盘价突破EMA12移动止盈线"
	}

	if !shouldExit {
		return nil
	}

	return &TradingSignal{
		Symbol:    symbol,
		Type:      SignalTakeProfit,
		Price:     currentKline.Close,
		Confidence: 0.9,
		Reason:    reason,
		Timestamp: currentKline.Timestamp,
		Timeframe: "15M",
	}
}