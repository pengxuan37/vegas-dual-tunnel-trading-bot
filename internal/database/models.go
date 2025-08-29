package database

import (
	"database/sql"
	"fmt"
	"time"
)

// UserConfig 用户配置模型
type UserConfig struct {
	ID               int       `json:"id"`
	UserID           int64     `json:"user_id"`
	Username         string    `json:"username"`
	ChatID           int64     `json:"chat_id"`
	APIKey           string    `json:"api_key"`
	APISecret        string    `json:"api_secret"`
	Testnet          bool      `json:"testnet"`
	MaxPositionSize  float64   `json:"max_position_size"`
	RiskPercentage   float64   `json:"risk_percentage"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// WatchlistItem 监控列表项
type WatchlistItem struct {
	ID        int       `json:"id"`
	UserID    int64     `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Interval  string    `json:"interval"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Trade 交易记录
type Trade struct {
	ID              int       `json:"id"`
	UserID          int64     `json:"user_id"`
	Symbol          string    `json:"symbol"`
	OrderID         string    `json:"order_id"`
	ClientOrderID   string    `json:"client_order_id"`
	Side            string    `json:"side"`
	Type            string    `json:"type"`
	Quantity        float64   `json:"quantity"`
	Price           float64   `json:"price"`
	StopPrice       float64   `json:"stop_price"`
	Status          string    `json:"status"`
	FilledQuantity  float64   `json:"filled_quantity"`
	AvgPrice        float64   `json:"avg_price"`
	Commission      float64   `json:"commission"`
	RealizedPnl     float64   `json:"realized_pnl"`
	StrategyType    string    `json:"strategy_type"`
	SignalType      string    `json:"signal_type"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Signal 策略信号
type Signal struct {
	ID           int       `json:"id"`
	UserID       int64     `json:"user_id"`
	Symbol       string    `json:"symbol"`
	Interval     string    `json:"interval"`
	StrategyType string    `json:"strategy_type"`
	SignalType   string    `json:"signal_type"`
	Price        float64   `json:"price"`
	Volume       float64   `json:"volume"`
	Confidence   float64   `json:"confidence"`
	Metadata     string    `json:"metadata"`
	IsProcessed  bool      `json:"is_processed"`
	CreatedAt    time.Time `json:"created_at"`
}

// Position 持仓记录
type Position struct {
	ID              int        `json:"id"`
	UserID          int64      `json:"user_id"`
	Symbol          string     `json:"symbol"`
	Side            string     `json:"side"`
	Size            float64    `json:"size"`
	EntryPrice      float64    `json:"entry_price"`
	MarkPrice       float64    `json:"mark_price"`
	UnrealizedPnl   float64    `json:"unrealized_pnl"`
	Percentage      float64    `json:"percentage"`
	StopLossPrice   float64    `json:"stop_loss_price"`
	TakeProfitPrice float64    `json:"take_profit_price"`
	StrategyType    string     `json:"strategy_type"`
	IsOpen          bool       `json:"is_open"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
}

// SystemLog 系统日志
type SystemLog struct {
	ID           int       `json:"id"`
	Level        string    `json:"level"`
	Message      string    `json:"message"`
	Module       string    `json:"module"`
	UserID       *int64    `json:"user_id,omitempty"`
	ErrorDetails string    `json:"error_details"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserConfigRepository 用户配置仓库
type UserConfigRepository struct {
	db *sql.DB
}

// NewUserConfigRepository 创建用户配置仓库
func NewUserConfigRepository(db *sql.DB) *UserConfigRepository {
	return &UserConfigRepository{db: db}
}

// GetByUserID 根据用户ID获取配置
func (r *UserConfigRepository) GetByUserID(userID int64) (*UserConfig, error) {
	query := `
		SELECT id, user_id, username, chat_id, api_key, api_secret, testnet, 
		       max_position_size, risk_percentage, is_active, created_at, updated_at
		FROM user_configs WHERE user_id = ?
	`

	var config UserConfig
	err := r.db.QueryRow(query, userID).Scan(
		&config.ID, &config.UserID, &config.Username, &config.ChatID,
		&config.APIKey, &config.APISecret, &config.Testnet,
		&config.MaxPositionSize, &config.RiskPercentage, &config.IsActive,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}

	return &config, nil
}

// Create 创建用户配置
func (r *UserConfigRepository) Create(config *UserConfig) error {
	query := `
		INSERT INTO user_configs (user_id, username, chat_id, api_key, api_secret, testnet, 
		                         max_position_size, risk_percentage, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		config.UserID, config.Username, config.ChatID, config.APIKey, config.APISecret,
		config.Testnet, config.MaxPositionSize, config.RiskPercentage, config.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to create user config: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	config.ID = int(id)
	return nil
}

// Update 更新用户配置
func (r *UserConfigRepository) Update(config *UserConfig) error {
	query := `
		UPDATE user_configs 
		SET username = ?, chat_id = ?, api_key = ?, api_secret = ?, testnet = ?,
		    max_position_size = ?, risk_percentage = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`

	_, err := r.db.Exec(query,
		config.Username, config.ChatID, config.APIKey, config.APISecret, config.Testnet,
		config.MaxPositionSize, config.RiskPercentage, config.IsActive, config.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user config: %w", err)
	}

	return nil
}

// TradeRepository 交易记录仓库
type TradeRepository struct {
	db *sql.DB
}

// NewTradeRepository 创建交易记录仓库
func NewTradeRepository(db *sql.DB) *TradeRepository {
	return &TradeRepository{db: db}
}

// Create 创建交易记录
func (r *TradeRepository) Create(trade *Trade) error {
	query := `
		INSERT INTO trades (user_id, symbol, order_id, client_order_id, side, type, quantity, 
		                   price, stop_price, status, filled_quantity, avg_price, commission, 
		                   realized_pnl, strategy_type, signal_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		trade.UserID, trade.Symbol, trade.OrderID, trade.ClientOrderID, trade.Side, trade.Type,
		trade.Quantity, trade.Price, trade.StopPrice, trade.Status, trade.FilledQuantity,
		trade.AvgPrice, trade.Commission, trade.RealizedPnl, trade.StrategyType, trade.SignalType,
	)

	if err != nil {
		return fmt.Errorf("failed to create trade: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	trade.ID = int(id)
	return nil
}

// GetByUserID 获取用户的交易记录
func (r *TradeRepository) GetByUserID(userID int64, limit int) ([]*Trade, error) {
	query := `
		SELECT id, user_id, symbol, order_id, client_order_id, side, type, quantity, 
		       price, stop_price, status, filled_quantity, avg_price, commission, 
		       realized_pnl, strategy_type, signal_type, created_at, updated_at
		FROM trades WHERE user_id = ? ORDER BY created_at DESC LIMIT ?
	`

	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []*Trade
	for rows.Next() {
		var trade Trade
		err := rows.Scan(
			&trade.ID, &trade.UserID, &trade.Symbol, &trade.OrderID, &trade.ClientOrderID,
			&trade.Side, &trade.Type, &trade.Quantity, &trade.Price, &trade.StopPrice,
			&trade.Status, &trade.FilledQuantity, &trade.AvgPrice, &trade.Commission,
			&trade.RealizedPnl, &trade.StrategyType, &trade.SignalType,
			&trade.CreatedAt, &trade.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}
		trades = append(trades, &trade)
	}

	return trades, nil
}

// PositionRepository 持仓记录仓库
type PositionRepository struct {
	db *sql.DB
}

// NewPositionRepository 创建持仓记录仓库
func NewPositionRepository(db *sql.DB) *PositionRepository {
	return &PositionRepository{db: db}
}

// GetOpenPositions 获取用户的开放持仓
func (r *PositionRepository) GetOpenPositions(userID int64) ([]*Position, error) {
	query := `
		SELECT id, user_id, symbol, side, size, entry_price, mark_price, unrealized_pnl, 
		       percentage, stop_loss_price, take_profit_price, strategy_type, is_open, 
		       created_at, updated_at, closed_at
		FROM positions WHERE user_id = ? AND is_open = 1
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions: %w", err)
	}
	defer rows.Close()

	var positions []*Position
	for rows.Next() {
		var position Position
		err := rows.Scan(
			&position.ID, &position.UserID, &position.Symbol, &position.Side, &position.Size,
			&position.EntryPrice, &position.MarkPrice, &position.UnrealizedPnl, &position.Percentage,
			&position.StopLossPrice, &position.TakeProfitPrice, &position.StrategyType,
			&position.IsOpen, &position.CreatedAt, &position.UpdatedAt, &position.ClosedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		positions = append(positions, &position)
	}

	return positions, nil
}

// Create 创建持仓记录
func (r *PositionRepository) Create(position *Position) error {
	query := `
		INSERT INTO positions (user_id, symbol, side, size, entry_price, mark_price, 
		                      unrealized_pnl, percentage, stop_loss_price, take_profit_price, 
		                      strategy_type, is_open)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		position.UserID, position.Symbol, position.Side, position.Size, position.EntryPrice,
		position.MarkPrice, position.UnrealizedPnl, position.Percentage, position.StopLossPrice,
		position.TakeProfitPrice, position.StrategyType, position.IsOpen,
	)

	if err != nil {
		return fmt.Errorf("failed to create position: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	position.ID = int(id)
	return nil
}

// SignalRepository 信号仓库
type SignalRepository struct {
	db *sql.DB
}

// NewSignalRepository 创建信号仓库
func NewSignalRepository(db *sql.DB) *SignalRepository {
	return &SignalRepository{db: db}
}

// Create 创建信号
func (r *SignalRepository) Create(signal *Signal) error {
	query := `
		INSERT INTO signals (user_id, symbol, interval, strategy_type, signal_type, price, 
		                    volume, confidence, metadata, is_processed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		signal.UserID, signal.Symbol, signal.Interval, signal.StrategyType, signal.SignalType,
		signal.Price, signal.Volume, signal.Confidence, signal.Metadata, signal.IsProcessed,
	)

	if err != nil {
		return fmt.Errorf("failed to create signal: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	signal.ID = int(id)
	return nil
}

// GetUnprocessed 获取未处理的信号
func (r *SignalRepository) GetUnprocessed(userID int64) ([]*Signal, error) {
	query := `
		SELECT id, user_id, symbol, interval, strategy_type, signal_type, price, 
		       volume, confidence, metadata, is_processed, created_at
		FROM signals WHERE user_id = ? AND is_processed = 0 ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query signals: %w", err)
	}
	defer rows.Close()

	var signals []*Signal
	for rows.Next() {
		var signal Signal
		err := rows.Scan(
			&signal.ID, &signal.UserID, &signal.Symbol, &signal.Interval, &signal.StrategyType,
			&signal.SignalType, &signal.Price, &signal.Volume, &signal.Confidence,
			&signal.Metadata, &signal.IsProcessed, &signal.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan signal: %w", err)
		}
		signals = append(signals, &signal)
	}

	return signals, nil
}

// MarkProcessed 标记信号为已处理
func (r *SignalRepository) MarkProcessed(signalID int) error {
	query := "UPDATE signals SET is_processed = 1 WHERE id = ?"
	_, err := r.db.Exec(query, signalID)
	if err != nil {
		return fmt.Errorf("failed to mark signal as processed: %w", err)
	}
	return nil
}