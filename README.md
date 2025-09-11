Wallet Transaction Notifier (N-layer Go Scaffold)

Overview

This project monitors blockchain wallets and sends alerts when new transactions occur. It is structured with an N-layer architecture and clean, replaceable ports/adapters.

Layers

- internal/domain: Core entities and value objects.
- internal/ports: Interfaces (ports) for adapters, event bus, repositories, notifiers.
- internal/services: Application orchestration/services (consumes ports, emits events).
- internal/adapters: External integrations (blockchain, notifiers).
- internal/infra: Infrastructure implementations (HTTP server, event bus, repositories).
- cmd/api: API service entrypoint.

Quick start (local)

1) Prerequisites: Go 1.22+, Docker (optional).
2) Install deps:
   - go mod tidy
3) Run API:
   - go run ./cmd/api
4) Health check:
   - GET http://localhost:8080/health
5) List wallets (demo data):
   - GET http://localhost:8080/users/u-demo/wallets

Environment variables

- APP_PORT (default 8080)
- MONGO_URI (default mongodb://mongo:27017)
- MONGO_DB (default wallet_notifier)
- TELEGRAM_BOT_TOKEN (optional for Telegram integration)
- POLL_INTERVAL_SECONDS (default 30)

Docker

Build and run via docker-compose:

1) docker compose up --build -d
2) Open http://localhost:8080/health

Notes

- Mongo repository is stubbed; replace with real Mongo driver logic.
- Ethereum adapter is a polling stub; integrate with a real provider (Infura/Alchemy).
- Telegram notifier is a stub; integrate with Telegram Bot API.


