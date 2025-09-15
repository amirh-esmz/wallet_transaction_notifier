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

// Subscription ties a chat to a blockchain/address.
type Subscription struct {
    ChatID     string `json:"chatId"`
    Blockchain string `json:"blockchain"`
    Address    string `json:"address"`
}

// UserState represents the current state of a user in the bot conversation
type UserState string

const (
    StateIdle           UserState = "idle"
    StateSelectBlockchain UserState = "select_blockchain"
    StateAddAddress     UserState = "add_address"
    StateRemoveAddress  UserState = "remove_address"
    StateViewNotifications UserState = "view_notifications"
)

// TelegramSession represents a user's session state
type TelegramSession struct {
    ChatID     string    `json:"chatId"`
    State      UserState `json:"state"`
    LastAction string    `json:"lastAction,omitempty"`
    CreatedAt  time.Time `json:"createdAt"`
    UpdatedAt  time.Time `json:"updatedAt"`
}

// Notification log for a chat/address.
type Notification struct {
    ChatID     string     `json:"chatId"`
    Blockchain string     `json:"blockchain"`
    Address    string     `json:"address"`
    TxHash     string     `json:"txHash"`
    Direction  Direction  `json:"direction"`
    Amount     float64    `json:"amount"`
    Currency   string     `json:"currency"`
    Timestamp  int64      `json:"timestamp"`
}


