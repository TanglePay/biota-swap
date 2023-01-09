package tokens

import (
	"math/big"
)

type Token interface {
	Symbol() string
	PublicKey() []byte
	KeyType() string
	Address() string
	StartListen(chan *SwapOrder)
	SendSignedTxData(hash string, txData []byte) ([]byte, error)
}

type SourceToken interface {
	Token
	CreateUnWrapTxData(addr string, amount *big.Int, extra []byte) ([]byte, []byte, error)
	GetWrapTxByHash(txHash string) (BaseTransaction, error)
	ValiditeUnWrapTxData(hash, txData []byte) (BaseTransaction, string, error)
}

type DestinationToken interface {
	Token
	CreateWrapTxData(to string, amount *big.Int, txID string) ([]byte, []byte, error)
	GetUnWrapTxByHash(txHash string) (BaseTransaction, error)
	ValiditeWrapTxData(hash, txData []byte) (BaseTransaction, string, error)
}

type BaseTransaction struct {
	Chain  string
	To     string
	Amount *big.Int
}

type WrapExtra struct {
	Chain string `json:"chain"`
	TxID  string `json:"txid"`
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
	TxID   string
	From   string
	To     string
	Amount string
	Error  error
	Type   int // 0 need to reconnect and 1 only need to record
}
