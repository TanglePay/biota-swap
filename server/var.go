package server

import (
	"biota_swap/config"
	"biota_swap/tokens"
	"biota_swap/tokens/evm"
	"biota_swap/tokens/iota"
	"biota_swap/tokens/smrevm"

	"github.com/onrik/ethrpc"
)

var (
	client     *ethrpc.EthRPC
	srcTokens  map[string]tokens.SourceToken
	destTokens map[string]tokens.DestinationToken
)

type MsgContext struct {
	SrcToken  string `json:"src_token"`
	DestToken string `json:"dest_token"`
	Method    string `json:"method"`
	TxData    []byte `json:"txdata"`
}

func NewSourceChain(conf config.Token) tokens.SourceToken {
	switch conf.Symbol {
	case "IOTA":
		return iota.NewIotaToken(conf.NodeUrl, conf.PublicKey, "iota")
	case "ATOI":
		return iota.NewIotaToken(conf.NodeUrl, conf.PublicKey, "atoi")
	}
	return nil
}

func NewDestinationChain(conf config.Token) tokens.DestinationToken {
	var chain tokens.DestinationToken
	switch conf.Symbol {
	case "SMIOTA":
		chain, _ = smrevm.NewEvmSiota(conf.NodeUrl, conf.Contact, conf.PublicKey)
	case "MATIC":
		chain, _ = evm.NewEvmSiota(conf.NodeUrl, conf.Contact, conf.PublicKey)
	}
	return chain
}
