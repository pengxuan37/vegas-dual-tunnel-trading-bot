# Vegas Dual Tunnel Trading Bot

基于维加斯双隧道策略的加密货币自动交易机器人，支持Telegram通知和Binance期货交易。

## 功能特性

- 🎯 **维加斯双隧道策略**: 基于EMA338指标的多时间周期分析
- 📱 **Telegram集成**: 实时交易信号推送和远程控制
- 💹 **Binance期货**: 支持主网和测试网自动切换
- 🔄 **自动交易**: 智能开仓、止损止盈和仓位管理
- 📊 **实时监控**: WebSocket实时数据流和性能统计
- 🛡️ **风险控制**: 多重安全机制和错误恢复
- 📈 **回测功能**: 历史数据回测和策略优化

## 快速开始

### 环境要求

- Go 1.21+
- SQLite3
- Telegram Bot Token
- Binance API密钥

### 安装步骤

1. **克隆项目**
   ```bash
   git clone https://github.com/pengxuan37/vegas-dual-tunnel-trading-bot.git
   cd vegas-dual-tunnel-trading-bot
   ```

2. **初始化项目**
   ```bash
   make init
   ```

3. **配置设置**
   ```bash
   # 复制配置文件模板
   cp config.json.example config.json
   
   # 编辑配置文件
   vim config.json
   ```
   
   **重要配置项说明：**
   - `binance.api_key`: Binance API密钥
   - `binance.secret_key`: Binance API私钥
   - `binance.testnet`: 是否使用测试网（建议先用测试网）
   - `telegram.bot_token`: Telegram机器人Token
   - `telegram.admin_chat_id`: 管理员聊天ID
   - `trading.default_quantity`: 默认交易数量
   - `strategy.vegas_tunnel.enabled`: 是否启用维加斯隧道策略

4. **构建运行**
   ```bash
   # 构建项目
   make build
   
   # 运行项目
   make run
   
   # 或开发模式（热重载）
   make dev
   ```

### 配置说明

编辑 `config.json` 文件，填入以下关键信息：

```json
{
  "telegram": {
    "bot_token": "YOUR_BOT_TOKEN",
    "admin_chat_id": "YOUR_CHAT_ID"
  },
  "binance": {
    "api_key": "YOUR_API_KEY",
    "secret_key": "YOUR_SECRET_KEY",
    "testnet": true
  },
  "trading": {
    "default_quantity": "0.01",
    "max_positions": 5,
    "risk_per_trade": "0.02"
  },
  "strategy": {
    "vegas_tunnel": {
      "enabled": true,
      "ema_short_period": 12,
      "ema_long_period": 144
    }
  }
}
```

**获取配置信息：**
- Telegram Bot Token: 从 [@BotFather](https://t.me/BotFather) 获取
- Telegram Chat ID: 发送消息给 [@userinfobot](https://t.me/userinfobot) 获取
- Binance API: 在 [Binance API管理](https://www.binance.com/cn/my/settings/api-management) 创建

## 项目结构

```
.
├── cmd/                    # 命令行工具
├── internal/               # 内部包
│   ├── app/               # 应用主逻辑
│   ├── config/            # 配置管理
│   ├── database/          # 数据库操作
│   ├── telegram/          # Telegram机器人
│   ├── binance/           # Binance API客户端
│   ├── strategy/          # 交易策略
│   └── trading/           # 交易执行
├── pkg/                   # 公共包
│   ├── logger/            # 日志系统
│   └── utils/             # 工具函数
├── data/                  # 数据文件
├── logs/                  # 日志文件
├── main.go               # 程序入口
├── config.json.example   # 配置文件示例
├── Makefile              # 构建脚本
└── README.md             # 项目说明
```

## Telegram指令

- `/start` - 启动机器人
- `/status` - 查看运行状态
- `/positions` - 查看当前仓位
- `/balance` - 查看账户余额
- `/stop` - 停止交易
- `/resume` - 恢复交易
- `/help` - 显示帮助信息

## 开发指南

### 可用命令

```bash
make help           # 显示所有可用命令
make deps           # 安装依赖
make build          # 构建项目
make test           # 运行测试
make test-coverage  # 生成测试覆盖率报告
make fmt            # 格式化代码
make vet            # 静态分析
make lint           # 代码检查
make clean          # 清理构建文件
```

### 开发工具

```bash
# 安装开发工具
make install-tools

# 安装热重载工具
make install-air

# 开发模式运行
make dev
```

## 安全注意事项

⚠️ **重要提醒**：

1. **测试网优先**: 建议先在Binance测试网上运行和测试
2. **API密钥安全**: 不要将API密钥提交到版本控制系统
3. **权限最小化**: API密钥只开启必要的交易权限
4. **资金管理**: 合理设置仓位大小和风险参数
5. **监控告警**: 及时关注Telegram通知和系统日志

## 风险声明

本项目仅供学习和研究使用。加密货币交易存在高风险，可能导致资金损失。使用本软件进行实盘交易的所有风险由用户自行承担。

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 贡献

欢迎提交Issue和Pull Request来改进项目。

## 联系方式

如有问题或建议，请通过以下方式联系：

- GitHub Issues: [项目Issues页面](https://github.com/pengxuan37/vegas-dual-tunnel-trading-bot/issues)
- Email: your-email@example.com