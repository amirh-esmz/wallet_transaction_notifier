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
    walletsRepo, err := repository.NewMongoWalletRepository(cfg.MongoURI, cfg.DatabaseName)
    if err != nil {
        log.Printf("❌ Failed to create wallet repository: %v", err)
    }
    sessionsRepo, err := repository.NewMongoSessionRepository(cfg.MongoURI, cfg.DatabaseName)
    if err != nil {
        log.Printf("❌ Failed to create session repository: %v", err)
    }
    subsRepo, err := repository.NewMongoSubscriptionRepository(cfg.MongoURI, cfg.DatabaseName)
    if err != nil {
        log.Printf("❌ Failed to create subscription repository: %v", err)
    }
    notifRepo, err := repository.NewMongoNotificationRepository(cfg.MongoURI, cfg.DatabaseName)
    if err != nil {
        log.Printf("❌ Failed to create notification repository: %v", err)
        log.Printf("MongoURI: %s, DatabaseName: %s", cfg.MongoURI, cfg.DatabaseName)
    } else {
        log.Printf("✅ Notification repository created successfully")
    }
    srv := httpserver.NewServer(cfg, eb, walletsRepo)

    // Start services: Ethereum watcher and notifier dispatcher
    eth := blockchain.NewEthereumEventAdapter(eb, cfg.EthWSURL, subsRepo)
    go func() {
        if err := eth.Run(context.Background()); err != nil {
            log.Printf("ethereum adapter error: %v", err)
        }
    }()

    // Start Bitcoin watcher
    // btc := blockchain.NewBitcoinEventAdapter(eb, cfg.BitcoinRPCURL, cfg.BitcoinRPCUser, cfg.BitcoinRPCPass, subsRepo)
    // go func() {
    //     if err := btc.Run(context.Background()); err != nil {
    //         log.Printf("bitcoin adapter error: %v", err)
    //     }
    // }()

    notifier, err := notifiers.NewTelegramNotifier(cfg.TelegramBotToken, subsRepo)
    if err != nil {
        log.Printf("failed to create telegram notifier: %v", err)
    }
    app := services.NewAppService(eb, subsRepo, notifRepo, notifier)
    go app.Run(context.Background())

    // Telegram bot long polling
    bot, err := services.NewTelegramBotService(cfg.TelegramBotToken, sessionsRepo, subsRepo, notifRepo)
    if err != nil {
        log.Printf("failed to create telegram bot: %v", err)
    } else {
        go bot.Run(context.Background())
    }

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


