package iota

import (
	"bwrap/gl"
	"bwrap/tokens"
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotaledger/hive.go/serializer"
	iotago "github.com/iotaledger/iota.go/v2"
)

type IotaToken struct {
	rpc        string
	wss        string
	nodeAPI    *iotago.NodeHTTPAPIClient
	publicKey  []byte
	hrp        iotago.NetworkPrefix
	walletAddr iotago.Ed25519Address
}

// NewIotaToken
// url don't contain the prefix of "https://"
func NewIotaToken(_rpc, _wss, publicKey, _hrp string) *IotaToken {
	pubKey, err := hex.DecodeString(publicKey)
	if err != nil {
		panic(err)
	}
	return &IotaToken{
		rpc:        _rpc,
		wss:        _wss,
		nodeAPI:    iotago.NewNodeHTTPAPIClient(_rpc),
		publicKey:  pubKey,
		hrp:        iotago.NetworkPrefix(_hrp),
		walletAddr: iotago.AddressFromEd25519PubKey(pubKey),
	}
}

//
func (ei *IotaToken) MultiSignType() int {
	return tokens.SmpcSign
}

func (it *IotaToken) Symbol() string {
	return strings.ToUpper(string(it.hrp))
}

func (it *IotaToken) PublicKey() []byte {
	return it.publicKey
}

func (it *IotaToken) KeyType() string {
	return "ED25519"
}

func (it *IotaToken) Address() string {
	return it.walletAddr.Bech32(it.hrp)
}

func (it *IotaToken) CheckSentTx(txid []byte) (bool, error) {
	var msgid iotago.MessageID
	copy(msgid[:], txid)
	res, err := it.nodeAPI.MessageMetadataByMessageID(context.Background(), msgid)
	if err != nil {
		return true, err
	}
	if !res.Solid {
		return true, fmt.Errorf("Txid has not solid. %s", hex.EncodeToString(txid))
	}
	if res.ConflictReason != 0 {
		return false, fmt.Errorf("%d : %s : %s", res.ConflictReason, *res.LedgerInclusionState, hex.EncodeToString(txid))
	}
	return true, nil
}

func (it *IotaToken) CheckUserTx(txid []byte, toCoin string, d int) (string, string, *big.Int, error) {
	if d != 1 {
		return "", "", nil, fmt.Errorf("iota network d error. %d", d)
	}
	var msgID iotago.MessageID
	if len(txid) != iotago.MessageIDLength {
		return "", "", nil, fmt.Errorf("txid error. %s", hex.EncodeToString(txid))
	}
	copy(msgID[:], txid)

	meta, err := it.nodeAPI.MessageMetadataByMessageID(context.Background(), msgID)
	if err != nil {
		return "", "", nil, fmt.Errorf("MessageMetadataByMessageID error. %s, %v", hex.EncodeToString(txid), err)
	}
	if meta.ConflictReason != 0 {
		return "", "", nil, fmt.Errorf("ConflictReason is not confirm. %s : %d", hex.EncodeToString(txid), meta.ConflictReason)
	}

	message, err := it.nodeAPI.MessageByMessageID(context.Background(), msgID)
	if err != nil {
		return "", "", nil, fmt.Errorf("MessageByMessageID error. %s, %v", hex.EncodeToString(txid), err)
	}

	//Unmarshal the payload of message
	data, err := message.Payload.MarshalJSON()
	if err != nil {
		return "", "", nil, fmt.Errorf("MarshalJSON for(data) error. %v, %s", err, hex.EncodeToString(txid))
	}
	payload := Payload{}
	err = json.Unmarshal(data, &payload)
	if err != nil {
		return "", "", nil, fmt.Errorf("Unmarshal payload error. %v, %s", err, hex.EncodeToString(txid))
	}
	if payload.Type != 0 { //payload's type must be 0
		return "", "", nil, fmt.Errorf("payload type is not 0. %d : %s", payload.Type, hex.EncodeToString(txid))
	}

	//caculate the total amount of message from outputs
	totalAmount := uint64(0)
	for _, output := range payload.Essence.Outputs {
		if output.Type != 0 {
			continue
		}
		addr := output.Addr.Addr
		if output.Addr.Type == iotago.AddressEd25519 {
			addr = iotago.MustParseEd25519AddressFromHexString(output.Addr.Addr).Bech32(it.hrp)
		}
		if addr != it.Address() {
			continue
		}
		totalAmount += output.Amount
	}
	if totalAmount == 0 || len(payload.UnlockBlocks) == 0 {
		return "", "", nil, fmt.Errorf("message outputs amount is 0 or unlockBlocks is empty. %s : %d", hex.EncodeToString(txid), len(payload.UnlockBlocks))
	}

	pubKey, _ := hex.DecodeString(payload.UnlockBlocks[0].Sign.PublicKey)
	from := iotago.AddressFromEd25519PubKey(pubKey)
	bech32Addr := from.Bech32(it.hrp)

	payloadData := EssencePayloadData{}
	if err = json.Unmarshal([]byte(payload.Essence.EssPayload.Data), &payloadData); err != nil {
		return "", "", nil, fmt.Errorf("payload Unmarshal error. %s : %v", payload.Essence.EssPayload.Data, err)
	}

	if toCoin != payloadData.Symbol {
		return "", "", nil, fmt.Errorf("payload symbols not equal. %s : %s", payloadData.Symbol, toCoin)
	}

	return bech32Addr, payloadData.To, new(big.Int).SetUint64(totalAmount), nil
}

func (it *IotaToken) CheckTxFailed(failedTx, txid []byte, ed25519Addr string, amount *big.Int, d int) error {
	if d != -1 {
		return fmt.Errorf("iota network d error. %d", d)
	}

	var msgID iotago.MessageID
	copy(msgID[:], failedTx)
	meta, err := it.nodeAPI.MessageMetadataByMessageID(context.Background(), msgID)
	if err != nil {
		return fmt.Errorf("MessageMetadataByMessageID error. %s, %v", msgID, err)
	}
	if meta.ConflictReason == 0 {
		return fmt.Errorf("tx success. %s", msgID)
	}

	message, err := it.nodeAPI.MessageByMessageID(context.Background(), msgID)
	if err != nil {
		return fmt.Errorf("MessageByMessageID error. %s, %v", msgID, err)
	}

	//Unmarshal the payload of message
	data, err := message.Payload.MarshalJSON()
	if err != nil {
		return fmt.Errorf("MarshalJSON for(data) error. %v, %s", err, msgID)
	}
	payload := Payload{}
	err = json.Unmarshal(data, &payload)
	if err != nil {
		return fmt.Errorf("Unmarshal payload error. %v, %s", err, msgID)
	}
	if payload.Type != 0 { //payload's type must be 0
		return fmt.Errorf("payload type is not 0. %d : %s", payload.Type, msgID)
	}

	//to, err := iotago.ParseEd25519AddressFromHexString(ed25519Addr)
	//if err != nil {
	//	return nil, nil, fmt.Errorf("iota to address error. %v : %s", err, ed25519Addr)
	//}
	pubKey, err := hex.DecodeString(payload.UnlockBlocks[0].Sign.PublicKey)
	if err != nil || bytes.Compare(it.publicKey, pubKey) != 0 {
		return fmt.Errorf("publickeys are not equal. %s", payload.UnlockBlocks[0].Sign.PublicKey)
	}

	payloadData := make(map[string]string)
	if err = json.Unmarshal([]byte(payload.Essence.EssPayload.Data), &payloadData); err != nil {
		return fmt.Errorf("payload Unmarshal error. %s : %d", payload.Essence.EssPayload.Data, err)
	}

	tx, err := hex.DecodeString(payloadData["txid"])
	if err != nil || bytes.Compare(txid, tx) != 0 {
		return fmt.Errorf("txids are not equal. %s : %s", payloadData["txid"], hex.EncodeToString(txid))
	}

	if ed25519Addr != payloadData["to"] {
		return fmt.Errorf("to addresses are not equal. %s : %s", payloadData["to"], ed25519Addr)
	}

	a, b := new(big.Int).SetString(payloadData["amount"], 10)
	if !b || a.Cmp(amount) != 0 {
		return fmt.Errorf("amounts are not equal. %s : %s", payloadData["amount"], a.String())
	}

	return nil
}

func (it *IotaToken) CreateUnWrapTxData(ed25519Addr string, amount *big.Int, extra []byte) ([]byte, []byte, error) {
	sendAmount := amount.Uint64()

	if sendAmount < gl.MIN_IOTA_AMOUNT {
		return nil, nil, fmt.Errorf("Sending iota amount(%d) can't be small than %d", sendAmount, gl.MIN_IOTA_AMOUNT)
	}

	to, err := iotago.ParseEd25519AddressFromHexString(ed25519Addr)
	if err != nil {
		return nil, nil, fmt.Errorf("iota to address error. %v : %s", err, ed25519Addr)
	}
	output := iotago.SigLockedSingleOutput{
		Address: to,
		Amount:  sendAmount,
	}
	essencePayload := iotago.Indexation{
		Index: []byte("TpBridge"),
		Data:  extra,
	}

	b := NewUnsignTransactionBuilder()

	_, unspentOutputs, err := it.nodeAPI.OutputsByEd25519Address(context.Background(), &it.walletAddr, false)
	if err != nil {
		return nil, nil, fmt.Errorf("Get OutputsByEd25519Address error. %v", err)
	}

	sum := uint64(0)
	count := 0
	for utxoInput, output := range unspentOutputs {
		if a, err := output.Deposit(); err != nil {
			return nil, nil, fmt.Errorf("output.Deposit() error. %v", err)
		} else {
			b.AddInput(&iotago.ToBeSignedUTXOInput{Address: &it.walletAddr, Input: utxoInput})
			sum += a
			count++
			if count >= gl.MAX_INPUT_COUNT {
				break
			}
		}
	}
	if sum < sendAmount {
		return nil, nil, fmt.Errorf("balance amount is not enough. %d, %d", sum, sendAmount)
	}
	left := sum - sendAmount
	if left > 0 {
		b.AddOutput(&iotago.SigLockedSingleOutput{
			Address: &it.walletAddr,
			Amount:  left,
		})
	}

	unsignedData, err := b.AddOutput(&output).AddIndexationPayload(&essencePayload).GetTxEssenceData()
	if err != nil {
		return nil, nil, err
	}

	context, err := b.essence.MarshalJSON()

	return unsignedData, context, err
}

func (it *IotaToken) GetWrapTxByHash(txHash string) (tokens.BaseTransaction, error) {
	baseTx := tokens.BaseTransaction{}
	msgID, err := iotago.MessageIDFromHexString(txHash)
	if err != nil {
		return baseTx, fmt.Errorf("MessageIDFromHexString error. %s, %v", txHash, err)
	}
	meta, err := it.nodeAPI.MessageMetadataByMessageID(context.Background(), msgID)
	if err != nil {
		return baseTx, fmt.Errorf("MessageMetadataByMessageID error. %s, %v", msgID, err)
	}
	if meta.ConflictReason != 0 {
		return baseTx, fmt.Errorf("ConflictReason is not confirm. %s : %d", msgID, meta.ConflictReason)
	}

	message, err := it.nodeAPI.MessageByMessageID(context.Background(), msgID)
	if err != nil {
		return baseTx, fmt.Errorf("MessageByMessageID error. %s, %v", msgID, err)
	}

	//Unmarshal the payload of message
	data, err := message.Payload.MarshalJSON()
	if err != nil {
		return baseTx, fmt.Errorf("MarshalJSON for(data) error. %v, %s", err, msgID)
	}
	payload := Payload{}
	err = json.Unmarshal(data, &payload)
	if err != nil {
		return baseTx, fmt.Errorf("Unmarshal payload error. %v, %s", err, msgID)
	}
	if payload.Type != 0 { //payload's type must be 0
		return baseTx, fmt.Errorf("payload type is not 0. %d : %s", payload.Type, msgID)
	}

	//caculate the total amount of message from outputs
	totalAmount := uint64(0)
	for _, output := range payload.Essence.Outputs {
		if output.Type != 0 {
			continue
		}
		addr := output.Addr.Addr
		if output.Addr.Type == iotago.AddressEd25519 {
			addr = iotago.MustParseEd25519AddressFromHexString(output.Addr.Addr).Bech32(it.hrp)
		}
		if addr != it.Address() {
			continue
		}
		totalAmount += output.Amount
	}

	if totalAmount == 0 || len(payload.UnlockBlocks) == 0 {
		return baseTx, fmt.Errorf("message outputs amount is 0 or unlockBlocks is empty. %s : %d", msgID, len(payload.UnlockBlocks))
	}
	baseTx.Amount = new(big.Int).SetUint64(totalAmount)

	payloadData := EssencePayloadData{}
	if err = json.Unmarshal([]byte(payload.Essence.EssPayload.Data), &payloadData); err != nil {
		return baseTx, fmt.Errorf("payload Unmarshal error. %s : %d", payload.Essence.EssPayload.Data, err)
	}
	baseTx.To = payloadData.To
	return baseTx, nil
}

func (it *IotaToken) ValiditeUnWrapTxData(hash, txData []byte) (tokens.BaseTransaction, error) {
	baseTx := tokens.BaseTransaction{}

	seri := &iotago.TransactionEssence{}
	if err := seri.UnmarshalJSON(txData); err != nil {
		return baseTx, fmt.Errorf("msgContext can't be UnmarshalJSON to TransactionEssence. %v", err)
	}

	if sign, err := seri.SigningMessage(); err != nil {
		return baseTx, fmt.Errorf("seri.SigningMessage error. %v", err)
	} else if bytes.Compare(hash, sign) != 0 {
		return baseTx, fmt.Errorf("hash is not right. %s : %s", hex.EncodeToString(hash), hex.EncodeToString(sign))
	}

	payload := seri.Payload.(*iotago.Indexation)
	extra := &tokens.WrapExtra{}
	if err := json.Unmarshal(payload.Data, extra); err != nil {
		return baseTx, fmt.Errorf("payload json.Unmarshal error. %v", err)
	}

	for i := range seri.Outputs {
		output := seri.Outputs[i].(*iotago.SigLockedSingleOutput)
		outAddr := output.Address.(*iotago.Ed25519Address).Bech32(it.hrp)
		if outAddr == it.Address() {
			continue
		}
		baseTx.Amount = new(big.Int).SetUint64(output.Amount)
		baseTx.To = output.Address.(*iotago.Ed25519Address).String()
	}
	baseTx.Txid = common.FromHex(extra.TxID)
	return baseTx, nil
}

func (it *IotaToken) SendSignedTxData(hash string, txData []byte) ([]byte, error) {
	txEssence := &iotago.TransactionEssence{}
	if err := txEssence.UnmarshalJSON(txData); err != nil {
		return nil, fmt.Errorf("txEssence.UnmarshalJSON error. %s, %v", hex.EncodeToString(txData), hash)
	}

	sign, _ := hex.DecodeString(hash)
	signature := iotago.Ed25519Signature{}
	copy(signature.PublicKey[:], it.publicKey)
	copy(signature.Signature[:], sign)
	unlockBlocks := serializer.Serializables{}
	unlockBlocks = append(unlockBlocks, &iotago.SignatureUnlockBlock{Signature: &signature})
	for i := 1; i < len(txEssence.Inputs); i++ {
		unlockBlocks = append(unlockBlocks, &iotago.ReferenceUnlockBlock{Reference: uint16(0)})
	}

	sigTxPayload := &iotago.Transaction{Essence: txEssence, UnlockBlocks: unlockBlocks}

	info, err := it.nodeAPI.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Get node info error. %v", err)
	}
	msg, err := iotago.NewMessageBuilder().
		NetworkIDFromString(info.NetworkID).
		Payload(sigTxPayload).
		Tips(context.Background(), it.nodeAPI).
		ProofOfWork(context.Background(), info.MinPowScore).
		Build()
	if err != nil {
		return nil, fmt.Errorf("Build message error. %v", err)
	}

	msg, err = it.nodeAPI.SubmitMessage(context.Background(), msg)
	if err != nil {
		return nil, fmt.Errorf("Send message to node error. %v", err)
	}

	id, err := msg.ID()
	if err != nil {
		return nil, fmt.Errorf("Get message id error. %v", err)
	}

	return id[:], nil
}

func (it *IotaToken) SendUnWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error) {
	return nil, fmt.Errorf("Don't support this method")
}

// NewTransactionBuilder creates a new TransactionBuilder.
func NewUnsignTransactionBuilder() *UnsignTransactionBuilder {
	return &UnsignTransactionBuilder{
		essence: &iotago.TransactionEssence{
			Inputs:  serializer.Serializables{},
			Outputs: serializer.Serializables{},
			Payload: nil,
		},
	}
}

// TransactionBuilder is used to easily build up a Transaction.
type UnsignTransactionBuilder struct {
	essence *iotago.TransactionEssence
}

// AddInput adds the given input to the builder.
func (b *UnsignTransactionBuilder) AddInput(input *iotago.ToBeSignedUTXOInput) *UnsignTransactionBuilder {
	b.essence.Inputs = append(b.essence.Inputs, input.Input)
	return b
}

// AddOutput adds the given output to the builder.
func (b *UnsignTransactionBuilder) AddOutput(output iotago.Output) *UnsignTransactionBuilder {
	b.essence.Outputs = append(b.essence.Outputs, output)
	return b
}

// AddIndexationPayload adds the given Indexation as the inner payload.
func (b *UnsignTransactionBuilder) AddIndexationPayload(payload *iotago.Indexation) *UnsignTransactionBuilder {
	b.essence.Payload = payload
	return b
}

// GetTxEssenceData gets the tx essence data for signing
func (b *UnsignTransactionBuilder) GetTxEssenceData() ([]byte, error) {
	// sort inputs and outputs by their serialized byte order
	return b.essence.SigningMessage()
}

func (b *UnsignTransactionBuilder) GetEssence() *iotago.TransactionEssence {
	return b.essence
}
