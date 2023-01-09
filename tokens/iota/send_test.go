package iota

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/iotaledger/hive.go/serializer"
	iotago "github.com/iotaledger/iota.go/v2"
	"github.com/iotaledger/iota.go/v2/ed25519"
)

//Data for the data of payload which is in the transaction's essence
type Data struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Amount     uint64 `json:"amount"`
	Collection int    `json:"collection"`
}

func TestSendSignedData(t *testing.T) {
	bech32To := "iota1qq5hapwllcsn9wync3aangry9nejung00s3gvfff7h6kexesd2uzwujfrqj"
	sendAmount := uint64(1000000)

	nodeAPI := iotago.NewNodeHTTPAPIClient("https://chrysalis-nodes.iota.org")
	info, err := nodeAPI.Info(context.Background())
	if err != nil {
		t.Fatalf("Get node info error. %v", err)
	}

	pk, err := hex.DecodeString(string("7b8b821264e031a3c0ffc1a8eea887521e1b3e3a081af4e777fa609789506fbd715593d2c4dfa9bc5b2718e6a4c704b63cd3b62a81ca92b17ee3487daf3d593a"))
	if pk == nil || err != nil {
		t.Fatalf("wallet iota pk error.")
	}
	private := ed25519.PrivateKey(pk)
	public := ed25519.PublicKey(pk[32:])
	ed25519Addr := iotago.AddressFromEd25519PubKey(public)
	addrKey := iotago.NewAddressKeysForEd25519Address(&ed25519Addr, private)
	signer := iotago.NewInMemoryAddressSigner(addrKey)

	prefix, to, err := iotago.ParseBech32(bech32To)

	output := iotago.SigLockedSingleOutput{
		Address: to,
		Amount:  sendAmount,
	}
	data, _ := json.Marshal(Data{
		From:       ed25519Addr.Bech32(prefix),
		To:         bech32To,
		Amount:     sendAmount,
		Collection: 0,
	})
	essencePayload := iotago.Indexation{
		Index: []byte("TpBridge"),
		Data:  data,
	}

	b := NewUnsignTransactionBuilder()

	_, unspentOutputs, err := nodeAPI.OutputsByEd25519Address(context.Background(), &ed25519Addr, false)
	if err != nil {
		t.Fatalf("Get OutputsByEd25519Address error. %v", err)
	}

	sum := uint64(0)
	for utxoInput, output := range unspentOutputs {
		if a, err := output.Deposit(); err != nil {
			t.Fatalf("Get Deposit error. %v", err)
		} else {
			b.AddInput(&iotago.ToBeSignedUTXOInput{Address: &ed25519Addr, Input: utxoInput})
			sum += a
			if (sum == sendAmount) || ((sum > sendAmount) && (sum-sendAmount > 1000000)) {
				break
			}
		}
	}
	if sum < sendAmount {
		t.Fatalf("balance amount is not enough. %d : %d", sum, sendAmount)
	}

	left := sum - sendAmount
	if left > 0 {
		b.AddOutput(&iotago.SigLockedSingleOutput{
			Address: &ed25519Addr,
			Amount:  left,
		})
	}

	txEssenceData, err := b.AddOutput(&output).AddIndexationPayload(&essencePayload).GetTxEssenceData()

	t.Log(hex.EncodeToString(txEssenceData))

	signature, err := signer.Sign(&ed25519Addr, txEssenceData)
	if err != nil {
		t.Fatalf("signature error. %v", err)
	}

	unlockBlocks := serializer.Serializables{}
	unlockBlocks = append(unlockBlocks, &iotago.SignatureUnlockBlock{Signature: signature})

	sigTxPayload := &iotago.Transaction{Essence: b.essence, UnlockBlocks: unlockBlocks}

	msg, err := iotago.NewMessageBuilder().
		NetworkIDFromString(info.NetworkID).
		Payload(sigTxPayload).
		Tips(context.Background(), nodeAPI).
		ProofOfWork(context.Background(), info.MinPowScore).
		Build()
	if err != nil {
		t.Fatalf("Build message error. %v", err)
	}

	msg, err = nodeAPI.SubmitMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("Send message to node error. %v", err)
	}

	id, err := msg.ID()
	if err != nil {
		t.Fatalf("Get message id error. %v", err)
	}

	t.Log(hex.EncodeToString(id[:]))
}
