package smpc

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/onrik/ethrpc"
)

var client *ethrpc.EthRPC
var keyWrapper *keystore.Key

type RunMode int

var runMode RunMode

const (
	Debug   RunMode = 0
	Product RunMode = 1
)

func InitSmpc(url string, keyJson []byte, pwd string) {
	client = ethrpc.New(url)
	var err error
	if keyWrapper, err = keystore.DecryptKey(keyJson, pwd); err != nil {
		panic(err)
	}
}
