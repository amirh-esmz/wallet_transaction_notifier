package notifiers

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/you/wallet_transaction_notifier/internal/domain"
)

type TelegramNotifier struct {
    botToken string
    chatID   string
}

func NewTelegramNotifier(botToken string, chatID string) *TelegramNotifier {
    return &TelegramNotifier{botToken: botToken, chatID: chatID}
}

func (t *TelegramNotifier) SendAlert(userID string, event domain.TransactionEvent) error {
    _ = userID
    if t.botToken == "" || t.chatID == "" {
        return nil
    }
    msg := fmt.Sprintf("[%s] %s tx %s %s", event.Blockchain, event.Direction, event.Amount, event.TxHash)
    payload := map[string]any{
        "chat_id": t.chatID,
        "text":    msg,
    }
    body, _ := json.Marshal(payload)
    url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
    _, _ = http.Post(url, "application/json", bytes.NewReader(body))
    return nil
}


