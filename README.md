# Wallet Transaction Notifier

A Go-based service that monitors Ethereum and Bitcoin wallet transactions and sends real-time notifications via Telegram.

## Features

- Real-time Ethereum transaction monitoring
- Bitcoin transaction support
- Telegram bot notifications
- MongoDB for data persistence
- Docker support
- RESTful API

## Quick Start

1. Clone the repository
2. Copy environment template: `cp env.example .env`
3. Edit `.env` with your API keys
4. Run with Docker: `docker-compose up`

## Environment Setup

Create a `.env` file with these variables:

```bash
# Required
TELEGRAM_BOT_TOKEN=your-bot-token
JWT_SECRET=your-secret-key
ETH_WS_URL=wss://eth-mainnet.g.alchemy.com/v2/your-api-key

# Optional
MONGO_URI=mongodb://localhost:27017
MONGO_DB=wallet_notifier
BITCOIN_RPC_URL=localhost:8332
BITCOIN_RPC_USER=bitcoin
BITCOIN_RPC_PASS=bitcoin
```

## Getting API Keys

**Telegram Bot:**
1. Message @BotFather on Telegram
2. Create a new bot
3. Copy the token

**Ethereum RPC:**
- Alchemy: Sign up at alchemy.com, create an app, get your API key
- Infura: Sign up at infura.io, create a project, get your project ID
- QuickNode: Sign up at quicknode.com, create an endpoint

## Usage

1. Start the service: `docker-compose up`
2. Add your bot to a Telegram chat
3. Send `/start` to begin
4. Use `/add <address>` to monitor a wallet
5. Get notifications for incoming/outgoing transactions

## API Endpoints

- `GET /wallets` - List user wallets
- `POST /wallets` - Add a new wallet
- `DELETE /wallets/:id` - Remove a wallet

## Development

```bash
# Run locally
go run cmd/api/main.go

# Build
go build -o api cmd/api/main.go

# Test
go test ./...
```

## Docker

```bash
# Build and run
docker-compose up --build

# Run in background
docker-compose up -d

# View logs
docker-compose logs -f
```

## Project Structure

```
├── cmd/api/           # Application entry point
├── internal/
│   ├── adapters/      # External service adapters
│   ├── config/        # Configuration
│   ├── domain/        # Business models
│   ├── infra/         # Infrastructure (HTTP, DB)
│   ├── ports/         # Interfaces
│   └── services/      # Business logic
├── docker-compose.yml
├── Dockerfile
└── README.md
```

## License

MIT