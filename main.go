package main

import (
	"bwrap/config"
	"bwrap/daemon"
	"bwrap/gl"
	"bwrap/model"
	"bwrap/server"
	"bwrap/smpc"
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	if (len(os.Args) == 1) || (os.Args[1] == "-d") {
		input()
		if len(os.Args) == 2 {
			os.Args[1] = "daemon"
		}
	}

	if len(os.Args) == 2 && os.Args[1] == "daemon" {
		daemon.Background("./out.log", true)
	}

	pwd := readRand()

	config.Load(pwd)

	gl.CreateLogFiles()

	model.ConnectToMysql()

	smpc.InitSmpc(config.Smpc.NodeUrl, config.Smpc.KeyWrapper)

	server.Accept()

	server.ListenTokens()

	daemon.WaitForKill()
}

func readRand() string {
	data, err := os.ReadFile("rand.data")
	if err != nil {
		log.Panicf("read rand.data error. %v", err)
	}
	if err := os.WriteFile("rand.data", []byte("start the process successful! You are very great. Best to every one."), 0666); err != nil {
		log.Panicf("write rand.data error. %v", err)
	}
	os.Remove("rand.data")
	return string(data)
}

func input() {
	var pwd string
	fmt.Println("input password:")
	fmt.Scanf("%s", &pwd)
	if err := os.WriteFile("rand.data", []byte(pwd), 0666); err != nil {
		log.Panicf("write rand.data error. %v", err)
	}
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
