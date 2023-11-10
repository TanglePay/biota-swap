package model

import (
	"bwrap/config"
	"fmt"
	"testing"
)

func TestCheckAccounts(t *testing.T) {
	config.Db = config.Database{
		Host:   "127.0.0.1",
		Port:   "3306",
		DbName: "smpc",
		Usr:    "root",
		Pwd:    "851012",
	}
	ConnectToMysql()

	accounts := []string{"0xfb6e712F4f71D418A298EBe239889A2496f1359b", "0x380dF538Ab2587B11466d07ca5c671d33497d5Ca", "0x3Fdd4B2d69848F74E44765e6AD423198bdBD94fa", "0x5e80cf0C104D2D4f685A15deb65A319e95dd80dD"}
	fmt.Println(CheckAccountsState(accounts))
}
