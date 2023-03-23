package main

import (
	"bwrap/config"
	"bwrap/daemon"
	"bwrap/gl"
	"bwrap/model"
	"bwrap/server"
	"bwrap/smpc"
	"fmt"
	"log"
	"os"
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
