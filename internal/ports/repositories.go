package ports

import (
    "context"
    "github.com/you/wallet_transaction_notifier/internal/domain"
)

// WalletRepository persists wallets.
type WalletRepository interface {
    Create(ctx context.Context, wallet domain.Wallet) error
    ListByUser(ctx context.Context, userID string) ([]domain.Wallet, error)
}

// AlertRepository persists alert channels.
type AlertRepository interface {
    ListByUser(ctx context.Context, userID string) ([]domain.Alert, error)
}

// Telegram session reflects user's chat and chosen blockchain context.
type TelegramSession struct {
    ChatID     string
    Blockchain string // e.g. "ethereum", "bitcoin"
}

type SessionRepository interface {
    UpsertTelegramSession(ctx context.Context, s TelegramSession) error
    GetTelegramSession(ctx context.Context, chatID string) (TelegramSession, error)
}

// Subscription ties a chat to a blockchain/address.
type Subscription struct {
    ChatID     string
    Blockchain string
    Address    string
}

type SubscriptionRepository interface {
    AddSubscription(ctx context.Context, sub Subscription) error
    RemoveSubscription(ctx context.Context, chatID string, blockchain string, address string) error
    ListSubscriptions(ctx context.Context, chatID string, blockchain string) ([]Subscription, error)
    ListSubscribersByAddress(ctx context.Context, blockchain string, address string) ([]Subscription, error)
}

// Notification log for a chat/address.
type Notification struct {
    ChatID     string
    Blockchain string
    Address    string
    TxHash     string
    Direction  domain.Direction
    Amount     float64
    Currency   string
    Timestamp  int64
}

type NotificationRepository interface {
    Save(ctx context.Context, n Notification) error
    ListByAddress(ctx context.Context, chatID string, blockchain string, address string, limit int) ([]Notification, error)
}


