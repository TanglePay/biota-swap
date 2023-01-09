package smpc

import (
	"biota_swap/tools/crypto"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotaledger/iota.go/v2/ed25519"
)

func Sign(pubkey, gid, context, hash, threshold string, keyType string) (string, error) {
	if runMode == Debug {
		data := make([]byte, 32)
		rand.Read(data)
		keyID := common.Bytes2Hex(data)
		sign(keyID, hash, keyType)
		return keyID, nil
	}
	// get sign nonce
	signNonce, err := client.Call("smpc_getSignNonce", keyWrapper.Address.String())
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
		Mode:          "1",
		AcceptTimeOut: "600",
		TimeStamp:     strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	playload, _ := json.Marshal(txdata)
	// sign tx
	rawTX, err := signTX(signer, keyWrapper.PrivateKey, nonce, playload)
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

func GetSignStatus(keyID string) ([]string, error) {
	if runMode == Debug {
		return hashDB[keyID], nil
	}
	var statusJSON signStatus
	reqStatus, err := client.Call("smpc_getSignStatus", keyID)
	if err != nil {
		return nil, fmt.Errorf("smpc_getSignStatus rpc error. %v", err)
	}
	statusJSONStr, err := getJSONResult(reqStatus)
	if err != nil {
		return nil, fmt.Errorf("smpc_getSignStatus=NotStart, keyID=%s. %v", keyID, err)
	}
	if err := json.Unmarshal([]byte(statusJSONStr), &statusJSON); err != nil {
		return nil, fmt.Errorf("Unmarshal statusJSONStr fail. %s, %v", statusJSONStr, err)
	}
	switch statusJSON.Status {
	case "Timeout", "Failure":
		return nil, fmt.Errorf("smpc_getSignStatus=%s, keyID=%s", statusJSON.Status, keyID)
	case "Success":
		return statusJSON.Rsv, nil
	default:
		return nil, nil
	}
}

func GetCurNodeSignInfo() ([]signCurNodeInfo, error) {
	// get approve list of condominium account
	reqListRep, err := client.Call("smpc_getCurNodeSignInfo", keyWrapper.Address.String())
	if err != nil {
		return nil, err
	}
	reqListJSON, err := getJSONData(reqListRep)
	if err != nil {
		return nil, fmt.Errorf("smpc_getCurNodeSignInfo getJSONData error. %v", err)
	}

	var keyList []signCurNodeInfo
	if err := json.Unmarshal(reqListJSON, &keyList); err != nil {
		return nil, fmt.Errorf("Unmarshal signCurNodeInfo fail. %s, %v", reqListJSON, err)
	}
	return keyList, nil
}

func AcceptSign(keyInfo signCurNodeInfo, agree bool) error {
	accept := "AGREE"
	if !agree {
		accept = "DISAGREE"
	}
	data := acceptSignData{
		TxType:     "",
		Key:        keyInfo.Key,
		Accept:     accept,
		MsgHash:    keyInfo.MsgHash,
		MsgContext: keyInfo.MsgContext,
		TimeStamp:  strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	playload, _ := json.Marshal(data)

	// sign tx
	rawTX, err := signTX(signer, keyWrapper.PrivateKey, 0, playload)
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

// --------------------------Debug--------------------------------
var hashDB map[string][]string = make(map[string][]string)

func sign(keyID, hash string, kt string) {
	h := common.FromHex(hash)
	var signHash []byte
	if kt == "EC256K1" {
		pk := "39e10402beff72d338b4c16b5094f88c94330aa32a17351bf5be05da92671a4d"
		privateKey, _ := crypto.HexToECDSA(pk)
		signHash, _ = crypto.Sign(h, privateKey)
	} else {
		pk, _ := hex.DecodeString(string("7b8b821264e031a3c0ffc1a8eea887521e1b3e3a081af4e777fa609789506fbd715593d2c4dfa9bc5b2718e6a4c704b63cd3b62a81ca92b17ee3487daf3d593a"))
		private := ed25519.PrivateKey(pk)
		signHash = ed25519.Sign(private, h)
	}
	hashDB[keyID] = append(hashDB[keyID], common.Bytes2Hex(signHash))
}
