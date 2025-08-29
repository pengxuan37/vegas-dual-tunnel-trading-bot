package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/internal/config"
	"github.com/pengxuan37/vegas-dual-tunnel-trading-bot/pkg/logger"
)

// Bot Telegram机器人结构
type Bot struct {
	api    *tgbotapi.BotAPI
	config *config.TelegramConfig
	logger logger.Logger
	chatID int64

	// 指令处理器
	commandHandlers map[string]CommandHandler
	
	// 消息队列
	messageQueue chan Message
	
	// 状态管理
	isRunning bool
}

// CommandHandler 指令处理器接口
type CommandHandler interface {
	Handle(ctx context.Context, bot *Bot, update tgbotapi.Update) error
	Description() string
}

// Message 消息结构
type Message struct {
	ChatID int64
	Text   string
	Type   MessageType
}

// MessageType 消息类型
type MessageType int

const (
	MessageTypeText MessageType = iota
	MessageTypeMarkdown
	MessageTypeHTML
)

// New 创建新的Telegram机器人实例
func New(cfg *config.TelegramConfig, log logger.Logger) (*Bot, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("bot token is required")
	}
	
	if cfg.AdminChatID == 0 {
		return nil, fmt.Errorf("admin chat ID is required")
	}

	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	bot := &Bot{
		api:             api,
		config:          cfg,
		logger:          log,
		chatID:          cfg.AdminChatID,
		commandHandlers: make(map[string]CommandHandler),
		messageQueue:    make(chan Message, 100),
		isRunning:       false,
	}

	// 注册默认指令处理器
	bot.registerDefaultHandlers()

	log.Infof("Telegram bot initialized for admin chat ID: %d", cfg.AdminChatID)
	return bot, nil
}

// Start 启动机器人
func (b *Bot) Start(ctx context.Context) error {
	if b.isRunning {
		return fmt.Errorf("bot is already running")
	}

	b.isRunning = true
	b.logger.Info("Starting Telegram bot...")

	// 获取机器人信息
	me, err := b.api.GetMe()
	if err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}

	b.logger.Infof("Bot started: @%s", me.UserName)

	// 启动消息发送协程
	go b.messageProcessor(ctx)

	// 启动更新处理协程
	go b.updateProcessor(ctx)

	// 发送启动消息
	b.SendMessage("🤖 Vegas Dual Tunnel Trading Bot 已启动")

	return nil
}

// Stop 停止机器人
func (b *Bot) Stop() {
	if !b.isRunning {
		return
	}

	b.isRunning = false
	b.logger.Info("Stopping Telegram bot...")
	
	// 发送停止消息
	b.SendMessage("🛑 Vegas Dual Tunnel Trading Bot 已停止")
	
	// 关闭消息队列
	close(b.messageQueue)
}

// SendMessage 发送文本消息
func (b *Bot) SendMessage(text string) error {
	return b.SendMessageToChat(b.chatID, text)
}

// SendMessageToChat 发送消息到指定聊天
func (b *Bot) SendMessageToChat(chatID int64, text string) error {
	select {
	case b.messageQueue <- Message{
		ChatID: chatID,
		Text:   text,
		Type:   MessageTypeText,
	}:
		return nil
	default:
		return fmt.Errorf("message queue is full")
	}
}

// SendMarkdownMessage 发送Markdown格式消息
func (b *Bot) SendMarkdownMessage(text string) error {
	select {
	case b.messageQueue <- Message{
		ChatID: b.chatID,
		Text:   text,
		Type:   MessageTypeMarkdown,
	}:
		return nil
	default:
		return fmt.Errorf("message queue is full")
	}
}

// RegisterCommandHandler 注册指令处理器
func (b *Bot) RegisterCommandHandler(command string, handler CommandHandler) {
	b.commandHandlers[command] = handler
	b.logger.Debugf("Registered command handler: %s", command)
}

// messageProcessor 消息发送处理器
func (b *Bot) messageProcessor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-b.messageQueue:
			if !ok {
				return
			}
			
			if err := b.sendMessage(msg); err != nil {
				b.logger.Errorf("Failed to send message: %v", err)
			}
		}
	}
}

// sendMessage 实际发送消息
func (b *Bot) sendMessage(msg Message) error {
	msgConfig := tgbotapi.NewMessage(msg.ChatID, msg.Text)
	
	switch msg.Type {
	case MessageTypeMarkdown:
		msgConfig.ParseMode = "Markdown"
	case MessageTypeHTML:
		msgConfig.ParseMode = "HTML"
	}

	_, err := b.api.Send(msgConfig)
	return err
}

// updateProcessor 更新处理器
func (b *Bot) updateProcessor(ctx context.Context) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			if err := b.handleUpdate(ctx, update); err != nil {
				b.logger.Errorf("Failed to handle update: %v", err)
			}
		}
	}
}

// handleUpdate 处理更新
func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	// 只处理来自指定聊天的消息
	if update.Message == nil {
		return nil
	}

	if update.Message.Chat.ID != b.chatID {
		b.logger.Warnf("Received message from unauthorized chat: %d", update.Message.Chat.ID)
		return nil
	}

	// 处理指令
	if update.Message.IsCommand() {
		return b.handleCommand(ctx, update)
	}

	return nil
}

// handleCommand 处理指令
func (b *Bot) handleCommand(ctx context.Context, update tgbotapi.Update) error {
	command := update.Message.Command()
	handler, exists := b.commandHandlers[command]
	
	if !exists {
		return b.SendMessage(fmt.Sprintf("❌ 未知指令: /%s\n\n使用 /help 查看可用指令", command))
	}

	b.logger.Infof("Handling command: /%s from user: %s", command, update.Message.From.UserName)
	return handler.Handle(ctx, b, update)
}

// registerDefaultHandlers 注册默认指令处理器
func (b *Bot) registerDefaultHandlers() {
	b.RegisterCommandHandler("start", &StartHandler{})
	b.RegisterCommandHandler("help", &HelpHandler{})
	b.RegisterCommandHandler("status", &StatusHandler{})
	b.RegisterCommandHandler("stop", &StopHandler{})
	b.RegisterCommandHandler("resume", &ResumeHandler{})
	b.RegisterCommandHandler("positions", &PositionsHandler{})
	b.RegisterCommandHandler("balance", &BalanceHandler{})
}