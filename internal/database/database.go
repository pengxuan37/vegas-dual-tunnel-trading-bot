package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
	_ "github.com/mattn/go-sqlite3"
)

// Database 数据库客户端
type Database struct {
	db     *sql.DB
	logger logger.Logger
}

// New 创建新的数据库实例
func New(dbPath string, log logger.Logger) (*Database, error) {
	// 确保数据库文件路径是绝对路径
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:     db,
		logger: log,
	}

	// 初始化数据库表
	if err := database.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	log.Infof("Database initialized: %s", absPath)
	return database, nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// initTables 初始化数据库表
func (d *Database) initTables() error {
	// 用户配置表
	userConfigSQL := `
	CREATE TABLE IF NOT EXISTS user_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER UNIQUE NOT NULL,
		username TEXT,
		chat_id INTEGER,
		api_key TEXT,
		api_secret TEXT,
		testnet BOOLEAN DEFAULT 1,
		max_position_size REAL DEFAULT 100.0,
		risk_percentage REAL DEFAULT 2.0,
		is_active BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	// 监控列表表
	watchlistSQL := `
	CREATE TABLE IF NOT EXISTS watchlist (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		symbol TEXT NOT NULL,
		interval TEXT DEFAULT '1h',
		is_active BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES user_configs(user_id),
		UNIQUE(user_id, symbol)
	);
	`

	// 交易记录表
	tradesSQL := `
	CREATE TABLE IF NOT EXISTS trades (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		symbol TEXT NOT NULL,
		order_id TEXT,
		client_order_id TEXT,
		side TEXT NOT NULL, -- BUY/SELL
		type TEXT NOT NULL, -- MARKET/LIMIT
		quantity REAL NOT NULL,
		price REAL,
		stop_price REAL,
		status TEXT NOT NULL, -- NEW/FILLED/CANCELED/REJECTED
		filled_quantity REAL DEFAULT 0,
		avg_price REAL,
		commission REAL DEFAULT 0,
		realized_pnl REAL DEFAULT 0,
		strategy_type TEXT, -- vegas_tunnel
		signal_type TEXT, -- entry/exit/stop_loss/take_profit
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES user_configs(user_id)
	);
	`

	// 策略信号表
	signalsSQL := `
	CREATE TABLE IF NOT EXISTS signals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		symbol TEXT NOT NULL,
		interval TEXT NOT NULL,
		strategy_type TEXT NOT NULL,
		signal_type TEXT NOT NULL, -- BUY/SELL
		price REAL NOT NULL,
		volume REAL,
		confidence REAL, -- 信号强度 0-1
		metadata TEXT, -- JSON格式的额外信息
		is_processed BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES user_configs(user_id)
	);
	`

	// 持仓记录表
	positionsSQL := `
	CREATE TABLE IF NOT EXISTS positions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		symbol TEXT NOT NULL,
		side TEXT NOT NULL, -- LONG/SHORT
		size REAL NOT NULL,
		entry_price REAL NOT NULL,
		mark_price REAL,
		unrealized_pnl REAL DEFAULT 0,
		percentage REAL DEFAULT 0,
		stop_loss_price REAL,
		take_profit_price REAL,
		strategy_type TEXT,
		is_open BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		closed_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES user_configs(user_id)
	);
	`

	// 系统日志表
	logsSQL := `
	CREATE TABLE IF NOT EXISTS system_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		module TEXT,
		user_id INTEGER,
		error_details TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	// 执行所有建表语句
	tables := []string{
		userConfigSQL,
		watchlistSQL,
		tradesSQL,
		signalsSQL,
		positionsSQL,
		logsSQL,
	}

	for _, tableSQL := range tables {
		if _, err := d.db.Exec(tableSQL); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_trades_user_symbol ON trades(user_id, symbol);",
		"CREATE INDEX IF NOT EXISTS idx_trades_created_at ON trades(created_at);",
		"CREATE INDEX IF NOT EXISTS idx_signals_user_symbol ON signals(user_id, symbol);",
		"CREATE INDEX IF NOT EXISTS idx_signals_created_at ON signals(created_at);",
		"CREATE INDEX IF NOT EXISTS idx_positions_user_symbol ON positions(user_id, symbol);",
		"CREATE INDEX IF NOT EXISTS idx_logs_created_at ON system_logs(created_at);",
	}

	for _, indexSQL := range indexes {
		if _, err := d.db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	d.logger.Info("Database tables initialized successfully")
	return nil
}

// GetDB 获取数据库连接
func (d *Database) GetDB() *sql.DB {
	return d.db
}

// Health 检查数据库健康状态
func (d *Database) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return d.db.PingContext(ctx)
}