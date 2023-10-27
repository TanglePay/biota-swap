package tokens

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	EvmMultiSign int = 0
	SmpcSign     int = 2
	CenterSign   int = 3
)

const (
	ListenEvent = 0
	ScanBlock   = 1
)

type Token interface {
	MultiSignType() int
	ChainID() string
	Symbol() string
	KeyType() string
	Address() string
	CheckSentTx(txid []byte) (bool, error)
	CheckUserTx(txid []byte, toCoin string, d int) (string, string, *big.Int, error)
	CheckTxFailed(failedTx, txid []byte, to string, amount *big.Int, d int) error // if return nil, means Tx is failed
}

type SourceToken interface {
	Token
	StartWrapListen(chan *SwapOrder)
	PublicKey() []byte
	SendSignedTxData(hash string, txData []byte) ([]byte, error)
	CreateUnWrapTxData(addr string, amount *big.Int, extra []byte) ([]byte, []byte, error)
	ValiditeUnWrapTxData(hash, txData []byte) (BaseTransaction, error)
	SendUnWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error)
}

type DestinationToken interface {
	Token
	StartUnWrapListen(chan *SwapOrder)
	CheckUnWrapTx(txid []byte, to, symbol string, amount *big.Int) error
	SendWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error)
}

type EvmToken interface {
	Symbol() string
	CheckPendingAndSpeedUp(txHash common.Hash, prv *ecdsa.PrivateKey) (common.Hash, error)
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
	FromToken string
	ToToken   string
	From      string
	To        string
	Amount    *big.Int
	Org       string // tag the platform, "IotaBee", "TangleSwap"
	Error     error
	Type      int // 0 need to reconnect and 1 only need to record
}

type TxErrorRecord struct {
	D        int // 1 is wrap, -1 is unwrap
	Txid     []byte
	FromCoin string
	ToCoin   string
	Error    error
	Type     int // 0 need to reconnect and 1 only need to record
}
