# Docker Setup for Wallet Transaction Notifier

## Quick Start

1. **Set up environment variables:**
   ```bash
   # Create .env file
   echo "TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here" > .env
   echo "JWT_SECRET=your_jwt_secret_here" >> .env
   ```

2. **Start all services:**
   ```bash
   docker-compose up -d
   ```

3. **Check logs:**
   ```bash
   # View all logs
   docker-compose logs -f
   
   # View specific service logs
   docker-compose logs -f api
   docker-compose logs -f ethereum
   ```

## Services

### API Service
- **Port:** 8080
- **Description:** Main application with Telegram bot and transaction monitoring
- **Dependencies:** MongoDB, Ethereum node

### MongoDB
- **Port:** 27017
- **Description:** Database for storing user sessions, subscriptions, and notifications
- **Volume:** `mongo_data` (persistent storage)

### Ethereum Node
- **HTTP RPC:** 8545
- **WebSocket RPC:** 8546
- **Network:** Sepolia testnet (snap sync mode)
- **Description:** Local Ethereum node for monitoring transactions
- **Volume:** `ethereum_data` (minimal storage - snap sync)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TELEGRAM_BOT_TOKEN` | Required | Your Telegram bot token |
| `JWT_SECRET` | `dev-secret` | Secret for JWT authentication |
| `ETH_WS_URL` | `ws://ethereum:8546` | Ethereum WebSocket URL |
| `MONGO_URI` | `mongodb://mongo:27017` | MongoDB connection string |
| `MONGO_DB` | `wallet_notifier` | Database name |
| `APP_PORT` | `8080` | API server port |

## Features

- **Lightweight Ethereum Node:** Uses snap sync mode to minimize storage and bandwidth
- **No Blockchain Storage:** Only processes and forwards relevant transactions
- **Multi-user Notifications:** Sends notifications to all users monitoring an address
- **Docker Compose:** Easy setup with all dependencies included

## Stopping Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: This will delete all data)
docker-compose down -v
```

## Troubleshooting

1. **Ethereum node not syncing:** The node uses snap sync mode which is faster but may take time to catch up
2. **Telegram bot not responding:** Check that `TELEGRAM_BOT_TOKEN` is set correctly
3. **Database connection issues:** Ensure MongoDB is running and accessible
