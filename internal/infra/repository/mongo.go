package repository

import (
    "context"
    "time"
    "errors"

    "github.com/you/wallet_transaction_notifier/internal/domain"
    "github.com/you/wallet_transaction_notifier/internal/ports"
)

// Mongo repositories are left as stubs for MVP poller/notifier wiring without DB.

type MongoWalletRepository struct{}

func NewMongoWalletRepository(uri string, dbName string) (ports.WalletRepository, error) {
    // TODO: connect to Mongo using official driver
    return &MongoWalletRepository{}, nil
}

func (r *MongoWalletRepository) Create(ctx context.Context, wallet domain.Wallet) error {
    _ = ctx
    _ = wallet
    return errors.New("not implemented")
}

func (r *MongoWalletRepository) ListByUser(ctx context.Context, userID string) ([]domain.Wallet, error) {
    _ = ctx
    _ = userID
    return []domain.Wallet{
        {ID: "w-demo", UserID: "u-demo", Blockchain: "ethereum", Address: "0x0000000000000000000000000000000000000000", CreatedAt: time.Now()},
    }, nil
}

// Sessions
type MongoSessionRepository struct{}

func NewMongoSessionRepository(uri string, dbName string) (ports.SessionRepository, error) {
    return &MongoSessionRepository{}, nil
}

func (r *MongoSessionRepository) UpsertTelegramSession(ctx context.Context, s ports.TelegramSession) error {
    _ = ctx; _ = s
    return nil
}

func (r *MongoSessionRepository) GetTelegramSession(ctx context.Context, chatID string) (ports.TelegramSession, error) {
    _ = ctx; _ = chatID
    return ports.TelegramSession{ChatID: chatID, Blockchain: "ethereum"}, nil
}

// Subscriptions
type MongoSubscriptionRepository struct{}

func NewMongoSubscriptionRepository(uri string, dbName string) (ports.SubscriptionRepository, error) {
    return &MongoSubscriptionRepository{}, nil
}

func (r *MongoSubscriptionRepository) AddSubscription(ctx context.Context, sub ports.Subscription) error {
    _ = ctx; _ = sub
    return nil
}

func (r *MongoSubscriptionRepository) RemoveSubscription(ctx context.Context, chatID string, blockchain string, address string) error {
    _ = ctx; _ = chatID; _ = blockchain; _ = address
    return nil
}

func (r *MongoSubscriptionRepository) ListSubscriptions(ctx context.Context, chatID string, blockchain string) ([]ports.Subscription, error) {
    _ = ctx; _ = chatID; _ = blockchain
    return []ports.Subscription{}, nil
}

func (r *MongoSubscriptionRepository) ListSubscribersByAddress(ctx context.Context, blockchain string, address string) ([]ports.Subscription, error) {
    _ = ctx; _ = blockchain; _ = address
    return []ports.Subscription{}, nil
}

// Notifications
type MongoNotificationRepository struct{}

func NewMongoNotificationRepository(uri string, dbName string) (ports.NotificationRepository, error) {
    return &MongoNotificationRepository{}, nil
}

func (r *MongoNotificationRepository) Save(ctx context.Context, n ports.Notification) error {
    _ = ctx; _ = n
    return nil
}

func (r *MongoNotificationRepository) ListByAddress(ctx context.Context, chatID string, blockchain string, address string, limit int) ([]ports.Notification, error) {
    _ = ctx; _ = chatID; _ = blockchain; _ = address; _ = limit
    return []ports.Notification{}, nil
}


