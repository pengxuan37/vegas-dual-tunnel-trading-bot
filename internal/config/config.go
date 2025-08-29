package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 应用程序配置
type Config struct {
	Telegram TelegramConfig `json:"telegram"`
	Binance  BinanceConfig  `json:"binance"`
	Database DatabaseConfig `json:"database"`
	Trading  TradingConfig  `json:"trading"`
	Logging  LoggingConfig  `json:"logging"`
}

// TelegramConfig Telegram机器人配置
type TelegramConfig struct {
	BotToken    string   `json:"bot_token"`
	ChatIDs     []int64  `json:"chat_ids"`     // 允许的聊天ID列表
	AdminChatID int64    `json:"admin_chat_id"` // 管理员聊天ID
	WebhookURL  string   `json:"webhook_url"`   // Webhook URL（可选）
	Timeout     int      `json:"timeout"`       // 请求超时时间（秒）
}

// BinanceConfig 币安API配置
type BinanceConfig struct {
	APIKey      string `json:"api_key"`
	SecretKey   string `json:"secret_key"`
	Testnet     bool   `json:"testnet"`      // 是否使用测试网
	BaseURL     string `json:"base_url"`     // API基础URL
	WSURL       string `json:"ws_url"`       // WebSocket URL
	Timeout     int    `json:"timeout"`      // 请求超时时间（秒）
	RateLimit   int    `json:"rate_limit"`   // 请求频率限制（每分钟）
	RecvWindow  int    `json:"recv_window"`  // 接收窗口时间（毫秒）
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path            string `json:"path"`              // 数据库文件路径
	MaxOpenConns    int    `json:"max_open_conns"`    // 最大打开连接数
	MaxIdleConns    int    `json:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime int    `json:"conn_max_lifetime"` // 连接最大生存时间（秒）
	BackupInterval  int    `json:"backup_interval"`   // 备份间隔（小时）
	BackupPath      string `json:"backup_path"`       // 备份路径
}

// TradingConfig 交易配置
type TradingConfig struct {
	DefaultRiskPercent   float64 `json:"default_risk_percent"`   // 默认风险百分比
	MaxPositions         int     `json:"max_positions"`          // 最大持仓数量
	MinOrderValue        float64 `json:"min_order_value"`        // 最小订单价值（USDT）
	MaxOrderValue        float64 `json:"max_order_value"`        // 最大订单价值（USDT）
	DefaultLeverage      int     `json:"default_leverage"`       // 默认杠杆倍数
	SlippageTolerance    float64 `json:"slippage_tolerance"`     // 滑点容忍度
	OrderTimeout         int     `json:"order_timeout"`          // 订单超时时间（秒）
	PriceCheckInterval   int     `json:"price_check_interval"`   // 价格检查间隔（秒）
	EmergencyStopEnabled bool    `json:"emergency_stop_enabled"` // 紧急停止开关
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `json:"level"`       // 日志级别
	FilePath   string `json:"file_path"`   // 日志文件路径
	MaxSize    int    `json:"max_size"`    // 最大文件大小（MB）
	MaxBackups int    `json:"max_backups"` // 最大备份文件数
	MaxAge     int    `json:"max_age"`     // 最大保存天数
	Compress   bool   `json:"compress"`    // 是否压缩
	Console    bool   `json:"console"`     // 是否输出到控制台
}

// Load 从文件加载配置
func Load(configPath string) (*Config, error) {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := getDefaultConfig()
		if err := Save(defaultConfig, configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return defaultConfig, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 从环境变量覆盖敏感配置
	if err := loadFromEnv(&config); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// 验证配置
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Save 保存配置到文件
func Save(config *Config, configPath string) error {
	// 创建目录
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *Config {
	return &Config{
		Telegram: TelegramConfig{
			BotToken:    "", // 需要从环境变量设置
			ChatIDs:     []int64{},
			AdminChatID: 0,
			WebhookURL:  "",
			Timeout:     30,
		},
		Binance: BinanceConfig{
			APIKey:     "", // 需要从环境变量设置
			SecretKey:  "", // 需要从环境变量设置
			Testnet:    true,
			BaseURL:    "https://testnet.binancefuture.com",
			WSURL:      "wss://stream.binancefuture.com",
			Timeout:    10,
			RateLimit:  1200,
			RecvWindow: 5000,
		},
		Database: DatabaseConfig{
			Path:            "./data/trading.db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 3600,
			BackupInterval:  24,
			BackupPath:      "./data/backups",
		},
		Trading: TradingConfig{
			DefaultRiskPercent:   2.0,
			MaxPositions:         5,
			MinOrderValue:        10.0,
			MaxOrderValue:        1000.0,
			DefaultLeverage:      1,
			SlippageTolerance:    0.1,
			OrderTimeout:         60,
			PriceCheckInterval:   5,
			EmergencyStopEnabled: false,
		},
		Logging: LoggingConfig{
			Level:      "info",
			FilePath:   "./logs/trading.log",
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
			Console:    true,
		},
	}
}

// loadFromEnv 从环境变量加载敏感配置
func loadFromEnv(config *Config) error {
	// Telegram配置
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		config.Telegram.BotToken = token
	}

	if chatIDs := os.Getenv("TELEGRAM_CHAT_IDS"); chatIDs != "" {
		ids := strings.Split(chatIDs, ",")
		config.Telegram.ChatIDs = make([]int64, 0, len(ids))
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if chatID, err := strconv.ParseInt(id, 10, 64); err == nil {
				config.Telegram.ChatIDs = append(config.Telegram.ChatIDs, chatID)
			}
		}
	}

	if adminID := os.Getenv("TELEGRAM_ADMIN_CHAT_ID"); adminID != "" {
		if id, err := strconv.ParseInt(adminID, 10, 64); err == nil {
			config.Telegram.AdminChatID = id
		}
	}

	// 币安配置
	if apiKey := os.Getenv("BINANCE_API_KEY"); apiKey != "" {
		config.Binance.APIKey = apiKey
	}

	if secretKey := os.Getenv("BINANCE_SECRET_KEY"); secretKey != "" {
		config.Binance.SecretKey = secretKey
	}

	if testnet := os.Getenv("BINANCE_TESTNET"); testnet != "" {
		if isTestnet, err := strconv.ParseBool(testnet); err == nil {
			config.Binance.Testnet = isTestnet
			if isTestnet {
				config.Binance.BaseURL = "https://testnet.binancefuture.com"
				config.Binance.WSURL = "wss://stream.binancefuture.com"
			} else {
				config.Binance.BaseURL = "https://fapi.binance.com"
				config.Binance.WSURL = "wss://fstream.binance.com"
			}
		}
	}

	return nil
}

// validate 验证配置
func validate(config *Config) error {
	// 验证Telegram配置
	if config.Telegram.BotToken == "" {
		return fmt.Errorf("telegram bot token is required")
	}

	if len(config.Telegram.ChatIDs) == 0 {
		return fmt.Errorf("at least one telegram chat ID is required")
	}

	// 验证币安配置
	if config.Binance.APIKey == "" {
		return fmt.Errorf("binance API key is required")
	}

	if config.Binance.SecretKey == "" {
		return fmt.Errorf("binance secret key is required")
	}

	// 验证交易配置
	if config.Trading.DefaultRiskPercent <= 0 || config.Trading.DefaultRiskPercent > 100 {
		return fmt.Errorf("default risk percent must be between 0 and 100")
	}

	if config.Trading.MaxPositions <= 0 {
		return fmt.Errorf("max positions must be greater than 0")
	}

	if config.Trading.MinOrderValue <= 0 {
		return fmt.Errorf("min order value must be greater than 0")
	}

	if config.Trading.MaxOrderValue <= config.Trading.MinOrderValue {
		return fmt.Errorf("max order value must be greater than min order value")
	}

	return nil
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return "./config/config.json"
}

// IsProduction 判断是否为生产环境
func (c *Config) IsProduction() bool {
	return !c.Binance.Testnet
}

// GetBinanceWSURL 获取币安WebSocket URL
func (c *Config) GetBinanceWSURL() string {
	if c.Binance.Testnet {
		return "wss://stream.binancefuture.com"
	}
	return "wss://fstream.binance.com"
}

// GetBinanceBaseURL 获取币安API基础URL
func (c *Config) GetBinanceBaseURL() string {
	if c.Binance.Testnet {
		return "https://testnet.binancefuture.com"
	}
	return "https://fapi.binance.com"
}