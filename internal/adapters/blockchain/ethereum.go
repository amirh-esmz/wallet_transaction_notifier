package blockchain

import (
    "context"
    "fmt"
    "math/big"
    "strings"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"

    "github.com/you/wallet_transaction_notifier/internal/domain"
    "github.com/you/wallet_transaction_notifier/internal/ports"
)

// EthereumEventAdapter listens to new blocks and publishes transaction events for monitored addresses.
// It doesn't store blockchain data, just processes and forwards relevant transactions.
type EthereumEventAdapter struct {
    wsURL     string
    client    *ethclient.Client
    eb        ports.EventBus
    addresses map[common.Address]struct{}
    subsRepo  ports.SubscriptionRepository
}

func NewEthereumEventAdapter(eb ports.EventBus, wsURL string, subsRepo ports.SubscriptionRepository) *EthereumEventAdapter {
    return &EthereumEventAdapter{
        wsURL:     wsURL,
        eb:        eb,
        addresses: make(map[common.Address]struct{}),
        subsRepo:  subsRepo,
    }
}

func (a *EthereumEventAdapter) Subscribe(address string) error {
    a.addresses[common.HexToAddress(address)] = struct{}{}
    return nil
}

func (a *EthereumEventAdapter) Unsubscribe(address string) error {
    delete(a.addresses, common.HexToAddress(address))
    return nil
}

func (a *EthereumEventAdapter) Events() <-chan domain.TransactionEvent {
    ch, _ := a.eb.Subscribe()
    return ch
}

func (a *EthereumEventAdapter) Run(ctx context.Context) error {
    // Try to connect with retry logic
    var client *ethclient.Client
    var err error
    
    for {
        select {
        case <-ctx.Done():
            return nil
        default:
            client, err = ethclient.Dial(a.wsURL)
            if err == nil {
                break
            }
            fmt.Printf("Failed to connect to Ethereum node: %v. Retrying in 10 seconds...\n", err)
            time.Sleep(10 * time.Second)
        }
    }
    
    a.client = client
    fmt.Println("Successfully connected to Ethereum node")

    // Load all Ethereum addresses from database
    a.loadAddressesFromDB(ctx)

    headers := make(chan *types.Header, 16)
    sub, err := client.SubscribeNewHead(ctx, headers)
    if err != nil {
        return fmt.Errorf("failed to subscribe to new headers: %w", err)
    }
    
    for {
        select {
        case <-ctx.Done():
            sub.Unsubscribe()
            return nil
        case err := <-sub.Err():
            fmt.Printf("Ethereum subscription error: %v. Attempting to reconnect...\n", err)
            // Try to reconnect
            time.Sleep(5 * time.Second)
            return a.Run(ctx) // Recursive retry
        case header := <-headers:
            a.onNewBlock(ctx, header)
        }
    }
}

func (a *EthereumEventAdapter) loadAddressesFromDB(ctx context.Context) {
    if a.subsRepo == nil {
        fmt.Println("No subscription repository available, using manual addresses only")
        return
    }
    
    fmt.Println("Loading Ethereum addresses from database...")
    
    // In a real implementation, you would:
    // 1. Query the database for all unique Ethereum addresses
    // 2. Add them to the monitoring list
    // 3. Set up a periodic refresh mechanism
    
    // For now, we'll keep the manual subscription approach
    // but this is where you'd implement database-driven address loading
    fmt.Printf("Currently monitoring %d Ethereum addresses\n", len(a.addresses))
}

// RefreshAddresses reloads addresses from the database
func (a *EthereumEventAdapter) RefreshAddresses(ctx context.Context) {
    if a.subsRepo == nil {
        return
    }
    
    // Clear current addresses
    a.addresses = make(map[common.Address]struct{})
    
    // In a real implementation, you would query the database here
    // and add all unique Ethereum addresses to a.addresses
    
    fmt.Printf("Refreshed address list: %d addresses\n", len(a.addresses))
}

func (a *EthereumEventAdapter) onNewBlock(ctx context.Context, header *types.Header) {
    // Only process if we have addresses to monitor
    if len(a.addresses) == 0 {
        return
    }

    // Get block with transactions (lightweight - no full state)
    block, err := a.client.BlockByHash(ctx, header.Hash())
    if err != nil {
        fmt.Printf("Failed to get block %s: %v\n", header.Hash().Hex(), err)
        return
    }

    // Get chain ID once per block
    chainID, err := a.client.ChainID(ctx)
    if err != nil {
        fmt.Printf("Failed to get chain ID: %v\n", err)
        return
    }
    signer := types.LatestSignerForChainID(chainID)

    // Process transactions in the block
    transactions := block.Transactions()
    if len(transactions) == 0 {
        return
    }

    fmt.Printf("Processing block %d with %d transactions\n", block.Number().Uint64(), len(transactions))

    // Process each transaction
    for _, tx := range transactions {
        a.processTransaction(ctx, tx, signer)
    }
}

func (a *EthereumEventAdapter) processTransaction(ctx context.Context, tx *types.Transaction, signer types.Signer) {
    // Get sender address
    fromAddr, err := types.Sender(signer, tx)
    if err != nil {
        return // Skip invalid transactions
    }

    to := tx.To()
    var toAddr common.Address
    if to != nil {
        toAddr = *to
    }

    // Check if either sender or receiver is in our watch list
    isFromMonitored := a.match(fromAddr)
    isToMonitored := to != nil && a.match(toAddr)

    if !isFromMonitored && !isToMonitored {
        return // No monitored addresses involved
    }

    // Determine transaction direction and wallet
    direction := domain.DirectionOutgoing
    wallet := strings.ToLower(fromAddr.Hex())
    
    if isToMonitored {
        direction = domain.DirectionIncoming
        wallet = strings.ToLower(toAddr.Hex())
    }

    // Convert wei to ETH
    amountEth := weiToETH(tx.Value())

    // Create and publish event
    evt := domain.TransactionEvent{
        WalletID:   wallet,
        Blockchain: "ethereum",
        TxHash:     tx.Hash().Hex(),
        Direction:  direction,
        Amount:     amountEth,
        Currency:   "ETH",
        Timestamp:  time.Now().Unix(),
    }

    fmt.Printf("Publishing transaction event: %s %s %.6f ETH\n", 
        direction, wallet, amountEth)
    
    a.eb.Publish(evt)
}

func (a *EthereumEventAdapter) match(addr common.Address) bool {
    _, ok := a.addresses[addr]
    return ok
}

func weiToETH(wei *big.Int) float64 {
    if wei == nil {
        return 0
    }
    rat := new(big.Rat).SetInt(wei)
    eth := new(big.Rat).Quo(rat, big.NewRat(1, 1))
    // divide by 1e18
    eth.Quo(eth, new(big.Rat).SetFloat64(1e18))
    f, _ := eth.Float64()
    return f
}


