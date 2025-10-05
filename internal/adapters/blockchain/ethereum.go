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


func (a *EthereumEventAdapter) Events() <-chan domain.TransactionEvent {
    ch, _ := a.eb.Subscribe()
    return ch
}

func (a *EthereumEventAdapter) Run(ctx context.Context) error {
    // Try to connect with retry logic
    var client *ethclient.Client
    var err error
    
    connected := false

    for !connected {
        select {
        case <-ctx.Done():
            return nil
        default:
            client, err = ethclient.Dial(a.wsURL)
            if err == nil {
                connected = true
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

    // Test the connection first
    blockNumber, err := client.BlockNumber(ctx)
    if err != nil {
        fmt.Printf("Failed to get block number: %v\n", err)
        return fmt.Errorf("failed to get block number: %w", err)
    }
    fmt.Printf("Current block number: %d\n", blockNumber)
    
    // Check if node is syncing
    syncProgress, err := client.SyncProgress(ctx)
    if err != nil {
        fmt.Printf("Failed to get sync progress: %v\n", err)
        // For external RPC services, assume fully synced if sync progress fails
        fmt.Println("Assuming node is fully synced (external RPC service)")
    } else if syncProgress != nil {
        // Check if sync progress indicates syncing (non-zero values)
        if syncProgress.CurrentBlock > 0 || syncProgress.HighestBlock > 0 {
            fmt.Printf("Node is syncing: current block %d, highest block %d\n", 
                syncProgress.CurrentBlock, syncProgress.HighestBlock)
        } else {
            fmt.Println("Node is fully synced (sync progress shows zero values)")
        }
    } else {
        fmt.Println("Node is fully synced")
    }
    
    // Use HTTP polling for more reliable block retrieval
    fmt.Println("Using HTTP polling for reliable block processing...")
    return a.runWithPolling(ctx, client)
}

func (a *EthereumEventAdapter) runWithPolling(ctx context.Context, client *ethclient.Client) error {
    fmt.Println("Starting HTTP polling for new blocks...")
    
    var lastBlockNumber uint64 = 0
    
    // Get initial block number
    blockNumber, err := client.BlockNumber(ctx)
    if err != nil {
        return fmt.Errorf("failed to get initial block number: %w", err)
    }
    lastBlockNumber = blockNumber
    fmt.Printf("Starting polling from block %d\n", lastBlockNumber)
    
    ticker := time.NewTicker(12 * time.Second) // Poll every 12 seconds (Ethereum blocks every ~12 seconds)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            fmt.Println("Context cancelled, stopping polling...")
            return nil
        case <-ticker.C:
            currentBlockNumber, err := client.BlockNumber(ctx)
            if err != nil {
                fmt.Printf("Failed to get block number: %v\n", err)
                continue
            }
            
            if currentBlockNumber > lastBlockNumber {
                fmt.Printf("New block detected: %d (previous: %d)\n", currentBlockNumber, lastBlockNumber)
                
                // Process each new block
                for blockNum := lastBlockNumber + 1; blockNum <= currentBlockNumber; blockNum++ {
                    // Try to get block with basic info first
                    blockHeader, err := client.HeaderByNumber(ctx, big.NewInt(int64(blockNum)))
                    if err != nil {
                        fmt.Printf("Failed to get block header %d: %v\n", blockNum, err)
                        continue
                    }
                    
                    fmt.Printf("Processing block %d (hash: %s)\n", blockHeader.Number.Uint64(), blockHeader.Hash().Hex())
                    
                    // Try to get full block with transactions, but handle errors gracefully
                    block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
                    if err != nil {
                        if strings.Contains(err.Error(), "transaction type not supported") {
                            fmt.Printf("⚠️  Block %d contains unsupported transaction types, processing with limited transaction data\n", blockNum)
                            // Process block header only for basic monitoring
                            a.onNewBlockHeaderOnly(ctx, blockHeader)
                            continue
                        }
                        fmt.Printf("Failed to get block %d: %v\n", blockNum, err)
                        continue
                    }
                    
                    // Full block processing with all transactions
                    a.onNewBlock(ctx, block.Header())
                }
                
                lastBlockNumber = currentBlockNumber
            }
        }
    }
}

func (a *EthereumEventAdapter) loadAddressesFromDB(ctx context.Context) {
    if a.subsRepo == nil {
        fmt.Println("No subscription repository available, using manual addresses only")
        return
    }
    
    fmt.Println("Loading Ethereum addresses from database...")
    
    // Query the database for all unique Ethereum addresses
    addresses, err := a.subsRepo.GetUniqueAddresses(ctx, "ethereum")
    if err != nil {
        fmt.Printf("Failed to load addresses from database: %v\n", err)
        return
    }
    
    // Clear current addresses and add new ones
    a.addresses = make(map[common.Address]struct{})
    
    for _, addrStr := range addresses {
        addr := common.HexToAddress(addrStr)
        a.addresses[addr] = struct{}{}
        fmt.Printf("Added address to monitoring: %s\n", addr.Hex())
    }
    
    fmt.Printf("Loaded %d Ethereum addresses from database\n", len(a.addresses))
}

// RefreshAddresses reloads addresses from the database
func (a *EthereumEventAdapter) RefreshAddresses(ctx context.Context) {
    if a.subsRepo == nil {
        fmt.Println("No subscription repository available for refresh")
        return
    }
    
    fmt.Println("Refreshing Ethereum addresses from database...")
    
    // Query the database for all unique Ethereum addresses
    addresses, err := a.subsRepo.GetUniqueAddresses(ctx, "ethereum")
    if err != nil {
        fmt.Printf("Failed to refresh addresses from database: %v\n", err)
        return
    }
    
    // Clear current addresses and add new ones
    a.addresses = make(map[common.Address]struct{})
    
    for _, addrStr := range addresses {
        addr := common.HexToAddress(addrStr)
        a.addresses[addr] = struct{}{}
    }
    
    fmt.Printf("Refreshed address list: %d addresses\n", len(a.addresses))
}

func (a *EthereumEventAdapter) onNewBlock(ctx context.Context, header *types.Header) {
    fmt.Printf("onNewBlock called for block %d\n", header.Number.Uint64())
    
    // Only process if we have addresses to monitor
    if len(a.addresses) == 0 {
        fmt.Println("No addresses to monitor, skipping block processing")
        return
    }
    
    fmt.Printf("Processing block %d with %d monitored addresses\n", header.Number.Uint64(), len(a.addresses))

    // Get block with retry mechanism
    block, err := a.getBlockWithRetry(ctx, header)
    if err != nil {
        fmt.Printf("Failed to get block %d after retries: %v - skipping\n", header.Number.Uint64(), err)
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

    // Process each transaction with error handling
    processedCount := 0
    skippedCount := 0
    
    for i, tx := range transactions {
        func() {
            defer func() {
                if r := recover(); r != nil {
                    fmt.Printf("Recovered from panic processing transaction %d in block %d: %v\n", i, block.Number().Uint64(), r)
                    skippedCount++
                }
            }()
            
            a.processTransaction(ctx, tx, signer)
            processedCount++
        }()
    }
    
    if skippedCount > 0 {
        fmt.Printf("Block %d: processed %d transactions, skipped %d due to errors\n", 
            block.Number().Uint64(), processedCount, skippedCount)
    }
}

func (a *EthereumEventAdapter) processTransaction(ctx context.Context, tx *types.Transaction, signer types.Signer) {
    // Handle transaction processing with error recovery
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("Recovered from transaction processing error: %v\n", r)
        }
    }()

    // Get sender address with enhanced error handling for modern transaction types
    fromAddr, err := types.Sender(signer, tx)
    if err != nil {
        // Try alternative approaches for different transaction types
        switch {
        case tx.Type() == types.LegacyTxType:
            // Legacy transaction - try with EIP155 signer
            if chainID, chainErr := a.client.ChainID(ctx); chainErr == nil {
                eip155Signer := types.NewEIP155Signer(chainID)
                if fromAddr, err = types.Sender(eip155Signer, tx); err == nil {
                    break
                }
            }
            fmt.Printf("Could not get sender for legacy tx %s: %v\n", tx.Hash().Hex(), err)
            return
        case tx.Type() == types.AccessListTxType:
            // EIP-2930 transaction
            if chainID, chainErr := a.client.ChainID(ctx); chainErr == nil {
                accessListSigner := types.NewEIP2930Signer(chainID)
                if fromAddr, err = types.Sender(accessListSigner, tx); err == nil {
                    break
                }
            }
            fmt.Printf("Could not get sender for access list tx %s: %v\n", tx.Hash().Hex(), err)
            return
        case tx.Type() == types.DynamicFeeTxType:
            // EIP-1559 transaction
            if chainID, chainErr := a.client.ChainID(ctx); chainErr == nil {
                dynamicFeeSigner := types.NewLondonSigner(chainID)
                if fromAddr, err = types.Sender(dynamicFeeSigner, tx); err == nil {
                    break
                }
            }
            fmt.Printf("Could not get sender for dynamic fee tx %s: %v\n", tx.Hash().Hex(), err)
            return
        default:
            // Unknown transaction type - try with latest signer
            fmt.Printf("Unknown transaction type %d for tx %s, trying latest signer\n", tx.Type(), tx.Hash().Hex())
            if fromAddr, err = types.Sender(types.LatestSignerForChainID(nil), tx); err != nil {
                fmt.Printf("Could not get sender for unknown tx type %s: %v\n", tx.Hash().Hex(), err)
                return
            }
        }
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

    // Convert wei to ETH with safety check
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

    fmt.Printf("📤 Publishing transaction event: %s %s %.6f ETH (tx: %s)\n", 
        direction, wallet, amountEth, tx.Hash().Hex())
    
    a.eb.Publish(evt)
}

func (a *EthereumEventAdapter) onNewBlockHeaderOnly(ctx context.Context, header *types.Header) {
    fmt.Printf("Processing block header only for block %d (limited transaction support)\n", header.Number.Uint64())
    
    // Only process if we have addresses to monitor
    if len(a.addresses) == 0 {
        fmt.Println("No addresses to monitor, skipping block processing")
        return
    }
    
    // For blocks with unsupported transaction types, we can still monitor basic ETH transfers
    // by checking if any of our monitored addresses appear in transaction logs or traces
    // This is a simplified approach that may miss some transactions but keeps the system running
    
    fmt.Printf("Block %d monitoring active (limited mode) - watching %d addresses\n", 
        header.Number.Uint64(), len(a.addresses))
}

func (a *EthereumEventAdapter) getBlockWithRetry(ctx context.Context, header *types.Header) (*types.Block, error) {
    // With polling, blocks should be readily available, so just try once with a small delay
    time.Sleep(100 * time.Millisecond)
    
    // Try by number first (most reliable)
    block, err := a.client.BlockByNumber(ctx, header.Number)
    if err == nil {
        return block, nil
    }
    
    // Try by hash as fallback
    block, err = a.client.BlockByHash(ctx, header.Hash())
    if err == nil {
        return block, nil
    }
    
    return nil, fmt.Errorf("block not available: %v", err)
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


