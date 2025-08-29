-- Vegas Dual Tunnel Trading Bot - 数据库表结构设计
-- SQLite数据库初始化脚本

-- 启用外键约束
PRAGMA foreign_keys = ON;

-- ============================================================================
-- 1. 用户设置表 (user_settings)
-- 存储用户的配置信息，包括操作模式、网络环境、仓位设置等
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL UNIQUE,              -- Telegram用户ID
    username TEXT,                              -- Telegram用户名
    
    -- 操作模式配置
    operation_mode TEXT NOT NULL DEFAULT 'signal_only', -- 'signal_only', 'semi_auto', 'full_auto'
    
    -- 网络环境配置
    network_env TEXT NOT NULL DEFAULT 'testnet',        -- 'mainnet', 'testnet'
    
    -- API密钥配置 (加密存储)
    api_key_encrypted TEXT,                     -- 加密后的API Key
    api_secret_encrypted TEXT,                  -- 加密后的API Secret
    api_key_iv TEXT,                           -- 加密初始化向量
    api_secret_iv TEXT,                        -- 加密初始化向量
    
    -- 仓位配置
    position_mode TEXT NOT NULL DEFAULT 'fixed_amount', -- 'fixed_amount', 'percentage'
    position_amount DECIMAL(20,8),             -- 固定金额 (USDT)
    position_percentage DECIMAL(5,2),          -- 账户百分比 (1-100)
    
    -- 风险管理配置
    max_positions INTEGER DEFAULT 3,           -- 最大同时持仓数
    daily_loss_limit DECIMAL(20,8),           -- 日损失限额
    
    -- 通知配置
    enable_notifications BOOLEAN DEFAULT 1,    -- 是否启用通知
    notification_level TEXT DEFAULT 'all',     -- 'all', 'important', 'minimal'
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_user_settings_user_id ON user_settings(user_id);

-- ============================================================================
-- 2. 监控合约表 (monitored_contracts)
-- 存储用户添加的监控合约列表
-- ============================================================================
CREATE TABLE IF NOT EXISTS monitored_contracts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,                     -- 关联用户ID
    symbol TEXT NOT NULL,                      -- 交易对符号 (如: BTCUSDT)
    base_asset TEXT NOT NULL,                  -- 基础资产 (如: BTC)
    quote_asset TEXT NOT NULL,                 -- 计价资产 (如: USDT)
    
    -- 合约状态
    status TEXT NOT NULL DEFAULT 'active',     -- 'active', 'paused', 'stopped'
    
    -- 策略配置 (可为每个合约单独配置)
    custom_position_size DECIMAL(20,8),        -- 自定义仓位大小
    custom_risk_level DECIMAL(3,2),           -- 自定义风险等级 (0.1-2.0)
    
    -- 统计信息
    total_signals INTEGER DEFAULT 0,           -- 总信号数
    total_trades INTEGER DEFAULT 0,            -- 总交易数
    win_trades INTEGER DEFAULT 0,              -- 盈利交易数
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES user_settings(user_id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_monitored_contracts_user_id ON monitored_contracts(user_id);
CREATE INDEX idx_monitored_contracts_symbol ON monitored_contracts(symbol);
CREATE UNIQUE INDEX idx_monitored_contracts_user_symbol ON monitored_contracts(user_id, symbol);

-- ============================================================================
-- 3. 交易记录表 (trades)
-- 存储所有的交易记录，包括开仓、平仓信息
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,                     -- 关联用户ID
    contract_id INTEGER NOT NULL,              -- 关联监控合约ID
    
    -- 交易基本信息
    symbol TEXT NOT NULL,                      -- 交易对
    side TEXT NOT NULL,                        -- 'LONG', 'SHORT'
    status TEXT NOT NULL DEFAULT 'active',     -- 'active', 'closed', 'cancelled'
    
    -- 网络环境
    network_env TEXT NOT NULL,                 -- 'mainnet', 'testnet'
    
    -- 开仓信息
    entry_price DECIMAL(20,8) NOT NULL,        -- 开仓价格
    entry_quantity DECIMAL(20,8) NOT NULL,     -- 开仓数量
    entry_amount DECIMAL(20,8) NOT NULL,       -- 开仓金额 (USDT)
    entry_order_id TEXT,                       -- 币安订单ID
    entry_time DATETIME NOT NULL,              -- 开仓时间
    
    -- 止损止盈设置
    stop_loss_price DECIMAL(20,8),            -- 止损价格
    take_profit_1_price DECIMAL(20,8),        -- TP1价格 (2R)
    take_profit_1_quantity DECIMAL(20,8),     -- TP1数量 (50%)
    take_profit_1_filled BOOLEAN DEFAULT 0,    -- TP1是否已执行
    
    -- 平仓信息
    exit_price DECIMAL(20,8),                 -- 平仓价格
    exit_quantity DECIMAL(20,8),              -- 平仓数量
    exit_amount DECIMAL(20,8),                -- 平仓金额
    exit_order_id TEXT,                       -- 平仓订单ID
    exit_time DATETIME,                       -- 平仓时间
    exit_reason TEXT,                         -- 平仓原因: 'tp1', 'tp2', 'sl', 'manual'
    
    -- 盈亏计算
    realized_pnl DECIMAL(20,8),               -- 已实现盈亏
    realized_pnl_percentage DECIMAL(8,4),     -- 盈亏百分比
    fees DECIMAL(20,8) DEFAULT 0,             -- 手续费
    
    -- 信号信息
    signal_type TEXT,                         -- 信号类型
    signal_strength DECIMAL(3,2),            -- 信号强度 (0-1)
    market_condition TEXT,                    -- 市场状态
    
    -- 风险管理
    risk_reward_ratio DECIMAL(8,4),          -- 风险收益比
    max_drawdown DECIMAL(8,4),               -- 最大回撤
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES user_settings(user_id) ON DELETE CASCADE,
    FOREIGN KEY (contract_id) REFERENCES monitored_contracts(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_trades_user_id ON trades(user_id);
CREATE INDEX idx_trades_symbol ON trades(symbol);
CREATE INDEX idx_trades_status ON trades(status);
CREATE INDEX idx_trades_entry_time ON trades(entry_time);
CREATE INDEX idx_trades_network_env ON trades(network_env);

-- ============================================================================
-- 4. K线数据表 (kline_data)
-- 缓存K线数据，用于策略计算和历史回测
-- ============================================================================
CREATE TABLE IF NOT EXISTS kline_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol TEXT NOT NULL,                      -- 交易对
    interval_type TEXT NOT NULL,               -- 时间周期: '15m', '4h'
    
    -- K线数据
    open_time BIGINT NOT NULL,                 -- 开盘时间 (毫秒时间戳)
    close_time BIGINT NOT NULL,                -- 收盘时间 (毫秒时间戳)
    open_price DECIMAL(20,8) NOT NULL,         -- 开盘价
    high_price DECIMAL(20,8) NOT NULL,         -- 最高价
    low_price DECIMAL(20,8) NOT NULL,          -- 最低价
    close_price DECIMAL(20,8) NOT NULL,        -- 收盘价
    volume DECIMAL(20,8) NOT NULL,             -- 成交量
    quote_volume DECIMAL(20,8) NOT NULL,       -- 成交额
    
    -- EMA指标缓存
    ema_12 DECIMAL(20,8),                      -- EMA 12
    ema_144 DECIMAL(20,8),                     -- EMA 144
    ema_169 DECIMAL(20,8),                     -- EMA 169
    ema_288 DECIMAL(20,8),                     -- EMA 288
    ema_338 DECIMAL(20,8),                     -- EMA 338
    
    -- 市场状态标记
    trend_4h TEXT,                             -- 4H趋势: 'bullish', 'bearish', 'sideways'
    tunnel_position TEXT,                     -- 隧道位置: 'above', 'below', 'inside'
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_kline_symbol_interval ON kline_data(symbol, interval_type);
CREATE INDEX idx_kline_open_time ON kline_data(open_time);
CREATE UNIQUE INDEX idx_kline_unique ON kline_data(symbol, interval_type, open_time);

-- ============================================================================
-- 5. 系统日志表 (system_logs)
-- 记录系统运行日志、错误信息、重要事件
-- ============================================================================
CREATE TABLE IF NOT EXISTS system_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- 日志基本信息
    level TEXT NOT NULL,                       -- 'DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL'
    module TEXT NOT NULL,                      -- 模块名称
    message TEXT NOT NULL,                     -- 日志消息
    
    -- 关联信息
    user_id TEXT,                              -- 关联用户ID (可选)
    trade_id INTEGER,                          -- 关联交易ID (可选)
    symbol TEXT,                               -- 关联交易对 (可选)
    
    -- 错误详情
    error_code TEXT,                           -- 错误代码
    error_details TEXT,                        -- 错误详情 (JSON格式)
    stack_trace TEXT,                          -- 堆栈跟踪
    
    -- 性能指标
    execution_time_ms INTEGER,                 -- 执行时间 (毫秒)
    memory_usage_mb DECIMAL(10,2),            -- 内存使用 (MB)
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES user_settings(user_id) ON DELETE SET NULL,
    FOREIGN KEY (trade_id) REFERENCES trades(id) ON DELETE SET NULL
);

-- 创建索引
CREATE INDEX idx_system_logs_level ON system_logs(level);
CREATE INDEX idx_system_logs_module ON system_logs(module);
CREATE INDEX idx_system_logs_created_at ON system_logs(created_at);
CREATE INDEX idx_system_logs_user_id ON system_logs(user_id);

-- ============================================================================
-- 6. 信号记录表 (trading_signals)
-- 记录所有生成的交易信号，用于分析和统计
-- ============================================================================
CREATE TABLE IF NOT EXISTS trading_signals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- 信号基本信息
    symbol TEXT NOT NULL,                      -- 交易对
    signal_type TEXT NOT NULL,                 -- 'LONG', 'SHORT'
    signal_strength DECIMAL(3,2) NOT NULL,     -- 信号强度 (0-1)
    
    -- 市场数据
    current_price DECIMAL(20,8) NOT NULL,      -- 当前价格
    ema_12 DECIMAL(20,8) NOT NULL,            -- EMA 12
    ema_144 DECIMAL(20,8) NOT NULL,           -- EMA 144
    ema_169 DECIMAL(20,8) NOT NULL,           -- EMA 169
    ema_288 DECIMAL(20,8) NOT NULL,           -- EMA 288
    ema_338 DECIMAL(20,8) NOT NULL,           -- EMA 338
    
    -- 信号条件
    trend_4h TEXT NOT NULL,                    -- 4H趋势状态
    momentum_trigger BOOLEAN NOT NULL,         -- 动能触发
    tunnel_breakout BOOLEAN NOT NULL,          -- 隧道突破
    
    -- 建议价格
    suggested_entry DECIMAL(20,8),            -- 建议入场价
    suggested_stop_loss DECIMAL(20,8),        -- 建议止损价
    suggested_take_profit DECIMAL(20,8),      -- 建议止盈价
    
    -- 执行状态
    is_executed BOOLEAN DEFAULT 0,             -- 是否已执行
    trade_id INTEGER,                          -- 关联交易ID
    execution_delay_ms INTEGER,                -- 执行延迟 (毫秒)
    
    -- 时间戳
    signal_time DATETIME NOT NULL,             -- 信号生成时间
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- 外键约束
    FOREIGN KEY (trade_id) REFERENCES trades(id) ON DELETE SET NULL
);

-- 创建索引
CREATE INDEX idx_trading_signals_symbol ON trading_signals(symbol);
CREATE INDEX idx_trading_signals_signal_time ON trading_signals(signal_time);
CREATE INDEX idx_trading_signals_is_executed ON trading_signals(is_executed);

-- ============================================================================
-- 7. 性能统计表 (performance_stats)
-- 存储用户的交易性能统计数据
-- ============================================================================
CREATE TABLE IF NOT EXISTS performance_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,                     -- 关联用户ID
    
    -- 统计周期
    period_type TEXT NOT NULL,                 -- 'daily', 'weekly', 'monthly', 'yearly'
    period_start DATE NOT NULL,                -- 统计开始日期
    period_end DATE NOT NULL,                  -- 统计结束日期
    
    -- 交易统计
    total_trades INTEGER DEFAULT 0,            -- 总交易数
    winning_trades INTEGER DEFAULT 0,          -- 盈利交易数
    losing_trades INTEGER DEFAULT 0,           -- 亏损交易数
    win_rate DECIMAL(5,2) DEFAULT 0,          -- 胜率 (%)
    
    -- 盈亏统计
    total_pnl DECIMAL(20,8) DEFAULT 0,        -- 总盈亏
    total_fees DECIMAL(20,8) DEFAULT 0,       -- 总手续费
    net_pnl DECIMAL(20,8) DEFAULT 0,          -- 净盈亏
    
    -- 风险指标
    max_drawdown DECIMAL(8,4) DEFAULT 0,      -- 最大回撤 (%)
    sharpe_ratio DECIMAL(8,4),                -- 夏普比率
    profit_factor DECIMAL(8,4),               -- 盈利因子
    
    -- 平均指标
    avg_win_amount DECIMAL(20,8) DEFAULT 0,   -- 平均盈利金额
    avg_loss_amount DECIMAL(20,8) DEFAULT 0,  -- 平均亏损金额
    avg_trade_duration_hours DECIMAL(8,2),    -- 平均持仓时间 (小时)
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES user_settings(user_id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_performance_stats_user_id ON performance_stats(user_id);
CREATE INDEX idx_performance_stats_period ON performance_stats(period_type, period_start);
CREATE UNIQUE INDEX idx_performance_stats_unique ON performance_stats(user_id, period_type, period_start);

-- ============================================================================
-- 8. 创建触发器 - 自动更新时间戳
-- ============================================================================

-- 用户设置表更新触发器
CREATE TRIGGER update_user_settings_timestamp 
    AFTER UPDATE ON user_settings
    FOR EACH ROW
BEGIN
    UPDATE user_settings SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- 监控合约表更新触发器
CREATE TRIGGER update_monitored_contracts_timestamp 
    AFTER UPDATE ON monitored_contracts
    FOR EACH ROW
BEGIN
    UPDATE monitored_contracts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- 交易记录表更新触发器
CREATE TRIGGER update_trades_timestamp 
    AFTER UPDATE ON trades
    FOR EACH ROW
BEGIN
    UPDATE trades SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- K线数据表更新触发器
CREATE TRIGGER update_kline_data_timestamp 
    AFTER UPDATE ON kline_data
    FOR EACH ROW
BEGIN
    UPDATE kline_data SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- 性能统计表更新触发器
CREATE TRIGGER update_performance_stats_timestamp 
    AFTER UPDATE ON performance_stats
    FOR EACH ROW
BEGIN
    UPDATE performance_stats SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ============================================================================
-- 9. 初始化数据
-- ============================================================================

-- 插入默认的系统配置 (如果需要)
-- INSERT INTO user_settings (user_id, username, operation_mode) 
-- VALUES ('system', 'system', 'signal_only');

-- ============================================================================
-- 10. 数据库版本管理
-- ============================================================================
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 插入当前版本
INSERT OR REPLACE INTO schema_migrations (version) VALUES ('1.0.0');

-- ============================================================================
-- 数据库初始化完成
-- ============================================================================