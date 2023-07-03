package server

import (
	"bwrap/config"
	"bwrap/model"
	"bwrap/tokens"
	"bwrap/tokens/evm"
	"bwrap/tokens/iota"
	"sync"
	"time"

	"github.com/onrik/ethrpc"
)

type DealedOrdersCheck struct {
	dealed map[string]bool
	mu     sync.Mutex
}

func NewDealedOrdersCheck() *DealedOrdersCheck {
	d := &DealedOrdersCheck{
		dealed: make(map[string]bool),
	}
	go d.expired()
	return d
}

func (doc *DealedOrdersCheck) Check(txid string) bool {
	doc.mu.Lock()
	defer doc.mu.Unlock()
	if _, exist := doc.dealed[txid]; exist {
		return false
	}
	doc.dealed[txid] = true
	return true
}

func (doc *DealedOrdersCheck) expired() {
	ticker := time.NewTicker(time.Hour * 24)
	for range ticker.C {
		doc.mu.Lock()
		doc.dealed = make(map[string]bool)
		doc.mu.Unlock()
	}
}

type SentIotaTx struct {
	orders []*model.SwapOrder
	ts     []tokens.Token
	mu     sync.Mutex
}

func (sit *SentIotaTx) push(order *model.SwapOrder, t tokens.Token) {
	sit.mu.Lock()
	defer sit.mu.Unlock()
	sit.orders = append(sit.orders, order)
	sit.ts = append(sit.ts, t)
}

func (sit *SentIotaTx) pop() (*model.SwapOrder, tokens.Token) {
	sit.mu.Lock()
	defer sit.mu.Unlock()
	if len(sit.orders) == 0 {
		return nil, nil
	}
	txid := sit.orders[0]
	sit.orders = sit.orders[1:]
	t := sit.ts[0]
	sit.ts = sit.ts[1:]
	return txid, t
}

var (
	client       *ethrpc.EthRPC
	srcTokens    map[string]tokens.SourceToken
	destTokens   map[string]tokens.DestinationToken
	dealedOrders *DealedOrdersCheck
	sentIotaTxes SentIotaTx

	seeds [4]uint64
	wg    sync.WaitGroup
)

const (
	UnwrapMethod = "unwrap"
)

func init() {
	srcTokens = make(map[string]tokens.SourceToken)
	destTokens = make(map[string]tokens.DestinationToken)
	dealedOrders = NewDealedOrdersCheck()
}

type MsgContext struct {
	SrcToken  string `json:"src_token"`  // the real token in the source chain
	DestToken string `json:"dest_token"` // the wrapped token in the target chain
	Method    string `json:"method"`     // "wrap" or "unwrap"
	TxData    []byte `json:"txdata"`     // txid of the source chain
	Timestamp int64  `json:"timestamp"`  // in seconds
}

func NewSourceChain(conf *config.Token) tokens.SourceToken {
	switch conf.Symbol {
	case "IOTA":
		return iota.NewIotaToken(conf.NodeUrl, conf.PublicKey, "iota")
	case "ATOI":
		return iota.NewIotaToken(conf.NodeUrl, conf.PublicKey, "atoi")
	default:
		token, err := evm.NewEvmToken(conf.NodeUrl, conf.Contract, conf.Symbol, conf.Account, tokens.ScanBlock, conf.ScanMaxHeight)
		if err != nil {
			panic(err)
		}
		return token
	}
}

func NewDestinationChain(conf *config.Token) tokens.DestinationToken {
	if chain, err := evm.NewEvmToken(conf.NodeUrl, conf.Contract, conf.Symbol, conf.Account, tokens.ScanBlock, conf.ScanMaxHeight); err != nil {
		panic(err)
	} else {
		return chain
	}
}
