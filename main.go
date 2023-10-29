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
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"golang.org/x/term"
)

func main() {
	args := make(map[string]bool)
	for i := range os.Args {
		args[os.Args[i]] = true
	}
	if args["key"] {
		fmt.Printf("Input the keystore's password(It must contain number, char(Upper and Lower), and special) \n:")
		pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			panic("read pwd error:" + err.Error())
		}
		CreateKeyStore(string(pwd), "key_"+time.Now().Format("20060102150405"))
		return
	}

	if os.Args[len(os.Args)-1] != "daemon" {
		input()
		os.Args = append(os.Args, "daemon")
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

	go server.Start()

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
