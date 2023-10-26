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
	"log"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"golang.org/x/term"
)

func main() {
	if os.Args[len(os.Args)-1] != "-d" {
		input()
		os.Args = append(os.Args, "-d")
	}
	daemon.Background("./out.log", true)

	config.Load(readRand())

	gl.CreateLogFiles()

	model.ConnectToMysql()

	if len(config.Smpc.NodeUrl) > 0 {
		smpc.InitSmpc(config.Smpc.NodeUrl, config.Smpc.Account)
		server.Accept()
	}

	fmt.Printf("Smpc Bridge Version %s is starting...\n", config.Version)

	server.ListenTokens()

	if len(config.TxErrorRecord.Contract) > 0 {
		go server.ListenTxErrorRecord()
	}

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
	fmt.Printf("Input password \n:")
	pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		panic("read pwd error:" + err.Error())
	}
	if err := os.WriteFile("rand.data", []byte(pwd), 0666); err != nil {
		log.Panicf("write rand.data error. %v", err)
	}
}

func CreateKeyStore(pwd, filename string) {
	ks := keystore.NewKeyStore("./keystores", keystore.StandardScryptN, keystore.StandardScryptP)
	account, err := ks.NewAccount(pwd)
	if err != nil {
		log.Fatal(err)
	}
	jsonData, err := ks.Export(account, pwd, pwd)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(filename, jsonData, 0666); err != nil {
		log.Fatal(err)
	}
}
