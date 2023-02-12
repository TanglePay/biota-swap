package tokens

import (
	"math/big"
)

const (
	Contract int = 0
	Smpc     int = 2
)

type Token interface {
	MultiSignType() int
	Symbol() string
	PublicKey() []byte
	KeyType() string
	Address() string
	StartListen(chan *SwapOrder)
	SendSignedTxData(hash string, txData []byte) ([]byte, error)
	CheckTxData(txid []byte, to string, amount *big.Int) error
}

type SourceToken interface {
	Token
	CreateUnWrapTxData(addr string, amount *big.Int, extra []byte) ([]byte, []byte, error)
	ValiditeUnWrapTxData(hash, txData []byte) (BaseTransaction, error)
	SendUnWrap(txid string, amount *big.Int, to string) ([]byte, error)
}

type DestinationToken interface {
	Token
	CreateWrapTxData(to string, amount *big.Int, txID string) ([]byte, []byte, error)
	ValiditeWrapTxData(hash, txData []byte) (BaseTransaction, error)
	SendWrap(txid string, amount *big.Int, to string) ([]byte, error)
}

type BaseTransaction struct {
	Txid   []byte
	To     string
	Amount *big.Int
}

type WrapExtra struct {
	Symbol string `json:"symbol"`
	TxID   string `json:"txid"`
}

type MsgContext struct {
	Symbol string `json:"symbol"`
	Method string `json:"method"`
	TxData []byte `json:"txdata"`
}

type WrapOrder struct {
	TxID   string
	From   string
	To     string
	Amount string
}

type ChainError struct {
	Code  int
	Error error
}

type SwapOrder struct {
	TxID      string
	SrcToken  string
	DestToken string
	From      string
	To        string
	Amount    *big.Int
	Error     error
	Type      int // 0 need to reconnect and 1 only need to record
}
