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
    JWTSecret        string
    EthWSURL         string
    BitcoinRPCURL    string
    BitcoinRPCUser   string
    BitcoinRPCPass   string
}

func Load() Config {
    cfg := Config{
        AppPort:          getEnv("APP_PORT", "8080"),
        MongoURI:         getEnv("MONGO_URI", "mongodb://localhost:27017"),
        DatabaseName:     getEnv("MONGO_DB", "wallet_notifier"),
        TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
        JWTSecret:        getEnv("JWT_SECRET", ""),
        EthWSURL:         getEnv("ETH_WS_URL", "wss://eth-mainnet.g.alchemy.com/v2/demo"),
        BitcoinRPCURL:    getEnv("BITCOIN_RPC_URL", "localhost:8332"),
        BitcoinRPCUser:   getEnv("BITCOIN_RPC_USER", "bitcoin"),
        BitcoinRPCPass:   getEnv("BITCOIN_RPC_PASS", "bitcoin"),
    }
    log.Printf("config loaded: port=%s db=%s", cfg.AppPort, cfg.DatabaseName)
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


