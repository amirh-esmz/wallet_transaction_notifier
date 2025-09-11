package config

import (
    "log"
    "os"
    "strconv"
    "time"
)

type Config struct {
    AppPort          string
    MongoURI         string
    DatabaseName     string
    TelegramBotToken string
    TelegramChatID   string
    PollInterval     time.Duration
    JWTSecret        string
    EtherscanAPIKey  string
    EthWSURL         string
}

func Load() Config {
    cfg := Config{
        AppPort:          getEnv("APP_PORT", "8080"),
        MongoURI:         getEnv("MONGO_URI", "mongodb://mongo:27017"),
        DatabaseName:     getEnv("MONGO_DB", "wallet_notifier"),
        TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
        TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
        PollInterval:     getEnvDurationSeconds("POLL_INTERVAL_SECONDS", 30),
        JWTSecret:        getEnv("JWT_SECRET", "dev-secret"),
        EtherscanAPIKey:  getEnv("ETHERSCAN_API_KEY", ""),
        EthWSURL:         getEnv("ETH_WS_URL", ""),
    }
    log.Printf("config loaded: port=%s db=%s poll=%s", cfg.AppPort, cfg.DatabaseName, cfg.PollInterval)
    return cfg
}

func getEnv(key string, def string) string {
    v := os.Getenv(key)
    if v == "" {
        return def
    }
    return v
}

func getEnvDurationSeconds(key string, def int) time.Duration {
    v := os.Getenv(key)
    if v == "" {
        return time.Duration(def) * time.Second
    }
    n, err := strconv.Atoi(v)
    if err != nil {
        return time.Duration(def) * time.Second
    }
    return time.Duration(n) * time.Second
}


