package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/you/wallet_transaction_notifier/internal/domain"
	"github.com/you/wallet_transaction_notifier/internal/ports"
)

type TelegramBotService struct {
	bot      *tgbotapi.BotAPI
	sessions ports.SessionRepository
	subs     ports.SubscriptionRepository
	notifs   ports.NotificationRepository
}

func NewTelegramBotService(botToken string, sessions ports.SessionRepository, subs ports.SubscriptionRepository, notifs ports.NotificationRepository) (*TelegramBotService, error) {
	if botToken == "" {
		return &TelegramBotService{}, nil
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &TelegramBotService{
		bot:      bot,
		sessions: sessions,
		subs:     subs,
		notifs:   notifs,
	}, nil
}

func (t *TelegramBotService) Run(ctx context.Context) error {
	if t.bot == nil {
		return nil
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := t.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			if update.Message != nil {
				t.handleMessage(ctx, update.Message)
			} else if update.CallbackQuery != nil {
				t.handleCallbackQuery(ctx, update.CallbackQuery)
			}
		}
	}
}

func (t *TelegramBotService) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	chatID := fmt.Sprintf("%d", message.Chat.ID)
	text := strings.TrimSpace(message.Text)

	// Get or create user session
	session, err := t.sessions.GetTelegramSession(ctx, chatID)
	if err != nil {
		// Create new session if doesn't exist
		session = domain.TelegramSession{
			ChatID:    chatID,
			State:     domain.StateIdle,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		t.sessions.UpsertTelegramSession(ctx, session)
	}

	// Handle commands
	if strings.HasPrefix(text, "/") {
		t.handleCommand(ctx, chatID, text, &session)
		return
	}

	// Handle state-based responses
	t.handleStateResponse(ctx, chatID, text, &session)
}

func (t *TelegramBotService) handleCommand(ctx context.Context, chatID, text string, session *domain.TelegramSession) {
	command := strings.Fields(text)[0]

	switch command {
	case "/start":
		t.sendWelcomeMessage(chatID)
		session.State = domain.StateIdle
		t.sessions.UpsertTelegramSession(ctx, *session)

	case "/help":
		t.sendHelpMessage(chatID)

	case "/menu":
		t.sendMainMenu(chatID)
		session.State = domain.StateIdle
		t.sessions.UpsertTelegramSession(ctx, *session)

	default:
		t.sendMessage(chatID, "Unknown command. Use /help to see available commands.")
	}
}

func (t *TelegramBotService) handleStateResponse(ctx context.Context, chatID, text string, session *domain.TelegramSession) {
	log.Printf("Handling state response for chat %s, state: %s, text: %s", chatID, session.State, text)
	
	switch session.State {
	case domain.StateAddAddress:
		log.Printf("Processing add address for chat %s with text: %s", chatID, text)
		t.handleAddAddress(ctx, chatID, text, session)
	case domain.StateRemoveAddress:
		t.handleRemoveAddress(ctx, chatID, text, session)
	case domain.StateViewNotifications:
		t.handleViewNotifications(ctx, chatID, text, session)
	default:
		log.Printf("Unknown state %s for chat %s, sending generic message", session.State, chatID)
		t.sendMessage(chatID, "Please use the menu buttons or type /help for available commands.")
	}
}

func (t *TelegramBotService) handleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) {
	chatID := fmt.Sprintf("%d", query.Message.Chat.ID)
	data := query.Data

	// Acknowledge the callback
	callback := tgbotapi.NewCallback(query.ID, "")
	t.bot.Request(callback)

	// Get user session
	session, err := t.sessions.GetTelegramSession(ctx, chatID)
	if err != nil {
		t.sendMessage(chatID, "Session error. Please restart with /start")
		return
	}

	switch {
	case strings.HasPrefix(data, "blockchain_"):
		blockchain := strings.TrimPrefix(data, "blockchain_")
		t.handleBlockchainSelection(ctx, chatID, blockchain, &session)
	case strings.HasPrefix(data, "add_address_"):
		blockchain := strings.TrimPrefix(data, "add_address_")
		t.handleAddAddressForBlockchain(ctx, chatID, blockchain, &session)
	case strings.HasPrefix(data, "remove_address_"):
		blockchain := strings.TrimPrefix(data, "remove_address_")
		t.handleRemoveAddressForBlockchain(ctx, chatID, blockchain, &session)
	case strings.HasPrefix(data, "view_notifications_"):
		blockchain := strings.TrimPrefix(data, "view_notifications_")
		t.handleViewNotificationsForBlockchain(ctx, chatID, blockchain, &session)
	case data == "main_menu":
		t.sendMainMenu(chatID)
		session.State = domain.StateIdle
		t.sessions.UpsertTelegramSession(ctx, session)
	case data == "list_subscriptions":
		t.handleListSubscriptions(ctx, chatID, &session)
	case data == "add_address_menu":
		t.sendBlockchainSelection(chatID)
		session.LastAction = ""
		t.sessions.UpsertTelegramSession(ctx, session)
	case data == "remove_address_menu":
		t.sendBlockchainSelection(chatID)
	case data == "view_notifications_menu":
		t.sendBlockchainSelection(chatID)
	case strings.HasPrefix(data, "list_"):
		blockchain := strings.TrimPrefix(data, "list_")
		t.handleListSubscriptionsForBlockchain(ctx, chatID, blockchain, &session)
	case strings.HasPrefix(data, "remove_"):
		// Handle remove specific address
		parts := strings.Split(data, "_")
		if len(parts) >= 3 {
			blockchain := parts[1]
			index := parts[2]
			t.handleRemoveSpecificAddress(ctx, chatID, blockchain, index, &session)
		}
	case strings.HasPrefix(data, "notifications_"):
		// Handle view notifications for specific address
		parts := strings.Split(data, "_")
		if len(parts) >= 3 {
			blockchain := parts[1]
			index := parts[2]
			t.handleViewSpecificNotifications(ctx, chatID, blockchain, index, &session)
		}
	}
}

func (t *TelegramBotService) sendWelcomeMessage(chatID string) {
	msg := `🚀 *Welcome to Wallet Transaction Notifier!*

I'll help you monitor your cryptocurrency wallet addresses and notify you about incoming and outgoing transactions.

*Features:*
• 📊 Monitor multiple blockchain networks
• 🔔 Real-time transaction notifications  
• 📝 View transaction history
• ⚡ Easy address management

Use the buttons below to get started!`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Main Menu", "main_menu"),
		),
	)

	t.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (t *TelegramBotService) sendHelpMessage(chatID string) {
	msg := `*Available Commands:*

/start - Start the bot and show welcome message
/help - Show this help message
/menu - Show main menu

*How to use:*
1. Select a blockchain network
2. Add wallet addresses to monitor
3. Receive real-time notifications
4. View transaction history

Use the menu buttons for easy navigation!`

	t.sendMessage(chatID, msg)
}

func (t *TelegramBotService) sendMainMenu(chatID string) {
	msg := `🎯 *Main Menu*

Choose what you'd like to do:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Add Address", "add_address_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 List Subscriptions", "list_subscriptions"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Remove Address", "remove_address_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 View Notifications", "view_notifications_menu"),
		),
	)

	t.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (t *TelegramBotService) sendBlockchainSelection(chatID string) {
	msg := `🔗 *Select Blockchain Network*

Choose the blockchain you want to work with:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔷 Ethereum", "blockchain_ethereum"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟠 Bitcoin", "blockchain_bitcoin"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
		),
	)

	t.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (t *TelegramBotService) handleBlockchainSelection(ctx context.Context, chatID, blockchain string, session *domain.TelegramSession) {
	msg := fmt.Sprintf("✅ Selected *%s* network\n\nWhat would you like to do?", strings.Title(blockchain))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Add Address", fmt.Sprintf("add_address_%s", blockchain)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 List Addresses", fmt.Sprintf("list_%s", blockchain)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Remove Address", fmt.Sprintf("remove_address_%s", blockchain)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 View Notifications", fmt.Sprintf("view_notifications_%s", blockchain)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
		),
	)

	t.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (t *TelegramBotService) handleAddAddressForBlockchain(ctx context.Context, chatID, blockchain string, session *domain.TelegramSession) {
	msg := fmt.Sprintf("📝 *Add %s Address*\n\nPlease send me the wallet address you want to monitor:", strings.Title(blockchain))
	t.sendMessage(chatID, msg)
	
	log.Printf("Setting state to StateAddAddress for chat %s, blockchain: %s", chatID, blockchain)
	log.Printf("Session before update: State=%s, LastAction=%s", session.State, session.LastAction)
	session.State = domain.StateAddAddress
	session.LastAction = blockchain
	log.Printf("Session after update: State=%s, LastAction=%s", session.State, session.LastAction)
	err := t.sessions.UpsertTelegramSession(ctx, *session)
	if err != nil {
		log.Printf("Failed to save session state: %v", err)
	} else {
		log.Printf("Successfully saved session state for chat %s", chatID)
	}
}

func (t *TelegramBotService) handleAddAddress(ctx context.Context, chatID, address string, session *domain.TelegramSession) {
	blockchain := session.LastAction
	address = strings.ToLower(strings.TrimSpace(address))

	// If blockchain not chosen or invalid, try to auto-detect or prompt selection
	if blockchain == "" || blockchain == "menu" {
		// Heuristic: Ethereum address (0x-prefixed, 42 chars)
		if len(address) == 42 && strings.HasPrefix(address, "0x") {
			log.Printf("Auto-detected ethereum for address %s", address)
			blockchain = "ethereum"
			session.LastAction = blockchain
			_ = t.sessions.UpsertTelegramSession(ctx, *session)
		} else {
			log.Printf("Blockchain not chosen and could not auto-detect for address %s. Prompting selection.", address)
			t.sendMessage(chatID, "Please choose the blockchain for this address.")
			t.sendBlockchainSelection(chatID)
			session.State = domain.StateSelectBlockchain
			_ = t.sessions.UpsertTelegramSession(ctx, *session)
			return
		}
	}

	log.Printf("Adding address: %s for blockchain: %s, chatID: %s", address, blockchain, chatID)

	// Basic address validation
	if !t.isValidAddress(address, blockchain) {
		log.Printf("Invalid address format: %s for blockchain: %s", address, blockchain)
		t.sendMessage(chatID, "❌ Invalid address format. Please try again with a valid address.")
		return
	}

	// Add subscription
	subscription := domain.Subscription{
		ChatID:     chatID,
		Blockchain: blockchain,
		Address:    address,
	}

	log.Printf("Adding subscription to database: %+v", subscription)
	err := t.subs.AddSubscription(ctx, subscription)
	if err != nil {
		log.Printf("Failed to add subscription: %v", err)
		t.sendMessage(chatID, "❌ Failed to add subscription. Please try again.")
		return
	}

	log.Printf("Successfully added subscription for chat %s", chatID)
	msg := fmt.Sprintf("✅ Successfully added address to monitor!\n\n🔗 *Network:* %s\n📍 *Address:* `%s`",
		strings.Title(blockchain), address)
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Add Another", fmt.Sprintf("add_address_%s", blockchain)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
		),
	)

	t.sendMessageWithKeyboard(chatID, msg, keyboard)
	session.State = domain.StateIdle
	t.sessions.UpsertTelegramSession(ctx, *session)
}

func (t *TelegramBotService) handleListSubscriptions(ctx context.Context, chatID string, session *domain.TelegramSession) {
	// Get all subscriptions for this chat
	ethereumSubs, _ := t.subs.ListSubscriptions(ctx, chatID, "ethereum")
	bitcoinSubs, _ := t.subs.ListSubscriptions(ctx, chatID, "bitcoin")

	if len(ethereumSubs) == 0 && len(bitcoinSubs) == 0 {
		msg := "📋 *Your Subscriptions*\n\nNo addresses are being monitored yet.\n\nUse the menu to add some addresses!"
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Add Address", "add_address_menu"),
			),
		)
		t.sendMessageWithKeyboard(chatID, msg, keyboard)
		return
	}

	var msg strings.Builder
	msg.WriteString("📋 *Your Subscriptions*\n\n")

	if len(ethereumSubs) > 0 {
		msg.WriteString("🔷 *Ethereum:*\n")
		for _, sub := range ethereumSubs {
			msg.WriteString(fmt.Sprintf("• `%s`\n", sub.Address))
		}
		msg.WriteString("\n")
	}

	if len(bitcoinSubs) > 0 {
		msg.WriteString("🟠 *Bitcoin:*\n")
		for _, sub := range bitcoinSubs {
			msg.WriteString(fmt.Sprintf("• `%s`\n", sub.Address))
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Add More", "add_address_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
		),
	)

	t.sendMessageWithKeyboard(chatID, msg.String(), keyboard)
}

func (t *TelegramBotService) handleRemoveAddressForBlockchain(ctx context.Context, chatID, blockchain string, session *domain.TelegramSession) {
	// Get subscriptions for this blockchain
	subs, err := t.subs.ListSubscriptions(ctx, chatID, blockchain)
	if err != nil || len(subs) == 0 {
		msg := fmt.Sprintf("📋 *Remove %s Address*\n\nNo addresses found for %s network.", strings.Title(blockchain), strings.Title(blockchain))
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
			),
		)
		t.sendMessageWithKeyboard(chatID, msg, keyboard)
		return
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("🗑️ *Remove %s Address*\n\nSelect an address to remove:\n\n", strings.Title(blockchain)))

	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	for i, sub := range subs {
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("• %s", sub.Address[:8]+"..."),
				fmt.Sprintf("remove_%s_%d", blockchain, i),
			),
		))
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
	))

	t.sendMessageWithKeyboard(chatID, msg.String(), keyboard)
}

func (t *TelegramBotService) handleViewNotificationsForBlockchain(ctx context.Context, chatID, blockchain string, session *domain.TelegramSession) {
	// Get subscriptions for this blockchain
	subs, err := t.subs.ListSubscriptions(ctx, chatID, blockchain)
	if err != nil || len(subs) == 0 {
		msg := fmt.Sprintf("📊 *%s Notifications*\n\nNo addresses found for %s network.", strings.Title(blockchain), strings.Title(blockchain))
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
			),
		)
		t.sendMessageWithKeyboard(chatID, msg, keyboard)
		return
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("📊 *%s Notifications*\n\nSelect an address to view notifications:\n\n", strings.Title(blockchain)))

	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	for i, sub := range subs {
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("• %s", sub.Address[:8]+"..."),
				fmt.Sprintf("notifications_%s_%d", blockchain, i),
			),
		))
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
	))

	t.sendMessageWithKeyboard(chatID, msg.String(), keyboard)
}

func (t *TelegramBotService) handleRemoveAddress(ctx context.Context, chatID, text string, session *domain.TelegramSession) {
	// This would be called when user types an address to remove
	// For now, we'll handle this through callback queries
	t.sendMessage(chatID, "Please use the menu buttons to remove addresses.")
}

func (t *TelegramBotService) handleViewNotifications(ctx context.Context, chatID, text string, session *domain.TelegramSession) {
	// This would be called when user types an address to view notifications
	// For now, we'll handle this through callback queries
	t.sendMessage(chatID, "Please use the menu buttons to view notifications.")
}

func (t *TelegramBotService) isValidAddress(address, blockchain string) bool {
	// Basic validation - in a real implementation, you'd want more robust validation

	log.Printf("Validating address: %s for blockchain: %s", address, blockchain)
	switch blockchain {
	case "ethereum":
		return len(address) == 42 && strings.HasPrefix(address, "0x")
	case "bitcoin":
		return len(address) >= 26 && len(address) <= 35
	default:
		return false
	}
}

func (t *TelegramBotService) sendMessage(chatID, text string) {
	msg := tgbotapi.NewMessage(0, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	var chatIDInt int64
	fmt.Sscanf(chatID, "%d", &chatIDInt)
	msg.ChatID = chatIDInt
	t.bot.Send(msg)
}

func (t *TelegramBotService) sendMessageWithKeyboard(chatID, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(0, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	var chatIDInt int64
	fmt.Sscanf(chatID, "%d", &chatIDInt)
	msg.ChatID = chatIDInt
	msg.ReplyMarkup = keyboard
	t.bot.Send(msg)
}

func (t *TelegramBotService) handleListSubscriptionsForBlockchain(ctx context.Context, chatID, blockchain string, session *domain.TelegramSession) {
	subs, err := t.subs.ListSubscriptions(ctx, chatID, blockchain)
	if err != nil || len(subs) == 0 {
		msg := fmt.Sprintf("📋 *%s Addresses*\n\nNo addresses found for %s network.", strings.Title(blockchain), strings.Title(blockchain))
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Add Address", fmt.Sprintf("add_address_%s", blockchain)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
			),
		)
		t.sendMessageWithKeyboard(chatID, msg, keyboard)
		return
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("📋 *%s Addresses*\n\n", strings.Title(blockchain)))
	for i, sub := range subs {
		msg.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, sub.Address))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Add More", fmt.Sprintf("add_address_%s", blockchain)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
		),
	)

	t.sendMessageWithKeyboard(chatID, msg.String(), keyboard)
}

func (t *TelegramBotService) handleRemoveSpecificAddress(ctx context.Context, chatID, blockchain, indexStr string, session *domain.TelegramSession) {
	// Get subscriptions for this blockchain
	subs, err := t.subs.ListSubscriptions(ctx, chatID, blockchain)
	if err != nil {
		t.sendMessage(chatID, "❌ Error retrieving subscriptions.")
		return
	}

	// Parse index
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil || index < 0 || index >= len(subs) {
		t.sendMessage(chatID, "❌ Invalid address selection.")
		return
	}

	address := subs[index].Address
	err = t.subs.RemoveSubscription(ctx, chatID, blockchain, address)
	if err != nil {
		t.sendMessage(chatID, "❌ Failed to remove subscription.")
		return
	}

	msg := fmt.Sprintf("✅ Successfully removed address!\n\n🔗 *Network:* %s\n📍 *Address:* `%s`", 
		strings.Title(blockchain), address)
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Remove Another", fmt.Sprintf("remove_address_%s", blockchain)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
		),
	)

	t.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (t *TelegramBotService) handleViewSpecificNotifications(ctx context.Context, chatID, blockchain, indexStr string, session *domain.TelegramSession) {
	// Get subscriptions for this blockchain
	subs, err := t.subs.ListSubscriptions(ctx, chatID, blockchain)
	if err != nil {
		t.sendMessage(chatID, "❌ Error retrieving subscriptions.")
		return
	}

	// Parse index
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil || index < 0 || index >= len(subs) {
		t.sendMessage(chatID, "❌ Invalid address selection.")
		return
	}

	address := subs[index].Address
	notifications, err := t.notifs.ListByAddress(ctx, chatID, blockchain, address, 10)
	if err != nil {
		t.sendMessage(chatID, "❌ Error retrieving notifications.")
		return
	}

	if len(notifications) == 0 {
		msg := fmt.Sprintf("📊 *Notifications for %s*\n\nNo notifications found for this address yet.", address[:8]+"...")
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
			),
		)
		t.sendMessageWithKeyboard(chatID, msg, keyboard)
		return
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("📊 *Notifications for %s*\n\n", address[:8]+"..."))
	
	for i, notif := range notifications {
		direction := "📥"
		if notif.Direction == domain.DirectionOutgoing {
			direction = "📤"
		}
		timestamp := time.Unix(notif.Timestamp, 0).Format("2006-01-02 15:04:05")
		msg.WriteString(fmt.Sprintf("%d. %s %s %s %s\n   `%s`\n   %s\n\n", 
			i+1, direction, fmt.Sprintf("%.6f", notif.Amount), notif.Currency, 
			notif.TxHash[:8]+"...", notif.TxHash, timestamp))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "main_menu"),
		),
	)

	t.sendMessageWithKeyboard(chatID, msg.String(), keyboard)
}
