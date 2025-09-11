package domain

import "time"

type User struct {
    ID          string    `json:"_id"`
    Email       string    `json:"email"`
    PasswordHash string   `json:"passwordHash"`
    Plan        string    `json:"plan"`
    CreatedAt   time.Time `json:"createdAt"`
}

type Wallet struct {
    ID         string    `json:"_id"`
    UserID     string    `json:"userId"`
    Blockchain string    `json:"blockchain"`
    Address    string    `json:"address"`
    CreatedAt  time.Time `json:"createdAt"`
}

type Alert struct {
    ID        string                 `json:"_id"`
    UserID    string                 `json:"userId"`
    Type      string                 `json:"type"`
    Config    map[string]any         `json:"config"`
    CreatedAt time.Time              `json:"createdAt"`
}

type Direction string

const (
    DirectionIncoming Direction = "incoming"
    DirectionOutgoing Direction = "outgoing"
)

type TransactionEvent struct {
    WalletID   string     `json:"walletId"`
    Blockchain string     `json:"blockchain"`
    TxHash     string     `json:"txHash"`
    Direction  Direction  `json:"direction"`
    Amount     float64    `json:"amount"`
    Currency   string     `json:"currency"`
    Timestamp  int64      `json:"timestamp"`
}


