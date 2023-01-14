package config

import (
	"biota_swap/log"
	"io/ioutil"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

var (
	Env        string
	Server     ServerConfig
	Smpc       SmpcConfig
	Db         Database
	Tokens     map[string]Token
	WrapPairs  map[string]string
	KeyWrapper *keystore.Key
)

//Load load config file
func init() {
	type AllConfig struct {
		Env      string
		Server   ServerConfig
		Smpc     SmpcConfig
		Db       Database
		Tokens   []Token
		Pairs    []WrapPair
		KeyStore string
	}
	all := &AllConfig{}
	if _, err := toml.DecodeFile("./config/conf.toml", all); err != nil {
		panic(err)
	}

	Env = all.Env
	Server = all.Server
	Smpc = all.Smpc
	Db = all.Db
	Tokens = make(map[string]Token)
	for _, t := range all.Tokens {
		Tokens[t.Symbol] = t
	}
	WrapPairs = make(map[string]string)
	for _, p := range all.Pairs {
		WrapPairs[p.SrcToken] = p.DestToken
	}
	var keyjson []byte
	keyjson, err := ioutil.ReadFile(all.KeyStore)
	if err != nil {
		log.Panicf("Read keystore file fail. %s : %v\n", all.KeyStore, err)
	}
	keyWrapper, err := keystore.DecryptKey(keyjson, "secret")
	if err != nil {
		log.Panicf("keystore decrypt error : %v\n", err)
	}

	KeyWrapper = keyWrapper
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

type SmpcConfig struct {
	NodeUrl   string
	Gid       string
	ThresHold string
}

type ServerConfig struct {
	Detect      bool
	DetectCount int
	DetectTime  time.Duration
	Accept      bool
	AcceptTime  time.Duration
}
