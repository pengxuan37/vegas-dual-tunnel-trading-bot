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
	shortEMAPeriod   int     // 短期EMA周期，默认12
	longEMAPeriod    int     // 长期EMA周期，默认144
	tunnelWidth      float64 // 隧道宽度百分比，默认0.5%
	minTunnelPeriod  int     // 最小隧道持续周期，默认3
	volumeFactor     float64 // 成交量确认因子，默认1.5
	riskRewardRatio  float64 // 风险收益比，默认1:2
	stopLossPercent  float64 // 止损百分比，默认2%
	takeProfitPercent float64 // 止盈百分比，默认4%
}

// KlineData K线数据
type KlineData struct {
	Symbol      string
	OpenTime    time.Time
	CloseTime   time.Time
	Open        decimal.Decimal
	High        decimal.Decimal
	Low         decimal.Decimal
	Close       decimal.Decimal
	Volume      decimal.Decimal
	QuoteVolume decimal.Decimal
	IsClosed    bool
}

// TunnelData 隧道数据
type TunnelData struct {
	ShortEMA    decimal.Decimal
	LongEMA     decimal.Decimal
	UpperBound  decimal.Decimal
	LowerBound  decimal.Decimal
	TunnelWidth decimal.Decimal
	IsFormed    bool
	Direction   TunnelDirection
}

// TunnelDirection 隧道方向
type TunnelDirection int

const (
	TunnelNone TunnelDirection = iota
	TunnelBullish  // 看涨隧道
	TunnelBearish  // 看跌隧道
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
}

// NewVegasTunnelStrategy 创建新的维加斯隧道策略实例
func NewVegasTunnelStrategy(log logger.Logger) *VegasTunnelStrategy {
	return &VegasTunnelStrategy{
		logger:            log,
		shortEMAPeriod:    12,
		longEMAPeriod:     144,
		tunnelWidth:       0.005, // 0.5%
		minTunnelPeriod:   3,
		volumeFactor:      1.5,
		riskRewardRatio:   2.0,
		stopLossPercent:   0.02, // 2%
		takeProfitPercent: 0.04, // 4%
	}
}

// SetParameters 设置策略参数
func (v *VegasTunnelStrategy) SetParameters(shortEMA, longEMA int, tunnelWidth, stopLoss, takeProfit float64) {
	v.shortEMAPeriod = shortEMA
	v.longEMAPeriod = longEMA
	v.tunnelWidth = tunnelWidth
	v.stopLossPercent = stopLoss
	v.takeProfitPercent = takeProfit
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

// CalculateTunnel 计算隧道数据
func (v *VegasTunnelStrategy) CalculateTunnel(klines []KlineData) []TunnelData {
	if len(klines) < v.longEMAPeriod {
		return nil
	}

	// 提取收盘价
	prices := make([]decimal.Decimal, len(klines))
	for i, kline := range klines {
		prices[i] = kline.Close
	}

	// 计算EMA
	shortEMA := v.CalculateEMA(prices, v.shortEMAPeriod)
	longEMA := v.CalculateEMA(prices, v.longEMAPeriod)

	if shortEMA == nil || longEMA == nil {
		return nil
	}

	tunnelData := make([]TunnelData, len(klines))
	tunnelWidthDecimal := decimal.NewFromFloat(v.tunnelWidth)

	for i := v.longEMAPeriod - 1; i < len(klines); i++ {
		tunnel := &tunnelData[i]
		tunnel.ShortEMA = shortEMA[i]
		tunnel.LongEMA = longEMA[i]

		// 计算隧道边界
		midPrice := tunnel.ShortEMA.Add(tunnel.LongEMA).Div(decimal.NewFromInt(2))
		tunnel.TunnelWidth = midPrice.Mul(tunnelWidthDecimal)
		tunnel.UpperBound = midPrice.Add(tunnel.TunnelWidth)
		tunnel.LowerBound = midPrice.Sub(tunnel.TunnelWidth)

		// 判断隧道形成和方向
		tunnel.IsFormed = v.isTunnelFormed(tunnelData, i)
		tunnel.Direction = v.getTunnelDirection(shortEMA[i], longEMA[i])
	}

	return tunnelData
}

// isTunnelFormed 判断隧道是否形成
func (v *VegasTunnelStrategy) isTunnelFormed(tunnelData []TunnelData, index int) bool {
	if index < v.minTunnelPeriod {
		return false
	}

	// 检查最近几个周期的EMA是否保持相对稳定的距离
	currentTunnel := tunnelData[index]
	for i := 1; i <= v.minTunnelPeriod; i++ {
		prevIndex := index - i
		if prevIndex < 0 {
			return false
		}

		prevTunnel := tunnelData[prevIndex]
		
		// 检查EMA距离变化是否在合理范围内
		currentDistance := currentTunnel.ShortEMA.Sub(currentTunnel.LongEMA).Abs()
		prevDistance := prevTunnel.ShortEMA.Sub(prevTunnel.LongEMA).Abs()
		
		if currentDistance.IsZero() || prevDistance.IsZero() {
			return false
		}

		// 距离变化不应超过20%
		distanceChange := currentDistance.Sub(prevDistance).Abs().Div(prevDistance)
		if distanceChange.GreaterThan(decimal.NewFromFloat(0.2)) {
			return false
		}
	}

	return true
}

// getTunnelDirection 获取隧道方向
func (v *VegasTunnelStrategy) getTunnelDirection(shortEMA, longEMA decimal.Decimal) TunnelDirection {
	if shortEMA.GreaterThan(longEMA) {
		return TunnelBullish
	} else if shortEMA.LessThan(longEMA) {
		return TunnelBearish
	}
	return TunnelNone
}

// GenerateSignal 生成交易信号
func (v *VegasTunnelStrategy) GenerateSignal(klines []KlineData) *TradingSignal {
	if len(klines) < v.longEMAPeriod+v.minTunnelPeriod {
		return nil
	}

	tunnelData := v.CalculateTunnel(klines)
	if tunnelData == nil {
		return nil
	}

	currentIndex := len(klines) - 1
	currentKline := klines[currentIndex]
	currentTunnel := tunnelData[currentIndex]

	// 检查隧道是否形成
	if !currentTunnel.IsFormed {
		return nil
	}

	// 检查价格突破
	signal := v.checkBreakout(currentKline, currentTunnel, klines)
	if signal != nil {
		// 添加成交量确认
		if v.confirmWithVolume(klines, currentIndex) {
			signal.Confidence = math.Min(signal.Confidence*1.2, 1.0)
			v.logger.Infof("Volume confirmation added, confidence: %.2f", signal.Confidence)
		}

		// 计算止损止盈
		v.calculateStopLossAndTakeProfit(signal, currentKline.Close)
	}

	return signal
}

// checkBreakout 检查价格突破
func (v *VegasTunnelStrategy) checkBreakout(kline KlineData, tunnel TunnelData, klines []KlineData) *TradingSignal {
	price := kline.Close

	// 向上突破
	if price.GreaterThan(tunnel.UpperBound) && tunnel.Direction == TunnelBullish {
		// 检查是否是有效突破（收盘价突破）
		if v.isValidBreakout(klines, len(klines)-1, true) {
			return &TradingSignal{
				Type:       SignalBuy,
				Price:      price,
				Confidence: v.calculateConfidence(klines, tunnel, true),
				Reason:     "Bullish tunnel breakout",
				Timestamp:  kline.CloseTime,
			}
		}
	}

	// 向下突破
	if price.LessThan(tunnel.LowerBound) && tunnel.Direction == TunnelBearish {
		// 检查是否是有效突破
		if v.isValidBreakout(klines, len(klines)-1, false) {
			return &TradingSignal{
				Type:       SignalSell,
				Price:      price,
				Confidence: v.calculateConfidence(klines, tunnel, false),
				Reason:     "Bearish tunnel breakout",
				Timestamp:  kline.CloseTime,
			}
		}
	}

	return nil
}

// isValidBreakout 检查是否是有效突破
func (v *VegasTunnelStrategy) isValidBreakout(klines []KlineData, index int, isBullish bool) bool {
	if index < 2 {
		return false
	}

	currentKline := klines[index]
	prevKline := klines[index-1]

	if isBullish {
		// 看涨突破：当前收盘价 > 前一根收盘价，且有一定的价格幅度
		priceIncrease := currentKline.Close.Sub(prevKline.Close).Div(prevKline.Close)
		return priceIncrease.GreaterThan(decimal.NewFromFloat(0.001)) // 至少0.1%的涨幅
	} else {
		// 看跌突破：当前收盘价 < 前一根收盘价，且有一定的价格幅度
		priceDecrease := prevKline.Close.Sub(currentKline.Close).Div(prevKline.Close)
		return priceDecrease.GreaterThan(decimal.NewFromFloat(0.001)) // 至少0.1%的跌幅
	}
}

// calculateConfidence 计算信号置信度
func (v *VegasTunnelStrategy) calculateConfidence(klines []KlineData, tunnel TunnelData, isBullish bool) float64 {
	confidence := 0.5 // 基础置信度

	// 隧道方向一致性加分
	if (isBullish && tunnel.Direction == TunnelBullish) || (!isBullish && tunnel.Direction == TunnelBearish) {
		confidence += 0.2
	}

	// 突破幅度加分
	currentPrice := klines[len(klines)-1].Close
	var breakoutStrength decimal.Decimal
	if isBullish {
		breakoutStrength = currentPrice.Sub(tunnel.UpperBound).Div(tunnel.TunnelWidth)
	} else {
		breakoutStrength = tunnel.LowerBound.Sub(currentPrice).Div(tunnel.TunnelWidth)
	}

	if breakoutStrength.GreaterThan(decimal.NewFromFloat(0.5)) {
		confidence += 0.1
	}
	if breakoutStrength.GreaterThan(decimal.NewFromFloat(1.0)) {
		confidence += 0.1
	}

	return math.Min(confidence, 1.0)
}

// confirmWithVolume 成交量确认
func (v *VegasTunnelStrategy) confirmWithVolume(klines []KlineData, index int) bool {
	if index < 20 {
		return false
	}

	// 计算过去20根K线的平均成交量
	totalVolume := decimal.Zero
	for i := index - 19; i <= index; i++ {
		totalVolume = totalVolume.Add(klines[i].Volume)
	}
	avgVolume := totalVolume.Div(decimal.NewFromInt(20))

	// 当前成交量是否超过平均成交量的1.5倍
	currentVolume := klines[index].Volume
	volumeThreshold := avgVolume.Mul(decimal.NewFromFloat(v.volumeFactor))

	return currentVolume.GreaterThan(volumeThreshold)
}

// calculateStopLossAndTakeProfit 计算止损止盈
func (v *VegasTunnelStrategy) calculateStopLossAndTakeProfit(signal *TradingSignal, entryPrice decimal.Decimal) {
	stopLossPercent := decimal.NewFromFloat(v.stopLossPercent)
	takeProfitPercent := decimal.NewFromFloat(v.takeProfitPercent)

	if signal.Type == SignalBuy {
		// 做多：止损在入场价下方，止盈在入场价上方
		signal.StopLoss = entryPrice.Mul(decimal.NewFromInt(1).Sub(stopLossPercent))
		signal.TakeProfit = entryPrice.Mul(decimal.NewFromInt(1).Add(takeProfitPercent))
	} else if signal.Type == SignalSell {
		// 做空：止损在入场价上方，止盈在入场价下方
		signal.StopLoss = entryPrice.Mul(decimal.NewFromInt(1).Add(stopLossPercent))
		signal.TakeProfit = entryPrice.Mul(decimal.NewFromInt(1).Sub(takeProfitPercent))
	}
}

// GetStrategyInfo 获取策略信息
func (v *VegasTunnelStrategy) GetStrategyInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":               "Vegas Tunnel Strategy",
		"short_ema_period":   v.shortEMAPeriod,
		"long_ema_period":    v.longEMAPeriod,
		"tunnel_width":       v.tunnelWidth,
		"min_tunnel_period":  v.minTunnelPeriod,
		"volume_factor":      v.volumeFactor,
		"risk_reward_ratio":  v.riskRewardRatio,
		"stop_loss_percent":  v.stopLossPercent,
		"take_profit_percent": v.takeProfitPercent,
	}
}

// ValidateParameters 验证策略参数
func (v *VegasTunnelStrategy) ValidateParameters() error {
	if v.shortEMAPeriod <= 0 || v.longEMAPeriod <= 0 {
		return fmt.Errorf("EMA periods must be positive")
	}

	if v.shortEMAPeriod >= v.longEMAPeriod {
		return fmt.Errorf("short EMA period must be less than long EMA period")
	}

	if v.tunnelWidth <= 0 || v.tunnelWidth > 0.1 {
		return fmt.Errorf("tunnel width must be between 0 and 0.1 (10%%)")
	}

	if v.stopLossPercent <= 0 || v.stopLossPercent > 0.1 {
		return fmt.Errorf("stop loss percent must be between 0 and 0.1 (10%%)")
	}

	if v.takeProfitPercent <= 0 || v.takeProfitPercent > 0.2 {
		return fmt.Errorf("take profit percent must be between 0 and 0.2 (20%%)")
	}

	return nil
}