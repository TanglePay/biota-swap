package main

import (
	"biota_swap/config"
	"biota_swap/daemon"
	"biota_swap/gl"
	"biota_swap/model"
	"biota_swap/server"
	"biota_swap/smpc"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	if config.Env != "debug" {
		daemon.Background("./out.log", true)
	}

	gl.CreateLogFiles()

	model.ConnectToMysql()

	smpc.InitSmpc(config.Smpc.NodeUrl, config.KeyWrapper)

	if config.Server.Detect {
		server.Accept()
	}

	if config.Server.Accept {
		server.ListenTokens()
	}

	daemon.WaitForKill()
}

func TestQuery() {
	c, err := ethclient.Dial("https://rpc-mumbai.maticvigil.com")
	if err != nil {
		panic(err)
	}

	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress("0xC5ED170966756B86f8a35131D0dfB1F5148995aF")},
	}

	blockHeight, err := c.BlockNumber(context.Background())
	if err != nil {
		panic(err)
	}
	for {
		query.FromBlock = new(big.Int).SetUint64(blockHeight)
		logs, err := c.FilterLogs(context.Background(), query)
		if err != nil {
			fmt.Println(err)
			time.Sleep(5 * time.Second)
			continue
		}
		for i := range logs {
			blockHeight = logs[i].BlockNumber + 1
			fmt.Println(blockHeight)
		}
		if lastBlockNumber, err := c.BlockNumber(context.Background()); err != nil {
			fmt.Println(err)
		} else {
			blockHeight = lastBlockNumber
		}
		time.Sleep(10 * time.Second)
	}
}
