# Environment Setup

## Required Variables

```bash
# Copy the template
cp env.example .env

# Edit with your values
nano .env
```

**Required:**
- `TELEGRAM_BOT_TOKEN` - Get from @BotFather on Telegram
- `JWT_SECRET` - Random string for security
- `ETH_WS_URL` - Ethereum RPC endpoint

**Optional:**
- `MONGO_URI` - MongoDB connection (default: mongodb://localhost:27017)
- `MONGO_DB` - Database name (default: wallet_notifier)
- `BITCOIN_RPC_URL` - Bitcoin RPC endpoint
- `BITCOIN_RPC_USER` - Bitcoin RPC username
- `BITCOIN_RPC_PASS` - Bitcoin RPC password

## Getting API Keys

**Telegram Bot:**
1. Message @BotFather on Telegram
2. Send `/newbot`
3. Follow instructions and copy the token

**Ethereum RPC:**
- **Alchemy**: Sign up at alchemy.com, create app, get API key
- **Infura**: Sign up at infura.io, create project, get project ID
- **QuickNode**: Sign up at quicknode.com, create endpoint

## Example .env

```bash
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz
JWT_SECRET=your-super-secret-key
ETH_WS_URL=wss://eth-mainnet.g.alchemy.com/v2/your-api-key
MONGO_URI=mongodb://localhost:27017
MONGO_DB=wallet_notifier
```

## Troubleshooting

**Bot not responding:**
- Check your bot token is correct
- Make sure bot is added to the chat

**Ethereum connection failed:**
- Verify your RPC URL is correct
- Check if your provider is working
- Use WebSocket URLs (wss://) not HTTP

**MongoDB connection failed:**
- Make sure MongoDB is running
- Check your connection string
- Verify database permissions

**Check logs:**
```bash
docker-compose logs -f api
```
