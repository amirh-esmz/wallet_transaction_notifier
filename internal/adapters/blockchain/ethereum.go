package blockchain

import (
    "context"
    "math/big"
    "strings"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"

    "github.com/you/wallet_transaction_notifier/internal/domain"
    "github.com/you/wallet_transaction_notifier/internal/ports"
)

// EthereumEventAdapter listens to pending/confirmed txs via websocket and publishes events.
type EthereumEventAdapter struct {
    wsURL     string
    client    *ethclient.Client
    eb        ports.EventBus
    addresses map[common.Address]struct{}
}

func NewEthereumEventAdapter(eb ports.EventBus, wsURL string) *EthereumEventAdapter {
    return &EthereumEventAdapter{
        wsURL:     wsURL,
        eb:        eb,
        addresses: make(map[common.Address]struct{}),
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
    client, err := ethclient.Dial(a.wsURL)
    if err != nil {
        return err
    }
    a.client = client

    headers := make(chan *types.Header, 16)
    sub, err := client.SubscribeNewHead(ctx, headers)
    if err != nil {
        return err
    }
    for {
        select {
        case <-ctx.Done():
            sub.Unsubscribe()
            return nil
        case err := <-sub.Err():
            return err
        case header := <-headers:
            a.onNewBlock(ctx, header)
        }
    }
}

func (a *EthereumEventAdapter) onNewBlock(ctx context.Context, header *types.Header) {
    block, err := a.client.BlockByHash(ctx, header.Hash())
    if err != nil {
        return
    }
    chainID, err := a.client.ChainID(ctx)
    if err != nil {
        return
    }
    signer := types.LatestSignerForChainID(chainID)
    for _, tx := range block.Transactions() {
        fromAddr, err := types.Sender(signer, tx)
        if err != nil {
            continue
        }
        to := tx.To()
        var toAddr common.Address
        if to != nil { toAddr = *to }
        // If either participant is in our subscribe list, publish
        if a.match(fromAddr) || (to != nil && a.match(toAddr)) {
            direction := domain.DirectionOutgoing
            wallet := strings.ToLower(fromAddr.Hex())
            if to != nil && a.match(toAddr) {
                direction = domain.DirectionIncoming
                wallet = strings.ToLower(toAddr.Hex())
            }
            amountEth := weiToETH(tx.Value())
            evt := domain.TransactionEvent{
                WalletID:   wallet,
                Blockchain: "ethereum",
                TxHash:     tx.Hash().Hex(),
                Direction:  direction,
                Amount:     amountEth,
                Currency:   "ETH",
                Timestamp:  time.Now().Unix(),
            }
            a.eb.Publish(evt)
        }
    }
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


