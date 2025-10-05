package blockchain

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/btcsuite/btcd/chaincfg"
    "github.com/btcsuite/btcd/chaincfg/chainhash"
    "github.com/btcsuite/btcd/rpcclient"
    "github.com/btcsuite/btcd/txscript"
    "github.com/btcsuite/btcd/wire"

    "github.com/you/wallet_transaction_notifier/internal/domain"
    "github.com/you/wallet_transaction_notifier/internal/ports"
)

// BitcoinEventAdapter listens to new blocks and publishes transaction events for monitored addresses.
// It doesn't store blockchain data, just processes and forwards relevant transactions.
type BitcoinEventAdapter struct {
    rpcURL    string
    rpcUser   string
    rpcPass   string
    client    *rpcclient.Client
    eb        ports.EventBus
    addresses map[string]struct{} // Bitcoin addresses as strings
    subsRepo  ports.SubscriptionRepository
}

func NewBitcoinEventAdapter(eb ports.EventBus, rpcURL, rpcUser, rpcPass string, subsRepo ports.SubscriptionRepository) *BitcoinEventAdapter {
    return &BitcoinEventAdapter{
        rpcURL:    rpcURL,
        rpcUser:   rpcUser,
        rpcPass:   rpcPass,
        eb:        eb,
        addresses: make(map[string]struct{}),
        subsRepo:  subsRepo,
    }
}

func (a *BitcoinEventAdapter) Events() <-chan domain.TransactionEvent {
    ch, _ := a.eb.Subscribe()
    return ch
}

func (a *BitcoinEventAdapter) Run(ctx context.Context) error {
    // Create RPC client configuration
    connCfg := &rpcclient.ConnConfig{
        Host:         a.rpcURL,
        User:         a.rpcUser,
        Pass:         a.rpcPass,
        HTTPPostMode: true,
        DisableTLS:   true,
    }

    // Try to connect with retry logic
    var client *rpcclient.Client
    var err error
    
    connected := false

    for !connected {
        select {
        case <-ctx.Done():
            return nil
        default:
            client, err = rpcclient.New(connCfg, nil)
            if err == nil {
                // Test the connection
                _, err = client.GetBlockCount()
                if err == nil {
                    connected = true
                    break
                }
            }
            fmt.Printf("Failed to connect to Bitcoin node: %v. Retrying in 10 seconds...\n", err)
            time.Sleep(10 * time.Second)
        }
    }
    
    a.client = client
    fmt.Println("Successfully connected to Bitcoin node")

    // Load all Bitcoin addresses from database
    a.loadAddressesFromDB(ctx)

    // Test the connection first
    blockCount, err := client.GetBlockCount()
    if err != nil {
        fmt.Printf("Failed to get block count: %v\n", err)
        return fmt.Errorf("failed to get block count: %w", err)
    }
    fmt.Printf("Current block count: %d\n", blockCount)

    // Get the latest block hash
    latestHash, err := client.GetBestBlockHash()
    if err != nil {
        fmt.Printf("Failed to get latest block hash: %v\n", err)
        return fmt.Errorf("failed to get latest block hash: %w", err)
    }

    fmt.Printf("Latest block hash: %s\n", latestHash.String())
    fmt.Println("Successfully subscribed to Bitcoin blocks, waiting for new blocks...")

    // Poll for new blocks
    lastHash := latestHash
    ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            fmt.Println("Context cancelled, stopping Bitcoin monitoring...")
            return nil
        case <-ticker.C:
            a.checkForNewBlocks(ctx, lastHash)
            // Update lastHash to current best block
            currentHash, err := client.GetBestBlockHash()
            if err == nil {
                lastHash = currentHash
            }
        }
    }
}

func (a *BitcoinEventAdapter) loadAddressesFromDB(ctx context.Context) {
    if a.subsRepo == nil {
        fmt.Println("No subscription repository available, using manual addresses only")
        return
    }
    
    fmt.Println("Loading Bitcoin addresses from database...")
    
    // Query the database for all unique Bitcoin addresses
    addresses, err := a.subsRepo.GetUniqueAddresses(ctx, "bitcoin")
    if err != nil {
        fmt.Printf("Failed to load addresses from database: %v\n", err)
        return
    }
    
    // Clear current addresses and add new ones
    a.addresses = make(map[string]struct{})
    
    for _, addrStr := range addresses {
        a.addresses[addrStr] = struct{}{}
        fmt.Printf("Added address to monitoring: %s\n", addrStr)
    }
    
    fmt.Printf("Loaded %d Bitcoin addresses from database\n", len(a.addresses))
}

// RefreshAddresses reloads addresses from the database
func (a *BitcoinEventAdapter) RefreshAddresses(ctx context.Context) {
    if a.subsRepo == nil {
        fmt.Println("No subscription repository available for refresh")
        return
    }
    
    fmt.Println("Refreshing Bitcoin addresses from database...")
    
    // Query the database for all unique Bitcoin addresses
    addresses, err := a.subsRepo.GetUniqueAddresses(ctx, "bitcoin")
    if err != nil {
        fmt.Printf("Failed to refresh addresses from database: %v\n", err)
        return
    }
    
    // Clear current addresses and add new ones
    a.addresses = make(map[string]struct{})
    
    for _, addrStr := range addresses {
        a.addresses[addrStr] = struct{}{}
    }
    
    fmt.Printf("Refreshed address list: %d addresses\n", len(a.addresses))
}

func (a *BitcoinEventAdapter) checkForNewBlocks(ctx context.Context, lastHash *chainhash.Hash) {
    // Only process if we have addresses to monitor
    if len(a.addresses) == 0 {
        return
    }

    // Get current best block hash
    currentHash, err := a.client.GetBestBlockHash()
    if err != nil {
        fmt.Printf("Failed to get current block hash: %v\n", err)
        return
    }

    // Check if we have a new block
    if currentHash.String() == lastHash.String() {
        return // No new block
    }

    fmt.Printf("New Bitcoin block detected: %s\n", currentHash.String())

    // Get the new block
    block, err := a.client.GetBlock(currentHash)
    if err != nil {
        fmt.Printf("Failed to get block %s: %v\n", currentHash.String(), err)
        return
    }

    fmt.Printf("Processing Bitcoin block with %d transactions\n", len(block.Transactions))

    // Process each transaction in the block
    for _, tx := range block.Transactions {
        a.processTransaction(ctx, tx)
    }
}

func (a *BitcoinEventAdapter) processTransaction(ctx context.Context, tx *wire.MsgTx) {
    msgTx := tx
    
    // Check if any of the monitored addresses are involved in this transaction
    var involvedAddresses []string
    var directions []domain.Direction
    
    // Check outputs (incoming transactions)
    for _, out := range msgTx.TxOut {
        if out.PkScript != nil {
            // Try to extract address from output script
            addr, err := a.extractAddressFromScript(out.PkScript)
            if err == nil && a.match(addr) {
                involvedAddresses = append(involvedAddresses, addr)
                directions = append(directions, domain.DirectionIncoming)
            }
        }
    }
    
    // Check inputs (outgoing transactions)
    for _, in := range msgTx.TxIn {
        // For inputs, we need to look up the previous output to get the address
        // This is a simplified version - in production you'd want to cache UTXOs
        prevOut := in.PreviousOutPoint
        if prevOut.Hash.String() != "0000000000000000000000000000000000000000000000000000000000000000" {
            // This is not a coinbase transaction
            // In a real implementation, you'd look up the previous output
            // For now, we'll skip input analysis to keep it simple
        }
    }
    
    // Process each involved address
    for i, addr := range involvedAddresses {
        direction := directions[i]
        
        // Calculate total amount for this address
        amount := a.calculateAmountForAddress(tx, addr, direction)
        
        // Calculate transaction hash
        txHash := tx.TxHash().String()
        
        // Create and publish event
        evt := domain.TransactionEvent{
            WalletID:   strings.ToLower(addr),
            Blockchain: "bitcoin",
            TxHash:     txHash,
            Direction:  direction,
            Amount:     amount,
            Currency:   "BTC",
            Timestamp:  time.Now().Unix(),
        }

        fmt.Printf("Publishing Bitcoin transaction event: %s %s %.8f BTC\n", 
            direction, addr, amount)
        
        a.eb.Publish(evt)
    }
}

func (a *BitcoinEventAdapter) extractAddressFromScript(pkScript []byte) (string, error) {
    // Extract address from output script
    scriptClass, addresses, _, err := txscript.ExtractPkScriptAddrs(pkScript, &chaincfg.MainNetParams)
    if err != nil {
        return "", err
    }
    
    // Only handle P2PKH and P2SH for now
    if scriptClass == txscript.PubKeyHashTy || scriptClass == txscript.ScriptHashTy {
        if len(addresses) > 0 {
            return addresses[0].EncodeAddress(), nil
        }
    }
    
    return "", fmt.Errorf("unsupported script type or no address found")
}

func (a *BitcoinEventAdapter) calculateAmountForAddress(tx *wire.MsgTx, addr string, direction domain.Direction) float64 {
    var totalAmount int64
    
    if direction == domain.DirectionIncoming {
        // For incoming transactions, sum up all outputs to this address
        for _, out := range tx.TxOut {
            if out.PkScript != nil {
                extractedAddr, err := a.extractAddressFromScript(out.PkScript)
                if err == nil && extractedAddr == addr {
                    totalAmount += out.Value
                }
            }
        }
    } else {
        // For outgoing transactions, we would need to look up previous outputs
        // This is more complex and requires UTXO tracking
        // For now, return 0 for outgoing transactions
        return 0.0
    }
    
    // Convert satoshis to BTC
    return float64(totalAmount) / 100000000.0
}

func (a *BitcoinEventAdapter) match(addr string) bool {
    _, ok := a.addresses[addr]
    return ok
}
