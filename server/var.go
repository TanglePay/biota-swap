package server

import (
	"bwrap/config"
	"bwrap/tokens"
	"bwrap/tokens/evm"
	"bwrap/tokens/iota"

	"github.com/onrik/ethrpc"
)

var (
	client     *ethrpc.EthRPC
	srcTokens  map[string]tokens.SourceToken
	destTokens map[string]tokens.DestinationToken
)

func init() {
	srcTokens = make(map[string]tokens.SourceToken)
	destTokens = make(map[string]tokens.DestinationToken)
}

type MsgContext struct {
	SrcToken  string `json:"src_token"`  // the real token in the source chain
	DestToken string `json:"dest_token"` // the wrapped token in the target chain
	Method    string `json:"method"`     // "wrap" or "unwrap"
	TxData    []byte `json:"txdata"`     // txid of the source chain
	TimeStamp int64  `json:"timestamp"`  // in seconds
}

func NewSourceChain(conf config.Token) tokens.SourceToken {
	switch conf.Symbol {
	case "IOTA":
		return iota.NewIotaToken(conf.NodeUrl, conf.PublicKey, "iota")
	case "ATOI":
		return iota.NewIotaToken(conf.NodeUrl, conf.PublicKey, "atoi")
	case "MATIC":
		token, err := evm.NewEvmToken(conf.NodeUrl, conf.Contact, conf.PublicKey, conf.KeyWrapper.PrivateKey, 1)
		if err != nil {
			panic(err)
		}
		return token
	}
	return nil
}

func NewDestinationChain(conf config.Token) tokens.DestinationToken {
	var chain tokens.DestinationToken
	var err error
	switch conf.Symbol {
	case "SMIOTA":
		chain, err = evm.NewEvmToken(conf.NodeUrl, conf.Contact, conf.PublicKey, conf.KeyWrapper.PrivateKey, 1)
	case "SMATIC":
		chain, err = evm.NewEvmToken(conf.NodeUrl, conf.Contact, conf.PublicKey, conf.KeyWrapper.PrivateKey, 1)
	}
	if err != nil {
		panic(err)
	}
	return chain
}
