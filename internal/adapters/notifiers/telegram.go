package notifiers

import (
    "context"
    "fmt"
    "strings"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/you/wallet_transaction_notifier/internal/domain"
    "github.com/you/wallet_transaction_notifier/internal/ports"
)

type TelegramNotifier struct {
    bot      *tgbotapi.BotAPI
    subsRepo ports.SubscriptionRepository
}

func NewTelegramNotifier(botToken string, subsRepo ports.SubscriptionRepository) (*TelegramNotifier, error) {
    if botToken == "" {
        return &TelegramNotifier{}, nil
    }

    bot, err := tgbotapi.NewBotAPI(botToken)
    if err != nil {
        return nil, fmt.Errorf("failed to create bot: %w", err)
    }

    return &TelegramNotifier{
        bot:      bot,
        subsRepo: subsRepo,
    }, nil
}

func (t *TelegramNotifier) SendAlert(userID string, event domain.TransactionEvent) error {
    if t.bot == nil {
        return nil
    }

    // Get all users who subscribed to watch this address
    subscribers, err := t.subsRepo.ListSubscribersByAddress(context.Background(), event.Blockchain, event.WalletID)
    if err != nil {
        return fmt.Errorf("failed to get subscribers: %w", err)
    }

    if len(subscribers) == 0 {
        return nil // No subscribers for this address
    }

    // Create a beautiful notification message
    msg := t.createNotificationMessage(event)

    // Send notification to all subscribers
    for _, sub := range subscribers {
        if err := t.sendToUser(sub.ChatID, msg); err != nil {
            // Log error but continue with other users
            fmt.Printf("Failed to send notification to chat %s: %v\n", sub.ChatID, err)
        }
    }

    return nil
}

func (t *TelegramNotifier) createNotificationMessage(event domain.TransactionEvent) string {
    direction := "📥 Incoming"
    if event.Direction == domain.DirectionOutgoing {
        direction = "📤 Outgoing"
    }

    blockchain := "🔷 Ethereum"
    if event.Blockchain == "bitcoin" {
        blockchain = "🟠 Bitcoin"
    }

    timestamp := time.Unix(event.Timestamp, 0).Format("2006-01-02 15:04:05")
    
    return fmt.Sprintf(`🚨 *Transaction Alert*

%s %s
%s

💰 *Amount:* %.6f %s
🔗 *Network:* %s
📍 *Address:* ` + "`%s`" + `
🆔 *Tx Hash:* ` + "`%s`" + `
⏰ *Time:* %s

[View on Etherscan](https://etherscan.io/tx/%s)`,
        direction, event.Direction,
        blockchain,
        event.Amount, event.Currency,
        strings.Title(event.Blockchain),
        event.WalletID,
        event.TxHash,
        timestamp,
        event.TxHash)
}

func (t *TelegramNotifier) sendToUser(chatID, message string) error {
    msg := tgbotapi.NewMessage(0, message)
    msg.ParseMode = tgbotapi.ModeMarkdown
    var chatIDInt int64
    fmt.Sscanf(chatID, "%d", &chatIDInt)
    msg.ChatID = chatIDInt
    msg.DisableWebPagePreview = true

    _, err := t.bot.Send(msg)
    return err
}


