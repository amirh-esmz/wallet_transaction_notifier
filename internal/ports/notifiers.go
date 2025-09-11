package ports

import "github.com/you/wallet_transaction_notifier/internal/domain"

// Notifier sends alerts to the end user.
type Notifier interface {
    SendAlert(userID string, event domain.TransactionEvent) error
}


