package services

import (
    "context"
    "log"

    "github.com/you/wallet_transaction_notifier/internal/domain"
    "github.com/you/wallet_transaction_notifier/internal/ports"
)

// AppService wires adapters, eventbus, and notifiers.
type AppService struct {
    eventBus  ports.EventBus
    notifiers []ports.Notifier
    subs      ports.SubscriptionRepository
    notifs    ports.NotificationRepository
}

func NewAppService(eventBus ports.EventBus, subs ports.SubscriptionRepository, notifs ports.NotificationRepository, notifiers ...ports.Notifier) *AppService {
    return &AppService{eventBus: eventBus, subs: subs, notifs: notifs, notifiers: notifiers}
}

func (a *AppService) Run(ctx context.Context) {
    ch, unsubscribe := a.eventBus.Subscribe()
    defer unsubscribe()
    for {
        select {
        case <-ctx.Done():
            return
        case evt := <-ch:
            a.dispatch(evt)
        }
    }
}

func (a *AppService) dispatch(evt domain.TransactionEvent) {
    // Look up subscribers by address on this chain and notify each chat
    subs, err := a.subs.ListSubscribersByAddress(context.Background(), evt.Blockchain, evt.WalletID)
    if err != nil {
        log.Printf("list subs error: %v", err)
        return
    }
    for _, s := range subs {
        // Save notification
        _ = a.notifs.Save(context.Background(), ports.Notification{
            ChatID:     s.ChatID,
            Blockchain: evt.Blockchain,
            Address:    evt.WalletID,
            TxHash:     evt.TxHash,
            Direction:  evt.Direction,
            Amount:     evt.Amount,
            Currency:   evt.Currency,
            Timestamp:  evt.Timestamp,
        })
        for _, n := range a.notifiers {
            if err := n.SendAlert(s.ChatID, evt); err != nil {
                log.Printf("notifier error: %v", err)
            }
        }
    }
}

// APIService defines application use cases exposed to HTTP handlers.
type APIService struct {
    wallets ports.WalletRepository
}

func NewAPIService(wallets ports.WalletRepository) *APIService {
    return &APIService{wallets: wallets}
}

func (s *APIService) ListUserWallets(ctx context.Context, userID string) ([]domain.Wallet, error) {
    return s.wallets.ListByUser(ctx, userID)
}


