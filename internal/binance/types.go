package binance

import "github.com/shopspring/decimal"

// AccountInfo 账户信息
type AccountInfo struct {
	FeeTier                     int             `json:"feeTier"`
	CanTrade                    bool            `json:"canTrade"`
	CanDeposit                  bool            `json:"canDeposit"`
	CanWithdraw                 bool            `json:"canWithdraw"`
	UpdateTime                  int64           `json:"updateTime"`
	TotalInitialMargin          string          `json:"totalInitialMargin"`
	TotalMaintMargin            string          `json:"totalMaintMargin"`
	TotalWalletBalance          string          `json:"totalWalletBalance"`
	TotalUnrealizedProfit       string          `json:"totalUnrealizedProfit"`
	TotalMarginBalance          string          `json:"totalMarginBalance"`
	TotalPositionInitialMargin  string          `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin string          `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     string          `json:"totalCrossWalletBalance"`
	TotalCrossUnPnl             string          `json:"totalCrossUnPnl"`
	AvailableBalance            string          `json:"availableBalance"`
	MaxWithdrawAmount           string          `json:"maxWithdrawAmount"`
	Assets                      []AccountAsset  `json:"assets"`
	Positions                   []AccountPosition `json:"positions"`
}

// AccountAsset 账户资产
type AccountAsset struct {
	Asset                  string `json:"asset"`
	WalletBalance          string `json:"walletBalance"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	MarginBalance          string `json:"marginBalance"`
	MaintMargin            string `json:"maintMargin"`
	InitialMargin          string `json:"initialMargin"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	CrossWalletBalance     string `json:"crossWalletBalance"`
	CrossUnPnl             string `json:"crossUnPnl"`
	AvailableBalance       string `json:"availableBalance"`
	MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
	MarginAvailable        bool   `json:"marginAvailable"`
	UpdateTime             int64  `json:"updateTime"`
}

// AccountPosition 账户持仓
type AccountPosition struct {
	Symbol                 string `json:"symbol"`
	InitialMargin          string `json:"initialMargin"`
	MaintMargin            string `json:"maintMargin"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	Leverage               string `json:"leverage"`
	Isolated               bool   `json:"isolated"`
	EntryPrice             string `json:"entryPrice"`
	MaxNotional            string `json:"maxNotional"`
	PositionSide           string `json:"positionSide"`
	PositionAmt            string `json:"positionAmt"`
	UpdateTime             int64  `json:"updateTime"`
}

// Position 持仓信息
type Position struct {
	Symbol           string          `json:"symbol"`
	PositionAmt      string          `json:"positionAmt"`
	EntryPrice       string          `json:"entryPrice"`
	MarkPrice        string          `json:"markPrice"`
	UnRealizedProfit string          `json:"unRealizedProfit"`
	LiquidationPrice string          `json:"liquidationPrice"`
	Leverage         string          `json:"leverage"`
	MaxNotionalValue string          `json:"maxNotionalValue"`
	MarginType       string          `json:"marginType"`
	IsolatedMargin   string          `json:"isolatedMargin"`
	IsAutoAddMargin  string          `json:"isAutoAddMargin"`
	PositionSide     string          `json:"positionSide"`
	Notional         string          `json:"notional"`
	IsolatedWallet   string          `json:"isolatedWallet"`
	UpdateTime       int64           `json:"updateTime"`
}

// Kline K线数据
type Kline struct {
	OpenTime                 int64  `json:"openTime"`
	Open                     string `json:"open"`
	High                     string `json:"high"`
	Low                      string `json:"low"`
	Close                    string `json:"close"`
	Volume                   string `json:"volume"`
	CloseTime                int64  `json:"closeTime"`
	QuoteAssetVolume         string `json:"quoteAssetVolume"`
	NumberOfTrades           int    `json:"numberOfTrades"`
	TakerBuyBaseAssetVolume  string `json:"takerBuyBaseAssetVolume"`
	TakerBuyQuoteAssetVolume string `json:"takerBuyQuoteAssetVolume"`
}

// OrderRequest 下单请求
type OrderRequest struct {
	Symbol           string `json:"symbol"`
	Side             string `json:"side"`             // BUY, SELL
	Type             string `json:"type"`             // LIMIT, MARKET, STOP, TAKE_PROFIT, etc.
	TimeInForce      string `json:"timeInForce"`      // GTC, IOC, FOK
	Quantity         string `json:"quantity"`
	Price            string `json:"price,omitempty"`
	StopPrice        string `json:"stopPrice,omitempty"`
	ClosePosition    bool   `json:"closePosition,omitempty"`
	ActivationPrice  string `json:"activationPrice,omitempty"`
	CallbackRate     string `json:"callbackRate,omitempty"`
	WorkingType      string `json:"workingType,omitempty"`
	PriceProtect     bool   `json:"priceProtect,omitempty"`
	NewOrderRespType string `json:"newOrderRespType,omitempty"`
}

// OrderResponse 下单响应
type OrderResponse struct {
	OrderID       int64  `json:"orderId"`
	Symbol        string `json:"symbol"`
	Status        string `json:"status"`
	ClientOrderID string `json:"clientOrderId"`
	Price         string `json:"price"`
	AvgPrice      string `json:"avgPrice"`
	OrigQty       string `json:"origQty"`
	ExecutedQty   string `json:"executedQty"`
	CumQty        string `json:"cumQty"`
	CumQuote      string `json:"cumQuote"`
	TimeInForce   string `json:"timeInForce"`
	Type          string `json:"type"`
	ReduceOnly    bool   `json:"reduceOnly"`
	ClosePosition bool   `json:"closePosition"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	StopPrice     string `json:"stopPrice"`
	WorkingType   string `json:"workingType"`
	PriceProtect  bool   `json:"priceProtect"`
	OrigType      string `json:"origType"`
	UpdateTime    int64  `json:"updateTime"`
}

// APIError API错误响应
type APIError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// TickerPrice 价格信息
type TickerPrice struct {
	Symbol string          `json:"symbol"`
	Price  decimal.Decimal `json:"price"`
	Time   int64           `json:"time"`
}

// OrderSide 订单方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType 订单类型
type OrderType string

const (
	OrderTypeLimit              OrderType = "LIMIT"
	OrderTypeMarket             OrderType = "MARKET"
	OrderTypeStop               OrderType = "STOP"
	OrderTypeStopMarket         OrderType = "STOP_MARKET"
	OrderTypeTakeProfit         OrderType = "TAKE_PROFIT"
	OrderTypeTakeProfitMarket   OrderType = "TAKE_PROFIT_MARKET"
	OrderTypeTrailingStopMarket OrderType = "TRAILING_STOP_MARKET"
)

// TimeInForce 订单有效期
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Cancel
	TimeInForceIOC TimeInForce = "IOC" // Immediate or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill or Kill
	TimeInForceGTX TimeInForce = "GTX" // Good Till Crossing
)

// PositionSide 持仓方向
type PositionSide string

const (
	PositionSideBoth  PositionSide = "BOTH"
	PositionSideLong  PositionSide = "LONG"
	PositionSideShort PositionSide = "SHORT"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "NEW"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCanceled        OrderStatus = "CANCELED"
	OrderStatusRejected        OrderStatus = "REJECTED"
	OrderStatusExpired         OrderStatus = "EXPIRED"
)

// Interval K线时间间隔
type Interval string

const (
	Interval1m  Interval = "1m"
	Interval3m  Interval = "3m"
	Interval5m  Interval = "5m"
	Interval15m Interval = "15m"
	Interval30m Interval = "30m"
	Interval1h  Interval = "1h"
	Interval2h  Interval = "2h"
	Interval4h  Interval = "4h"
	Interval6h  Interval = "6h"
	Interval8h  Interval = "8h"
	Interval12h Interval = "12h"
	Interval1d  Interval = "1d"
	Interval3d  Interval = "3d"
	Interval1w  Interval = "1w"
	Interval1M  Interval = "1M"
)