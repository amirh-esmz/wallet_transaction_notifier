package eventbus

import (
    "sync"
    "github.com/you/wallet_transaction_notifier/internal/domain"
    "github.com/you/wallet_transaction_notifier/internal/ports"
)

type inMemoryEventBus struct {
    mu          sync.RWMutex
    subscribers map[chan domain.TransactionEvent]struct{}
}

func NewInMemoryEventBus() ports.EventBus {
    return &inMemoryEventBus{
        subscribers: make(map[chan domain.TransactionEvent]struct{}),
    }
}

func (b *inMemoryEventBus) Publish(event domain.TransactionEvent) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for ch := range b.subscribers {
        select {
        case ch <- event:
        default:
        }
    }
}

func (b *inMemoryEventBus) Subscribe() (<-chan domain.TransactionEvent, func()) {
    ch := make(chan domain.TransactionEvent, 32)
    b.mu.Lock()
    b.subscribers[ch] = struct{}{}
    b.mu.Unlock()
    unsubscribe := func() {
        b.mu.Lock()
        delete(b.subscribers, ch)
        close(ch)
        b.mu.Unlock()
    }
    return ch, unsubscribe
}


