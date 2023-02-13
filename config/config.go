package config

import (
	"bwrap/log"
	"io/ioutil"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

var (
	Env       string
	Server    ServerConfig
	Smpc      SmpcConfig
	Db        Database
	Tokens    map[string]Token
	WrapPairs map[string]string
)

//Load load config file
func Load(pwd string) {
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
	var err error
	for _, t := range all.Tokens {
		if len(t.KeyStore) > 0 {
			var keyjson []byte
			keyjson, err := ioutil.ReadFile(t.KeyStore)
			if err != nil {
				log.Panicf("Read keystore file fail. %s : %v\n", t.KeyStore, err)
			}
			if t.KeyWrapper, err = keystore.DecryptKey(keyjson, pwd); err != nil {
				log.Panicf("keystore decrypt error : %v\n", err)
			}
		}
		Tokens[t.Symbol] = t
	}
	WrapPairs = make(map[string]string)
	for _, p := range all.Pairs {
		WrapPairs[p.SrcToken] = p.DestToken
	}

	var keyjson []byte
	keyjson, err = ioutil.ReadFile(all.Smpc.KeyStore)
	if err != nil {
		log.Panicf("Read keystore file fail. %s : %v\n", all.Smpc.KeyStore, err)
	}
	keyWrapper, err := keystore.DecryptKey(keyjson, "secret")
	if err != nil {
		log.Panicf("keystore decrypt error : %v\n", err)
	}
	Smpc.KeyWrapper = keyWrapper
}

type Token struct {
	Symbol        string
	NodeUrl       string
	MultiSignType int
	PublicKey     string
	Contact       string
	KeyStore      string
	KeyWrapper    *keystore.Key
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
	NodeUrl    string
	Gid        string
	ThresHold  string
	KeyStore   string
	KeyWrapper *keystore.Key
}

type ServerConfig struct {
	Detect      bool
	DetectCount int
	DetectTime  time.Duration
	Accept      bool
	AcceptTime  time.Duration
}
