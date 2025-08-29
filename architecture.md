# Vegas Dual Tunnel Trading Bot - 项目架构设计

## 项目概述

本项目是一个基于Go语言开发的币安交易机器人，实现维加斯双隧道交易策略，通过Telegram机器人提供用户交互界面。

## 整体架构

```
维加斯双隧道交易机器人
├── 用户交互层 (Telegram Bot)
├── 业务逻辑层 (Strategy Engine)
├── 数据访问层 (Data Layer)
└── 外部服务层 (External APIs)
```

## 目录结构设计

```
vegas-dual-tunnel-trading-bot/
├── cmd/                          # 应用程序入口
│   └── bot/
│       └── main.go              # 主程序入口
├── internal/                     # 内部包，不对外暴露
│   ├── config/                  # 配置管理
│   │   ├── config.go           # 配置结构体和加载逻辑
│   │   └── crypto.go           # API密钥加密/解密
│   ├── database/                # 数据库层
│   │   ├── models/             # 数据模型
│   │   │   ├── user.go         # 用户配置模型
│   │   │   ├── contract.go     # 监控合约模型
│   │   │   ├── trade.go        # 交易记录模型
│   │   │   └── kline.go        # K线数据模型
│   │   ├── migrations/         # 数据库迁移
│   │   │   └── init.sql        # 初始化SQL
│   │   └── db.go              # 数据库连接和操作
│   ├── telegram/               # Telegram机器人
│   │   ├── bot.go             # 机器人主逻辑
│   │   ├── handlers/          # 指令处理器
│   │   │   ├── commands.go    # 基础指令处理
│   │   │   ├── settings.go    # 设置相关指令
│   │   │   └── trading.go     # 交易相关指令
│   │   └── notifications/     # 通知模块
│   │       ├── templates.go   # 消息模板
│   │       └── sender.go      # 消息发送器
│   ├── binance/                # 币安API客户端
│   │   ├── client.go          # HTTP客户端
│   │   ├── websocket.go       # WebSocket客户端
│   │   ├── market.go          # 市场数据API
│   │   ├── trading.go         # 交易API
│   │   └── types.go           # API数据类型
│   ├── strategy/               # 交易策略
│   │   ├── vegas/             # 维加斯双隧道策略
│   │   │   ├── calculator.go  # EMA计算器
│   │   │   ├── analyzer.go    # 市场分析器
│   │   │   ├── signals.go     # 信号生成器
│   │   │   └── types.go       # 策略相关类型
│   │   └── engine.go          # 策略引擎
│   ├── trading/                # 交易执行
│   │   ├── executor.go        # 交易执行器
│   │   ├── monitor.go         # 仓位监控器
│   │   ├── risk.go            # 风险管理
│   │   └── orders.go          # 订单管理
│   ├── data/                   # 数据管理
│   │   ├── manager.go         # 数据管理器
│   │   ├── kline.go           # K线数据处理
│   │   └── cache.go           # 数据缓存
│   └── utils/                  # 工具包
│       ├── logger.go          # 日志工具
│       ├── errors.go          # 错误处理
│       ├── math.go            # 数学计算工具
│       └── time.go            # 时间处理工具
├── pkg/                        # 可对外暴露的包
│   └── events/                 # 事件系统
│       ├── types.go           # 事件类型定义
│       └── bus.go             # 事件总线
├── configs/                    # 配置文件
│   ├── config.yaml            # 主配置文件
│   └── config.example.yaml    # 配置文件示例
├── scripts/                    # 脚本文件
│   ├── build.sh              # 构建脚本
│   ├── deploy.sh             # 部署脚本
│   └── migrate.sh            # 数据库迁移脚本
├── docs/                       # 文档
│   ├── api.md                # API文档
│   ├── deployment.md         # 部署文档
│   └── strategy.md           # 策略说明
├── tests/                      # 测试文件
│   ├── integration/          # 集成测试
│   └── unit/                 # 单元测试
├── go.mod                      # Go模块文件
├── go.sum                      # 依赖校验文件
├── Makefile                    # 构建配置
├── README.md                   # 项目说明
└── .env.example               # 环境变量示例
```

## 核心模块设计

### 1. 配置管理模块 (internal/config)
- **职责**: 管理应用配置、API密钥加密存储
- **核心文件**: 
  - `config.go`: 配置结构体定义和加载
  - `crypto.go`: API密钥加密/解密逻辑

### 2. 数据库模块 (internal/database)
- **职责**: 数据持久化、模型定义、数据库迁移
- **核心表**: 
  - `user_settings`: 用户配置
  - `monitored_contracts`: 监控合约
  - `trades`: 交易记录
  - `kline_data`: K线数据缓存
  - `system_logs`: 系统日志

### 3. Telegram机器人模块 (internal/telegram)
- **职责**: 用户交互、指令处理、消息通知
- **核心组件**:
  - 指令处理器: 处理用户指令
  - 通知系统: 发送交易信号和状态通知
  - 权限控制: 确保只有授权用户可操作

### 4. 币安API模块 (internal/binance)
- **职责**: 与币安交易所交互
- **核心功能**:
  - 市场数据获取 (REST API)
  - 实时数据订阅 (WebSocket)
  - 交易执行 (下单、查询、取消)
  - 主网/测试网环境切换

### 5. 交易策略模块 (internal/strategy)
- **职责**: 实现维加斯双隧道策略
- **核心组件**:
  - EMA计算器: 计算各周期EMA指标
  - 市场分析器: 判断市场趋势状态
  - 信号生成器: 生成买卖信号

### 6. 交易执行模块 (internal/trading)
- **职责**: 执行交易操作、风险管理
- **核心功能**:
  - 开仓执行: 根据信号自动开仓
  - 仓位监控: 监控TP/SL条件
  - 风险控制: 仓位大小计算、资金检查

### 7. 数据管理模块 (internal/data)
- **职责**: 多时间周期数据管理
- **核心功能**:
  - K线数据缓存和同步
  - 内存管理和数据清理
  - 实时数据更新

## 数据流设计

```
用户指令 → Telegram Bot → 业务逻辑处理 → 数据库操作
                    ↓
币安WebSocket → 数据管理器 → 策略引擎 → 信号生成
                    ↓
信号事件 → 交易执行器 → 币安API → 结果通知
```

## 事件驱动架构

使用事件总线模式，实现模块间的松耦合通信：

- **K线更新事件**: WebSocket接收到新K线数据
- **信号生成事件**: 策略引擎生成交易信号
- **交易执行事件**: 开仓/平仓操作完成
- **通知事件**: 需要发送用户通知

## 并发设计

- **WebSocket连接**: 独立goroutine处理实时数据
- **策略计算**: 每个合约独立goroutine计算
- **仓位监控**: 独立goroutine监控所有活跃仓位
- **通知发送**: 异步发送，避免阻塞主流程

## 错误处理策略

- **分层错误处理**: 每层定义特定错误类型
- **重试机制**: API调用失败自动重试
- **降级策略**: 关键服务不可用时的备用方案
- **错误通知**: 严重错误及时通知用户

## 安全设计

- **API密钥加密**: 使用AES加密存储API密钥
- **权限控制**: Telegram用户ID白名单
- **网络隔离**: 主网/测试网严格隔离
- **日志脱敏**: 敏感信息不记录到日志

## 性能优化

- **连接池**: 复用HTTP连接
- **数据缓存**: 缓存常用的K线数据
- **批量操作**: 数据库批量插入/更新
- **内存管理**: 定期清理过期数据

这个架构设计确保了系统的可扩展性、可维护性和稳定性，为后续的开发工作提供了清晰的指导。