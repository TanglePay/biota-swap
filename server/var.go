package server

import (
	"bwrap/config"
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

var (
	client       *ethrpc.EthRPC
	srcTokens    map[string]tokens.SourceToken
	destTokens   map[string]tokens.DestinationToken
	dealedOrders *DealedOrdersCheck
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

func NewSourceChain(conf config.Token) tokens.SourceToken {
	switch conf.Symbol {
	case "IOTA":
		return iota.NewIotaToken(conf.NodeUrl, conf.PublicKey, "iota")
	case "ATOI":
		return iota.NewIotaToken(conf.NodeUrl, conf.PublicKey, "atoi")
	case "MATIC":
		token, err := evm.NewEvmToken(conf.NodeUrl, conf.Contract, conf.Symbol, conf.KeyWrapper.PrivateKey, tokens.ScanBlock)
		if err != nil {
			panic(err)
		}
		return token
	}
	return nil
}

func NewDestinationChain(conf config.Token) tokens.DestinationToken {
	if chain, err := evm.NewEvmToken(conf.NodeUrl, conf.Contract, conf.Symbol, conf.KeyWrapper.PrivateKey, tokens.ScanBlock); err != nil {
		panic(err)
	} else {
		return chain
	}
}
