package main

import (
	"bwrap/config"
	"bwrap/daemon"
	"bwrap/gl"
	"bwrap/model"
	"bwrap/server"
	"bwrap/smpc"
	"bwrap/tools"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
)

func main() {
	if (len(os.Args) == 1) || (os.Args[1] == "-d") {
		input()
		if len(os.Args) == 2 {
			os.Args[1] = "daemon"
		}
	} else if (len(os.Args) == 4) && (os.Args[1] == "-key") {
		createKeyStore(os.Args[2], os.Args[3])
		return
	}

	if len(os.Args) == 2 && os.Args[1] == "daemon" {
		daemon.Background("./out.log", true)
	}

	config.Load(readRand())

	gl.CreateLogFiles()

	model.ConnectToMysql()

	smpc.InitSmpc(config.Smpc.NodeUrl, config.Smpc.Account)

	server.Accept()

	server.ListenTokens()

	daemon.WaitForKill()
}

func readRand() (string, [4]uint64) {
	data, err := os.ReadFile("rand.data")
	if err != nil {
		log.Panicf("read rand.data error. %v", err)
	}
	if err := os.WriteFile("rand.data", []byte("start the process successful! You are very great. Best to every one."), 0666); err != nil {
		log.Panicf("write rand.data error. %v", err)
	}
	os.Remove("rand.data")

	//generate seeds
	var seeds [4]uint64
	seeds[0] = tools.GenerateRandomSeed()
	seeds[1] = tools.GenerateRandomSeed()
	seeds[2] = tools.GenerateRandomSeed()
	seeds[3] = tools.GenerateRandomSeed()

	pwd := tools.GetEncryptString(string(data), seeds)
	return pwd, seeds
}

func input() {
	var pwd string
	fmt.Printf("input password: ")
	fmt.Scanf("%s", &pwd)
	//pwd = "secret"
	if err := os.WriteFile("rand.data", []byte(pwd), 0666); err != nil {
		log.Panicf("write rand.data error. %v", err)
	}
}

func createKeyStore(pwd, filename string) {
	ks := keystore.NewKeyStore("./keystores", keystore.StandardScryptN, keystore.StandardScryptP)
	account, err := ks.NewAccount(pwd)
	if err != nil {
		log.Fatal(err)
	}
	jsonData, err := ks.Export(account, pwd, pwd)
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("../keystore/"+filename, jsonData, 0666); err != nil {
		log.Fatal(err)
	}
}
