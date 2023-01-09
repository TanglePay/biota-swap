package server

import (
	"biota_swap/tokens"

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
