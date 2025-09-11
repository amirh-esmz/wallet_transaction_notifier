package ports

import "github.com/you/wallet_transaction_notifier/internal/domain"

// BlockchainAdapter defines subscriptions for wallet events and emits standardized TransactionEvent.
type BlockchainAdapter interface {
    Subscribe(address string) error
    Unsubscribe(address string) error
    Events() <-chan domain.TransactionEvent
}


