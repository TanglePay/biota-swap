package smpc

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

func Sign(pubkey, gid, context, hash, threshold, keyType string, pk *ecdsa.PrivateKey) (string, error) {
	// get sign nonce
	signNonce, err := client.Call("smpc_getSignNonce", account)
	if err != nil {
		return "", fmt.Errorf("smpc_getSignNonce error. %v", err)
	}
	nonceStr, err := getJSONResult(signNonce)
	if err != nil {
		return "", fmt.Errorf("getJSONResult of smpc_getSignNonce error. %v, %v", err, signNonce)
	}
	nonce, _ := strconv.ParseUint(nonceStr, 0, 64)

	// build tx data
	txdata := signData{
		TxType:        "SIGN",
		PubKey:        pubkey,
		InputCode:     "",
		MsgContext:    []string{context},
		MsgHash:       []string{hash},
		Keytype:       keyType,
		GroupID:       gid,
		ThresHold:     threshold,
		Mode:          "0",
		AcceptTimeOut: "1200",
		TimeStamp:     strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	playload, _ := json.Marshal(txdata)
	// sign tx
	rawTX, err := signTX(signer, pk, nonce, playload)
	if err != nil {
		return "", fmt.Errorf("signTx error. %v", err)
	}

	// get rawTx
	reqKeyID, err := client.Call("smpc_sign", rawTX)
	if err != nil {
		return "", fmt.Errorf("call smpc_sign error. %v", err)
	}

	// get keyID
	keyID, err := getJSONResult(reqKeyID)
	if err != nil {
		return "", fmt.Errorf("call smpc_sign getJSONResult error. %v : %v", reqKeyID, err)
	}

	return keyID, nil
}

var ErrNoAccept = errors.New("get sign accept data fail from db")

func GetSignStatus(keyID string) ([]string, error) {
	var statusJSON signStatus
	reqStatus, err := client.Call("smpc_getSignStatus", keyID)
	if err != nil {
		return nil, fmt.Errorf("smpc_getSignStatus rpc error. %v", err)
	}
	statusJSONStr, err := getJSONResult(reqStatus)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(statusJSONStr), &statusJSON); err != nil {
		return nil, fmt.Errorf("unmarshal statusJSONStr fail. %s, %v", statusJSONStr, err)
	}
	switch statusJSON.Status {
	case "Timeout", "Failure":
		return nil, fmt.Errorf("smpc_getSignStatus=%s", statusJSON.Status)
	case "Success":
		return statusJSON.Rsv, nil
	default:
		return nil, nil
	}
}

func GetCurNodeSignInfo() ([]signCurNodeInfo, error) {
	// get approve list of condominium account
	reqListRep, err := client.Call("smpc_getCurNodeSignInfo", account)
	if err != nil {
		return nil, err
	}
	reqListJSON, err := getJSONData(reqListRep)
	if err != nil {
		return nil, fmt.Errorf("smpc_getCurNodeSignInfo getJSONData error. %v", err)
	}

	var keyList []signCurNodeInfo
	if err := json.Unmarshal(reqListJSON, &keyList); err != nil {
		return nil, fmt.Errorf("unmarshal signCurNodeInfo fail. %s, %v", reqListJSON, err)
	}
	return keyList, nil
}

func AcceptSign(keyInfo signCurNodeInfo, agree bool, pk *ecdsa.PrivateKey) error {
	accept := "AGREE"
	if !agree {
		accept = "DISAGREE"
	}
	data := acceptSignData{
		TxType:     "ACCEPTSIGN",
		Key:        keyInfo.Key,
		Accept:     accept,
		MsgHash:    keyInfo.MsgHash,
		MsgContext: keyInfo.MsgContext,
		TimeStamp:  strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	playload, _ := json.Marshal(data)

	// sign tx
	rawTX, err := signTX(signer, pk, 0, playload)
	if err != nil {
		return err
	}
	// send rawTx
	acceptSignRep, err := client.Call("smpc_acceptSign", rawTX)
	if err != nil {
		return err
	}

	// get result
	_, err = getJSONResult(acceptSignRep)
	return err
}
