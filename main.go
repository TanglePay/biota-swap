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
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	if config.Env != "debug" {
		daemon.Background("./out.log", true)
	}

	gl.CreateLogFiles()

	model.ConnectToMysql()

	smpc.InitSmpc(config.Smpc.NodeUrl, config.KeyWrapper)

	if config.Server.Accept {
		server.Accept()
	}

	if config.Server.Detect {
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
	query := ethereum.FilterQuery{}

	fromHeight, err := c.BlockNumber(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		time.Sleep(5 * time.Second)
		var toHeight uint64
		if toHeight, err = c.BlockNumber(context.Background()); err != nil {
			fmt.Println(err)
			continue
		} else if toHeight <= fromHeight {
			continue
		}

		query.FromBlock = new(big.Int).SetUint64(fromHeight)
		query.ToBlock = new(big.Int).SetUint64(toHeight)
		logs, err := c.FilterLogs(context.Background(), query)
		if err != nil {
			fmt.Println(err)
			continue
		}
		preHeight := uint64(0)
		for i := range logs {
			if preHeight != logs[i].BlockNumber {
				fmt.Println(logs[i].BlockNumber)
				preHeight = logs[i].BlockNumber
			}
		}
		fromHeight = toHeight + 1
	}
}
