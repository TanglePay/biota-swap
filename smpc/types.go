package smpc

import (
	//"bwrap/tools/rlp"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

var SmpcToAddr string = "0x00000000000000000000000000000000000000dc"

// set signer and chain id
// CHID smpc wallet service ID
var CHID int64 = 30400 //SMPC_walletService  ID
var chainID *big.Int = big.NewInt(CHID)
var signer = types.NewEIP155Signer(chainID)

type response struct {
	Status string      `json:"Status"`
	Tip    string      `json:"Tip"`
	Error  string      `json:"Error"`
	Data   interface{} `json:"Data"`
}

type dataResult struct {
	Result string `json:"result"`
}

type signData struct {
	TxType        string   `json:"TxType"`
	PubKey        string   `json:"PubKey"`
	InputCode     string   `json:"InputCode"`
	MsgContext    []string `json:"MsgContext"`
	MsgHash       []string `json:"MsgHash"`
	Keytype       string   `json:"Keytype"`
	GroupID       string   `json:"GroupId"`
	ThresHold     string   `json:"ThresHold"`
	Mode          string   `json:"Mode"`
	AcceptTimeOut string   `json:"AcceptTimeOut"` //unit: second
	TimeStamp     string   `json:"TimeStamp"`
}

type acceptSignData struct {
	TxType     string   `json:"TxType"`
	Key        string   `json:"Key"`
	Accept     string   `json:"Accept"`
	MsgHash    []string `json:"MsgHash"`
	MsgContext []string `json:"MsgContext"`
	TimeStamp  string   `json:"TimeStamp"`
}

type signStatus struct {
	Status    string      `json:"Status"`
	Rsv       []string    `json:"Rsv"`
	Tip       string      `json:"Tip"`
	Error     string      `json:"Error"`
	AllReply  interface{} `json:"AllReply"`
	TimeStamp string      `json:"TimeStamp"`
}

type signCurNodeInfo struct {
	Account    string   `json:"Account"`
	GroupID    string   `json:"GroupId"`
	Key        string   `json:"Key"`
	KeyType    string   `json:"KeyType"`
	Mode       string   `json:"Mode"`
	MsgContext []string `json:"MsgContext"`
	MsgHash    []string `json:"MsgHash"`
	Nonce      string   `json:"Nonce"`
	PubKey     string   `json:"PubKey"`
	ThresHold  string   `json:"ThresHold"`
	TimeStamp  string   `json:"TimeStamp"`
}

// getJSONResult parse result from rpc return data
func getJSONResult(successResponse json.RawMessage) (string, error) {
	var data dataResult
	repData, err := getJSONData(successResponse)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(repData, &data); err != nil {
		return "", fmt.Errorf("getJSONResult Unmarshal json fail: %v", err)
	}
	return data.Result, nil
}

func getJSONData(successResponse json.RawMessage) ([]byte, error) {
	var rep response
	if err := json.Unmarshal(successResponse, &rep); err != nil {
		return nil, fmt.Errorf("getJSONData Unmarshal json fail: %v", err)
	}
	if rep.Status != "Success" {
		return nil, errors.New(rep.Error)
	}
	repData, err := json.Marshal(rep.Data)
	if err != nil {
		return nil, fmt.Errorf("getJSONData Marshal json fail: %v", err)
	}
	return repData, nil
}

// signTX build tx with sign
func signTX(signer types.EIP155Signer, privatekey *ecdsa.PrivateKey, nonce uint64, playload []byte) (string, error) {
	toAccDef := accounts.Account{
		Address: common.HexToAddress(SmpcToAddr),
	}
	// build tx
	tx := types.NewTransaction(
		uint64(nonce),     // nonce
		toAccDef.Address,  // to address
		big.NewInt(0),     // value
		100000,            // gasLimit
		big.NewInt(80000), // gasPrice
		playload)          // data
	// sign tx by privatekey
	signature, signatureErr := crypto.Sign(signer.Hash(tx).Bytes(), privatekey)
	if signatureErr != nil {
		return "", fmt.Errorf("signature create error, %v", signatureErr)
	}
	// build tx with sign
	sigTx, signErr := tx.WithSignature(signer, signature)
	if signErr != nil {
		return "", fmt.Errorf("signer with signature error. %v", signErr)
	}

	// get raw TX
	txdata, txerr := rlp.EncodeToBytes(sigTx)
	if txerr != nil {
		return "", fmt.Errorf("EncodeToBytes error. %v", txerr)
	}

	return common.Bytes2Hex(txdata), nil
}
