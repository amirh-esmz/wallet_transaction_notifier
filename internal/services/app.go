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
        log.Printf("Processing notification for chat %s, address %s", s.ChatID, evt.WalletID)
        
        // Save notification
        notification := domain.Notification{
            ChatID:     s.ChatID,
            Blockchain: evt.Blockchain,
            Address:    evt.WalletID,
            TxHash:     evt.TxHash,
            Direction:  evt.Direction,
            Amount:     evt.Amount,
            Currency:   evt.Currency,
            Timestamp:  evt.Timestamp,
        }
        
        log.Printf("Attempting to save notification: %+v", notification)
        if err := a.notifs.Save(context.Background(), notification); err != nil {
            log.Printf("❌ Failed to save notification for chat %s: %v", s.ChatID, err)
        } else {
            log.Printf("✅ Successfully saved notification for chat %s", s.ChatID)
        }
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


