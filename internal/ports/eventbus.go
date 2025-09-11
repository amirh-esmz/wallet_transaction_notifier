package ports

import "github.com/you/wallet_transaction_notifier/internal/domain"

// EventBus is an internal pub/sub for TransactionEvent.
type EventBus interface {
    Publish(event domain.TransactionEvent)
    Subscribe() (<-chan domain.TransactionEvent, func()) // returns channel and unsubscribe
}


