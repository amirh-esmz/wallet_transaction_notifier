package services

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"
    "github.com/you/wallet_transaction_notifier/internal/ports"
)

type TelegramBotService struct {
    botToken string
    sessions ports.SessionRepository
    subs     ports.SubscriptionRepository
    notifs   ports.NotificationRepository
}

func NewTelegramBotService(botToken string, sessions ports.SessionRepository, subs ports.SubscriptionRepository, notifs ports.NotificationRepository) *TelegramBotService {
    return &TelegramBotService{botToken: botToken, sessions: sessions, subs: subs, notifs: notifs}
}

func (t *TelegramBotService) Run(ctx context.Context) error {
    if t.botToken == "" {
        return nil
    }
    offset := 0
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-time.After(2 * time.Second):
            updates, err := t.getUpdates(offset)
            if err != nil { continue }
            for _, u := range updates {
                offset = u.UpdateID + 1
                if u.Message.Chat.ID == 0 || u.Message.Text == "" { continue }
                chatID := fmt.Sprintf("%d", u.Message.Chat.ID)
                t.handleMessage(ctx, chatID, strings.TrimSpace(u.Message.Text))
            }
        }
    }
}

func (t *TelegramBotService) handleMessage(ctx context.Context, chatID string, text string) {
    // Commands: /start, /crypto <ethereum|bitcoin>, /add <address>, /list, /remove <address>, /notifs <address>
    fields := strings.Fields(text)
    if len(fields) == 0 { return }
    switch fields[0] {
    case "/start":
        t.send(chatID, "Welcome! Use /crypto <ethereum|bitcoin> to select network.")
    case "/crypto":
        if len(fields) < 2 { t.send(chatID, "Usage: /crypto <ethereum|bitcoin>"); return }
        _ = t.sessions.UpsertTelegramSession(ctx, ports.TelegramSession{ChatID: chatID, Blockchain: strings.ToLower(fields[1])})
        t.send(chatID, "Set network to "+strings.ToLower(fields[1]))
    case "/add":
        sess, _ := t.sessions.GetTelegramSession(ctx, chatID)
        if sess.Blockchain == "" { t.send(chatID, "Set crypto first: /crypto <ethereum|bitcoin>"); return }
        if len(fields) < 2 { t.send(chatID, "Usage: /add <address>"); return }
        addr := fields[1]
        _ = t.subs.AddSubscription(ctx, ports.Subscription{ChatID: chatID, Blockchain: sess.Blockchain, Address: strings.ToLower(addr)})
        t.send(chatID, "Subscribed to "+addr)
    case "/list":
        sess, _ := t.sessions.GetTelegramSession(ctx, chatID)
        subs, _ := t.subs.ListSubscriptions(ctx, chatID, sess.Blockchain)
        if len(subs) == 0 { t.send(chatID, "No subscriptions."); return }
        var b strings.Builder
        for _, s := range subs { b.WriteString(s.Address+"\n") }
        t.send(chatID, b.String())
    case "/remove":
        sess, _ := t.sessions.GetTelegramSession(ctx, chatID)
        if len(fields) < 2 { t.send(chatID, "Usage: /remove <address>"); return }
        _ = t.subs.RemoveSubscription(ctx, chatID, sess.Blockchain, strings.ToLower(fields[1]))
        t.send(chatID, "Removed.")
    case "/notifs":
        sess, _ := t.sessions.GetTelegramSession(ctx, chatID)
        if len(fields) < 2 { t.send(chatID, "Usage: /notifs <address>"); return }
        items, _ := t.notifs.ListByAddress(ctx, chatID, sess.Blockchain, strings.ToLower(fields[1]), 10)
        if len(items) == 0 { t.send(chatID, "No notifications."); return }
        var b strings.Builder
        for _, n := range items { b.WriteString(n.TxHash+" "+time.Unix(n.Timestamp,0).Format(time.RFC3339)+"\n") }
        t.send(chatID, b.String())
    default:
        t.send(chatID, "Unknown command.")
    }
}

// Minimal types for Telegram updates
type tgUpdate struct { UpdateID int `json:"update_id"`; Message struct { Text string `json:"text"`; Chat struct { ID int64 `json:"id"` } `json:"chat"` } `json:"message"` }

func (t *TelegramBotService) getUpdates(offset int) ([]tgUpdate, error) {
    endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=25&offset=%d", t.botToken, offset)
    resp, err := http.Get(endpoint)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    var out struct{ OK bool `json:"ok"`; Result []tgUpdate `json:"result"` }
    if err := jsonNewDecoder(resp.Body).Decode(&out); err != nil { return nil, err }
    return out.Result, nil
}

func (t *TelegramBotService) send(chatID string, text string) {
    form := url.Values{}
    form.Set("chat_id", chatID)
    form.Set("text", text)
    http.PostForm(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken), form)
}

// tiny wrapper to avoid importing encoding/json in multiple files here
func jsonNewDecoder(r io.Reader) *json.Decoder { return json.NewDecoder(r) }


