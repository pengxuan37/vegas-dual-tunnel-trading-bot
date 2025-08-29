package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
)

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	logger     logger.Logger
	conn       *websocket.Conn
	baseURL    string
	streams    []string
	handlers   map[string]StreamHandler
	mu         sync.RWMutex
	isRunning  bool
	reconnect  bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// StreamHandler 数据流处理器接口
type StreamHandler interface {
	HandleKlineData(data *KlineStreamData) error
	HandleTickerData(data *TickerStreamData) error
	GetName() string
}

// KlineStreamData K线数据流
type KlineStreamData struct {
	Stream string `json:"stream"`
	Data   struct {
		EventType string `json:"e"`
		EventTime int64  `json:"E"`
		Symbol    string `json:"s"`
		Kline     struct {
			StartTime            int64  `json:"t"`
			EndTime              int64  `json:"T"`
			Symbol               string `json:"s"`
			Interval             string `json:"i"`
			FirstTradeID         int64  `json:"f"`
			LastTradeID          int64  `json:"L"`
			Open                 string `json:"o"`
			Close                string `json:"c"`
			High                 string `json:"h"`
			Low                  string `json:"l"`
			Volume               string `json:"v"`
			NumberOfTrades       int64  `json:"n"`
			IsClosed             bool   `json:"x"`
			QuoteAssetVolume     string `json:"q"`
			TakerBuyBaseVolume   string `json:"V"`
			TakerBuyQuoteVolume  string `json:"Q"`
		} `json:"k"`
	} `json:"data"`
}

// TickerStreamData 价格数据流
type TickerStreamData struct {
	Stream string `json:"stream"`
	Data   struct {
		EventType             string `json:"e"`
		EventTime             int64  `json:"E"`
		Symbol                string `json:"s"`
		PriceChange           string `json:"p"`
		PriceChangePercent    string `json:"P"`
		WeightedAvgPrice      string `json:"w"`
		PrevClosePrice        string `json:"x"`
		LastPrice             string `json:"c"`
		LastQty               string `json:"Q"`
		BidPrice              string `json:"b"`
		BidQty                string `json:"B"`
		AskPrice              string `json:"a"`
		AskQty                string `json:"A"`
		OpenPrice             string `json:"o"`
		HighPrice             string `json:"h"`
		LowPrice              string `json:"l"`
		Volume                string `json:"v"`
		QuoteVolume           string `json:"q"`
		OpenTime              int64  `json:"O"`
		CloseTime             int64  `json:"C"`
		FirstID               int64  `json:"F"`
		LastID                int64  `json:"L"`
		Count                 int64  `json:"c"`
	} `json:"data"`
}

// NewWebSocketClient 创建新的WebSocket客户端
func NewWebSocketClient(baseURL string, log logger.Logger) (*WebSocketClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base URL cannot be empty")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketClient{
		logger:    log,
		baseURL:   baseURL,
		streams:   make([]string, 0),
		handlers:  make(map[string]StreamHandler),
		isRunning: false,
		reconnect: true,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// Subscribe 订阅数据流
func (ws *WebSocketClient) Subscribe(stream string, handler StreamHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.streams = append(ws.streams, stream)
	ws.handlers[stream] = handler
	ws.logger.Infof("Subscribed to stream: %s", stream)
}

// Start 启动WebSocket连接
func (ws *WebSocketClient) Start() error {
	ws.mu.Lock()
	if ws.isRunning {
		ws.mu.Unlock()
		return fmt.Errorf("websocket client is already running")
	}
	ws.isRunning = true
	ws.mu.Unlock()

	ws.logger.Info("Starting WebSocket client...")

	// 启动连接和重连逻辑
	go ws.connectionLoop()

	return nil
}

// Stop 停止WebSocket连接
func (ws *WebSocketClient) Stop() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if !ws.isRunning {
		return
	}

	ws.logger.Info("Stopping WebSocket client...")
	ws.isRunning = false
	ws.reconnect = false
	ws.cancel()

	if ws.conn != nil {
		ws.conn.Close()
	}
}

// connectionLoop 连接循环
func (ws *WebSocketClient) connectionLoop() {
	for ws.reconnect {
		select {
		case <-ws.ctx.Done():
			return
		default:
		}

		if err := ws.connect(); err != nil {
			ws.logger.Errorf("Failed to connect: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// 处理消息
		ws.messageLoop()

		// 如果需要重连，等待一段时间
		if ws.reconnect {
			ws.logger.Info("Reconnecting in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}
}

// connect 建立WebSocket连接
func (ws *WebSocketClient) connect() error {
	ws.mu.RLock()
	streams := make([]string, len(ws.streams))
	copy(streams, ws.streams)
	ws.mu.RUnlock()

	if len(streams) == 0 {
		return fmt.Errorf("no streams to subscribe")
	}

	// 构建WebSocket URL
	streamParam := strings.Join(streams, "/")
	u, err := url.Parse(fmt.Sprintf("%s/%s", ws.baseURL, streamParam))
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// 建立连接
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	ws.mu.Lock()
	ws.conn = conn
	ws.mu.Unlock()

	ws.logger.Infof("WebSocket connected to: %s", u.String())
	return nil
}

// messageLoop 消息处理循环
func (ws *WebSocketClient) messageLoop() {
	defer func() {
		ws.mu.Lock()
		if ws.conn != nil {
			ws.conn.Close()
			ws.conn = nil
		}
		ws.mu.Unlock()
	}()

	for {
		select {
		case <-ws.ctx.Done():
			return
		default:
		}

		ws.mu.RLock()
		conn := ws.conn
		ws.mu.RUnlock()

		if conn == nil {
			return
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// 读取消息
		_, message, err := conn.ReadMessage()
		if err != nil {
			ws.logger.Errorf("Failed to read message: %v", err)
			return
		}

		// 处理消息
		if err := ws.handleMessage(message); err != nil {
			ws.logger.Errorf("Failed to handle message: %v", err)
		}
	}
}

// handleMessage 处理接收到的消息
func (ws *WebSocketClient) handleMessage(message []byte) error {
	// 解析基础消息结构
	var baseMsg struct {
		Stream string `json:"stream"`
	}

	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return fmt.Errorf("failed to parse base message: %w", err)
	}

	// 查找对应的处理器
	ws.mu.RLock()
	handler, exists := ws.handlers[baseMsg.Stream]
	ws.mu.RUnlock()

	if !exists {
		// 尝试匹配部分流名称
		for stream, h := range ws.handlers {
			if strings.Contains(baseMsg.Stream, stream) {
				handler = h
				exists = true
				break
			}
		}
	}

	if !exists {
		ws.logger.Debugf("No handler for stream: %s", baseMsg.Stream)
		return nil
	}

	// 根据流类型调用相应的处理器方法
	if strings.Contains(baseMsg.Stream, "kline") {
		var klineData KlineStreamData
		if err := json.Unmarshal(message, &klineData); err != nil {
			return fmt.Errorf("failed to parse kline data: %w", err)
		}
		return handler.HandleKlineData(&klineData)
	} else if strings.Contains(baseMsg.Stream, "ticker") {
		var tickerData TickerStreamData
		if err := json.Unmarshal(message, &tickerData); err != nil {
			return fmt.Errorf("failed to parse ticker data: %w", err)
		}
		return handler.HandleTickerData(&tickerData)
	}

	return nil
}

// IsConnected 检查连接状态
func (ws *WebSocketClient) IsConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.conn != nil && ws.isRunning
}

// GetStreams 获取已订阅的流
func (ws *WebSocketClient) GetStreams() []string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	streams := make([]string, len(ws.streams))
	copy(streams, ws.streams)
	return streams
}

// SetStreamHandler 设置数据流处理器
func (ws *WebSocketClient) SetStreamHandler(handler StreamHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	// 为所有流设置同一个处理器
	for _, stream := range ws.streams {
		ws.handlers[stream] = handler
	}
}

// SubscribeKline 订阅K线数据
func (ws *WebSocketClient) SubscribeKline(symbol, interval string) error {
	stream := fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), interval)
	ws.Subscribe(stream, nil)
	return nil
}

// UnsubscribeKline 取消订阅K线数据
func (ws *WebSocketClient) UnsubscribeKline(symbol, interval string) error {
	stream := fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), interval)
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	// 从streams中移除
	for i, s := range ws.streams {
		if s == stream {
			ws.streams = append(ws.streams[:i], ws.streams[i+1:]...)
			break
		}
	}
	
	// 从handlers中移除
	delete(ws.handlers, stream)
	return nil
}

// SubscribeTicker 订阅价格数据
func (ws *WebSocketClient) SubscribeTicker(symbol string) error {
	stream := fmt.Sprintf("%s@ticker", strings.ToLower(symbol))
	ws.Subscribe(stream, nil)
	return nil
}

// UnsubscribeTicker 取消订阅价格数据
func (ws *WebSocketClient) UnsubscribeTicker(symbol string) error {
	stream := fmt.Sprintf("%s@ticker", strings.ToLower(symbol))
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	// 从streams中移除
	for i, s := range ws.streams {
		if s == stream {
			ws.streams = append(ws.streams[:i], ws.streams[i+1:]...)
			break
		}
	}
	
	// 从handlers中移除
	delete(ws.handlers, stream)
	return nil
}