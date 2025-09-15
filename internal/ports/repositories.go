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

type SessionRepository interface {
    UpsertTelegramSession(ctx context.Context, s domain.TelegramSession) error
    GetTelegramSession(ctx context.Context, chatID string) (domain.TelegramSession, error)
}

type SubscriptionRepository interface {
    AddSubscription(ctx context.Context, sub domain.Subscription) error
    RemoveSubscription(ctx context.Context, chatID string, blockchain string, address string) error
    ListSubscriptions(ctx context.Context, chatID string, blockchain string) ([]domain.Subscription, error)
    ListSubscribersByAddress(ctx context.Context, blockchain string, address string) ([]domain.Subscription, error)
}

type NotificationRepository interface {
    Save(ctx context.Context, n domain.Notification) error
    ListByAddress(ctx context.Context, chatID string, blockchain string, address string, limit int) ([]domain.Notification, error)
}


