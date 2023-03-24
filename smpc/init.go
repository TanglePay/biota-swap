package smpc

import (
	"github.com/onrik/ethrpc"
)

var client *ethrpc.EthRPC
var account string

type RunMode int

var runMode RunMode

func InitSmpc(url, addr string) {
	client = ethrpc.New(url)
	account = addr
}
