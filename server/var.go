package server

import (
	"bwrap/config"
	"bwrap/model"
	"bwrap/tokens"
	"bwrap/tokens/evm"
	"bwrap/tokens/iotasmr"
	"bwrap/tokens/smr"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

type SentEvmTxQueue struct {
	txHashs []common.Hash
	chains  []tokens.EvmToken
	times   []int64
	sync.RWMutex
}

func NewSentEvmTxQueue() *SentEvmTxQueue {
	return &SentEvmTxQueue{
		txHashs: make([]common.Hash, 0),
		chains:  make([]tokens.EvmToken, 0),
		times:   make([]int64, 0),
	}
}

func (q *SentEvmTxQueue) Push(txHash common.Hash, t tokens.EvmToken, ts int64) {
	q.Lock()
	defer q.Unlock()
	q.txHashs = append(q.txHashs, txHash)
	q.chains = append(q.chains, t)
	q.times = append(q.times, ts)
}

func (q *SentEvmTxQueue) Pop() {
	q.Lock()
	defer q.Unlock()
	if len(q.txHashs) == 0 {
		return
	}
	q.txHashs = q.txHashs[1:]
	q.chains = q.chains[1:]
	q.times = q.times[1:]
}

func (q *SentEvmTxQueue) Top() (common.Hash, tokens.EvmToken, int64) {
	q.RLock()
	defer q.RUnlock()
	if len(q.txHashs) == 0 {
		return common.Hash{}, nil, 0
	}
	return q.txHashs[0], q.chains[0], q.times[0]
}

func (q *SentEvmTxQueue) UpdateTop(txHash common.Hash, t tokens.EvmToken, ts int64) {
	q.Lock()
	defer q.Unlock()
	if len(q.txHashs) == 0 {
		return
	}
	q.txHashs[0] = txHash
	q.chains[0] = t
	q.times[0] = ts
}

var (
	srcTokens    map[string]tokens.SourceToken
	destTokens   map[string]tokens.DestinationToken
	dealedOrders *DealedOrdersCheck
	sentIotaTxes SentIotaTx
	sentEvmTxes  map[string]*SentEvmTxQueue // key : address+chainid
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
		return iotasmr.NewIotaSmrToken(conf.NodeRpc, conf.PublicKey, conf.Symbol, conf.Contract, "iota")
	case "ATOI":
		return iotasmr.NewIotaSmrToken(conf.NodeRpc, conf.PublicKey, conf.Symbol, conf.Contract, "atoi")
	case "SOON":
		return smr.NewShimmerToken(conf.NodeRpc, conf.PublicKey, conf.Symbol, conf.Contract, "smr")
	default:
		token, err := evm.NewEvmToken(conf.NodeRpc, conf.NodeWss, conf.Contract, conf.Symbol, conf.Account, conf.ScanEventType, conf.ScanMaxHeight, conf.GasPriceUpper)
		if err != nil {
			panic(err)
		}
		return token
	}
}

func NewDestinationChain(conf *config.Token) tokens.DestinationToken {
	if chain, err := evm.NewEvmToken(conf.NodeRpc, conf.NodeWss, conf.Contract, conf.Symbol, conf.Account, conf.ScanEventType, conf.ScanMaxHeight, conf.GasPriceUpper); err != nil {
		panic(err)
	} else {
		return chain
	}
}
