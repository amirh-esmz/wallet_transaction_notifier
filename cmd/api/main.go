package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/you/wallet_transaction_notifier/internal/config"
    "github.com/you/wallet_transaction_notifier/internal/adapters/blockchain"
    "github.com/you/wallet_transaction_notifier/internal/adapters/notifiers"
    "github.com/you/wallet_transaction_notifier/internal/infra/eventbus"
    "github.com/you/wallet_transaction_notifier/internal/infra/httpserver"
    "github.com/you/wallet_transaction_notifier/internal/infra/repository"
    "github.com/you/wallet_transaction_notifier/internal/services"
)

func main() {
    cfg := config.Load()

    eb := eventbus.NewInMemoryEventBus()
    walletsRepo, _ := repository.NewMongoWalletRepository(cfg.MongoURI, cfg.DatabaseName)
    sessionsRepo, _ := repository.NewMongoSessionRepository(cfg.MongoURI, cfg.DatabaseName)
    subsRepo, _ := repository.NewMongoSubscriptionRepository(cfg.MongoURI, cfg.DatabaseName)
    notifRepo, _ := repository.NewMongoNotificationRepository(cfg.MongoURI, cfg.DatabaseName)
    srv := httpserver.NewServer(cfg, eb, walletsRepo)

    // Start services: Ethereum watcher and notifier dispatcher
    eth := blockchain.NewEthereumEventAdapter(eb, cfg.EthWSURL)
    _ = eth.Subscribe("0x0000000000000000000000000000000000000000")
    go func() {
        if err := eth.Run(context.Background()); err != nil {
            log.Printf("ethereum adapter error: %v", err)
        }
    }()

    notifier := notifiers.NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
    app := services.NewAppService(eb, subsRepo, notifRepo, notifier)
    go app.Run(context.Background())

    // Telegram bot long polling
    bot := services.NewTelegramBotService(cfg.TelegramBotToken, sessionsRepo, subsRepo, notifRepo)
    go bot.Run(context.Background())

    // graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        if err := srv.Start(); err != nil {
            log.Fatalf("server failed to start: %v", err)
        }
    }()

    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := srv.Stop(shutdownCtx); err != nil {
        log.Printf("graceful shutdown error: %v", err)
    }
    _ = os.Stdout.Sync()
}


