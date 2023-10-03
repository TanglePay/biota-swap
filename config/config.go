package config

import (
	"bwrap/log"
	"bwrap/tools"
	"crypto/ecdsa"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

var (
	Version       string
	PendingTime   int64
	Server        ServerConfig
	Smpc          SmpcConfig
	Db            Database
	Tokens        map[string]*Token
	WrapPairs     map[string]string
	seeds         [4]uint64
	password      string
	keyJsons      map[string][]byte
	TxErrorRecord TxErrorRecordConfig
)

//Load load config file
func Load(pwd string, _seeds [4]uint64) {
	password, seeds = pwd, _seeds
	type AllConfig struct {
		Version       string
		PendingTime   int64
		Server        ServerConfig
		Smpc          SmpcConfig
		Db            Database
		Tokens        []Token
		Pairs         []WrapPair
		TxErrorRecord TxErrorRecordConfig
	}
	all := &AllConfig{}
	if _, err := toml.DecodeFile("./config/conf.toml", all); err != nil {
		panic(err)
	}

	Version = all.Version
	PendingTime = all.PendingTime
	Server = all.Server
	Smpc = all.Smpc
	Db = all.Db
	TxErrorRecord = all.TxErrorRecord
	Tokens = make(map[string]*Token)
	var err error
	for _, t := range all.Tokens {
		tt := t
		if len(t.KeyStore) > 0 {
			var keyjson []byte
			keyjson, err := ioutil.ReadFile(t.KeyStore)
			if err != nil {
				log.Panicf("Read keystore file fail. %s : %v\n", t.KeyStore, err)
			}
			tt.KeyStore = string(keyjson)
		}
		Tokens[t.Symbol] = &tt
		if t.MinAmount == nil {
			log.Panicf("%s's MinAmount must to be set.", t.Symbol)
		}
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
	Smpc.KeyStore = string(keyjson)

	checkKeyStore()
}

func checkKeyStore() {
	keyJsons = make(map[string][]byte)
	pwd := tools.GetDecryptString(password, seeds)
	for symbol, t := range Tokens {
		if len(t.KeyStore) == 0 {
			continue
		}
		keyjson := []byte(t.KeyStore)
		keyWrapper, err := keystore.DecryptKey(keyjson, string(pwd))
		if err != nil {
			panic(symbol + " keystore error : " + err.Error())
		}
		t.Account = keyWrapper.Address
		keyJsons[symbol] = keyjson
	}
	keyWrapper, err := keystore.DecryptKey([]byte(Smpc.KeyStore), string(pwd))
	if err != nil {
		panic("Smpc keystore error : " + err.Error())
	}
	Smpc.Account = keyWrapper.Address.Hex()
	keyJsons["smpc"] = []byte(Smpc.KeyStore)
}

type Token struct {
	Symbol        string
	NodeRpc       string
	NodeWss       string
	ScanEventType int
	ScanMaxHeight uint64
	MultiSignType int
	PublicKey     string
	Contract      string
	MinAmount     *big.Int
	KeyStore      string
	Account       common.Address
	GasPriceUpper int64
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
	KeyStore  string
	Account   string
}

type ServerConfig struct {
	DetectCount    int
	DetectTime     time.Duration
	AcceptTime     time.Duration
	AcceptOverTime int64
}

type TxErrorRecordConfig struct {
	NodeRpc       string
	NodeWss       string
	Contract      string
	ScanEventType int
	TimePeriod    time.Duration
}

func GetPrivateKey(symbol string) (common.Address, *ecdsa.PrivateKey, error) {
	pwd := tools.GetDecryptString(password, seeds)
	keyWrapper, err := keystore.DecryptKey(keyJsons[symbol], string(pwd))
	if err != nil {
		return common.Address{}, nil, err
	}
	return keyWrapper.Address, keyWrapper.PrivateKey, nil
}
