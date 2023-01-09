package config

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	Env       string
	Sr        ServeConfig
	Db        Database
	NodeUrl   string
	Tokens    map[string]Token
	WrapPairs map[string]string
)

//Load load config file
func init() {
	type AllConfig struct {
		Env     string
		Server  ServeConfig
		Db      Database
		NodeUrl string
		Tokens  []Token
		Pairs   []WrapPair
	}
	all := &AllConfig{}
	if _, err := toml.DecodeFile("./config/conf.toml", all); err != nil {
		panic(err)
	}

	Env = all.Env
	Sr = all.Server
	Db = all.Db
	NodeUrl = all.NodeUrl
	Tokens = make(map[string]Token)
	for _, t := range all.Tokens {
		Tokens[t.Symbol] = t
	}
	WrapPairs = make(map[string]string)
	for _, p := range all.Pairs {
		WrapPairs[p.SrcToken] = p.DestToken
	}
	fmt.Println(all)
}

type Token struct {
	Symbol    string
	NodeUrl   string
	PublicKey string
	Contact   string
}

type WrapPair struct {
	SrcToken  string
	DestToken string
}

type Database struct {
	Host   string
	Port   string
	DbName string
	Usr    string
	Pwd    string
}

type ServeConfig struct {
	NodeUrl     string
	Gid         string
	ThresHold   string
	DetectCount int
	DetectTime  time.Duration
}
