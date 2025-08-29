package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/config"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
)

// Client Binance API客户端
type Client struct {
	config     *config.BinanceConfig
	logger     logger.Logger
	httpClient *http.Client
	baseURL    string
}

// New 创建新的Binance客户端
func New(cfg *config.BinanceConfig, log logger.Logger) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	
	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("secret key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		if cfg.Testnet {
			baseURL = "https://testnet.binancefuture.com"
		} else {
			baseURL = "https://fapi.binance.com"
		}
	}

	client := &Client{
		config: cfg,
		logger: log,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}

	log.Infof("Binance client initialized (testnet: %v)", cfg.Testnet)
	return client, nil
}

// TestConnection 测试连接
func (c *Client) TestConnection() error {
	_, err := c.GetServerTime()
	return err
}

// GetServerTime 获取服务器时间
func (c *Client) GetServerTime() (int64, error) {
	resp, err := c.makeRequest("GET", "/fapi/v1/time", nil, false)
	if err != nil {
		return 0, err
	}

	var result struct {
		ServerTime int64 `json:"serverTime"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, fmt.Errorf("failed to parse server time: %w", err)
	}

	return result.ServerTime, nil
}

// GetAccountInfo 获取账户信息
func (c *Client) GetAccountInfo() (*AccountInfo, error) {
	resp, err := c.makeRequest("GET", "/fapi/v2/account", nil, true)
	if err != nil {
		return nil, err
	}

	var account AccountInfo
	if err := json.Unmarshal(resp, &account); err != nil {
		return nil, fmt.Errorf("failed to parse account info: %w", err)
	}

	return &account, nil
}

// GetPositions 获取持仓信息
func (c *Client) GetPositions() ([]Position, error) {
	resp, err := c.makeRequest("GET", "/fapi/v2/positionRisk", nil, true)
	if err != nil {
		return nil, err
	}

	var positions []Position
	if err := json.Unmarshal(resp, &positions); err != nil {
		return nil, fmt.Errorf("failed to parse positions: %w", err)
	}

	return positions, nil
}

// GetKlines 获取K线数据
func (c *Client) GetKlines(symbol, interval string, limit int) ([]Kline, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	params.Set("limit", strconv.Itoa(limit))

	resp, err := c.makeRequest("GET", "/fapi/v1/klines", params, false)
	if err != nil {
		return nil, err
	}

	var rawKlines [][]interface{}
	if err := json.Unmarshal(resp, &rawKlines); err != nil {
		return nil, fmt.Errorf("failed to parse klines: %w", err)
	}

	klines := make([]Kline, len(rawKlines))
	for i, raw := range rawKlines {
		kline, err := parseKline(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kline %d: %w", i, err)
		}
		klines[i] = kline
	}

	return klines, nil
}

// PlaceOrder 下单
func (c *Client) PlaceOrder(order *OrderRequest) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", order.Symbol)
	params.Set("side", order.Side)
	params.Set("type", order.Type)
	params.Set("quantity", order.Quantity)
	
	if order.Price != "" {
		params.Set("price", order.Price)
	}
	
	if order.TimeInForce != "" {
		params.Set("timeInForce", order.TimeInForce)
	}
	
	if order.StopPrice != "" {
		params.Set("stopPrice", order.StopPrice)
	}

	resp, err := c.makeRequest("POST", "/fapi/v1/order", params, true)
	if err != nil {
		return nil, err
	}

	var orderResp OrderResponse
	if err := json.Unmarshal(resp, &orderResp); err != nil {
		return nil, fmt.Errorf("failed to parse order response: %w", err)
	}

	return &orderResp, nil
}

// CancelOrder 取消订单
func (c *Client) CancelOrder(symbol string, orderID int64) error {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", strconv.FormatInt(orderID, 10))

	_, err := c.makeRequest("DELETE", "/fapi/v1/order", params, true)
	return err
}

// makeRequest 发送HTTP请求
func (c *Client) makeRequest(method, endpoint string, params url.Values, signed bool) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}

	// 添加时间戳
	if signed {
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		
		// 生成签名
		signature := c.generateSignature(params.Encode())
		params.Set("signature", signature)
	}

	// 构建URL
	var reqURL string
	if method == "GET" || method == "DELETE" {
		reqURL = fmt.Sprintf("%s%s?%s", c.baseURL, endpoint, params.Encode())
	} else {
		reqURL = fmt.Sprintf("%s%s", c.baseURL, endpoint)
	}

	// 创建请求
	var req *http.Request
	var err error
	
	if method == "POST" || method == "PUT" {
		req, err = http.NewRequest(method, reqURL, strings.NewReader(params.Encode()))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequest(method, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
	}

	// 设置请求头
	req.Header.Set("X-MBX-APIKEY", c.config.APIKey)

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return nil, fmt.Errorf("API error: %s (code: %d)", apiErr.Msg, apiErr.Code)
		}
		return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// generateSignature 生成签名
func (c *Client) generateSignature(queryString string) string {
	h := hmac.New(sha256.New, []byte(c.config.SecretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// parseKline 解析K线数据
func parseKline(raw []interface{}) (Kline, error) {
	if len(raw) < 11 {
		return Kline{}, fmt.Errorf("invalid kline data length")
	}

	openTime, ok := raw[0].(float64)
	if !ok {
		return Kline{}, fmt.Errorf("invalid open time")
	}

	open, ok := raw[1].(string)
	if !ok {
		return Kline{}, fmt.Errorf("invalid open price")
	}

	high, ok := raw[2].(string)
	if !ok {
		return Kline{}, fmt.Errorf("invalid high price")
	}

	low, ok := raw[3].(string)
	if !ok {
		return Kline{}, fmt.Errorf("invalid low price")
	}

	close, ok := raw[4].(string)
	if !ok {
		return Kline{}, fmt.Errorf("invalid close price")
	}

	volume, ok := raw[5].(string)
	if !ok {
		return Kline{}, fmt.Errorf("invalid volume")
	}

	closeTime, ok := raw[6].(float64)
	if !ok {
		return Kline{}, fmt.Errorf("invalid close time")
	}

	return Kline{
		OpenTime:  int64(openTime),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
		CloseTime: int64(closeTime),
	}, nil
}