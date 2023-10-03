package main

import (
	"bwrap/config"
	"bwrap/gl"
	"bwrap/model"
	"bwrap/server"
	"bwrap/tools"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
)

func main() {
	input()

	config.Load(readRand())

	gl.CreateLogFiles()

	model.ConnectToMysql()

	fmt.Printf("Smpc Bridge Error Deal Version %s\n", config.Version)

	if len(os.Args) < 4 {
		fmt.Printf("Args error. %v\n", os.Args)
		return
	}
	srcToken := os.Args[1]
	desToken := os.Args[2]
	txid := os.Args[3]
	if _, exist := config.Tokens[srcToken]; !exist {
		fmt.Printf("Args 1 src token symbol error. %s\n", os.Args[1])
		return
	}
	if _, exist := config.Tokens[desToken]; !exist {
		fmt.Printf("Args 2 dest token symbol error. %s\n", os.Args[2])
		return
	}

	id := server.DealWrapError(srcToken, desToken, txid, "")
	fmt.Println(hex.EncodeToString(id))
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
	fmt.Printf("Enter keystore's password: ")
	fmt.Scanf("%s", &pwd)
	//pwd = "1234567"
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

	if err := ioutil.WriteFile(filename, jsonData, 0666); err != nil {
		log.Fatal(err)
	}
}
