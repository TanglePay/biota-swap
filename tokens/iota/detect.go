package iota

import (
	"bwrap/tokens"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	iotago "github.com/iotaledger/iota.go/v2"
	iotagox "github.com/iotaledger/iota.go/v2/x"
)

type Payload struct {
	Type         int           `json:"type"`
	Essence      Essence_      `json:"essence"`
	UnlockBlocks []UnlockBlock `json:"unlockBlocks"`
}

type Essence_ struct {
	Outputs    []Output       `json:"outputs"`
	EssPayload EssencePayload `json:"payload"`
}

type UnlockBlock struct {
	Type int       `json:"type"`
	Sign Signature `json:"signature"`
}

type Signature struct {
	PublicKey string `json:"publicKey"`
}

type Address struct {
	Type iotago.AddressType `json:"type"`
	Addr string             `json:"address"`
}

type Output struct {
	Type   int     `json:"type"`
	Addr   Address `json:"address"`
	Amount uint64  `json:"amount"`
}

type EssencePayload struct {
	Type  int    `json:"type"`
	Index string `json:"index"`
	Data  string `json:"data"`
}

type EssencePayloadData struct {
	To     string `json:"to"`
	Symbol string `json:"symbol"`
}

func (it *IotaToken) StartListen(ch chan *tokens.SwapOrder) {
	//Get the contract addresses for listening the iota output event
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	eventUrl := "wss://" + it.url + "/mqtt"
	nodeEventClient := iotagox.NewNodeEventAPIClient(eventUrl)
	if err := nodeEventClient.Connect(ctx); err != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("Connect to iota node error. %s, %v", eventUrl, err)}
		return
	}
	addrMsg := nodeEventClient.AddressOutputs(&it.walletAddr, it.hrp)

	for {
		select {
		case msg := <-addrMsg:
			if err := it.dealTransferMessage(ch, msg); err != nil {
				ch <- &tokens.SwapOrder{Type: 1, Error: err}
			}
		case err := <-nodeEventClient.Errors:
			ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("Iota node event connect error. %v. Try again.", err)}
			return
		}
	}
}

func (it *IotaToken) dealTransferMessage(ch chan *tokens.SwapOrder, msg *iotago.NodeOutputResponse) error {
	//Get message by MessageID
	mId, _ := iotago.MessageIDFromHexString(msg.MessageID)
	mRes, err := it.nodeAPI.MessageByMessageID(context.Background(), mId)
	if err != nil || mRes == nil {
		return fmt.Errorf("Get message by id error. %v, %s", err, msg.MessageID)
	}

	//Unmarshal the payload of message
	data, err := mRes.Payload.MarshalJSON()
	if err != nil {
		return fmt.Errorf("MarshalJSON for(data) error. %v, %s", err, msg.MessageID)
	}
	payload := Payload{}
	err = json.Unmarshal(data, &payload)
	if err != nil {
		return fmt.Errorf("Unmarshal payload error. %v, %s", err, msg.MessageID)
	}
	if payload.Type != 0 { //payload's type must be 0
		return fmt.Errorf("payload type is not 0. %d : %s", payload.Type, msg.MessageID)
	}

	extraData := EssencePayloadData{}
	if err = json.Unmarshal(common.Hex2Bytes(payload.Essence.EssPayload.Data), &extraData); err != nil {
		return fmt.Errorf("payload data Unmarshal error. %s : %s : %v", msg.MessageID, payload.Essence.EssPayload.Data, err)
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
		return fmt.Errorf("message outputs amount is 0 or unlockBlocks is empty. %s : %d", msg.MessageID, len(payload.UnlockBlocks))
	}

	pubKey, _ := hex.DecodeString(payload.UnlockBlocks[0].Sign.PublicKey)
	from := iotago.AddressFromEd25519PubKey(pubKey)
	bech32Addr := from.Bech32(it.hrp)
	if bech32Addr == it.Address() { //transfer from the wallet to some one.
		return nil
	}

	order := &tokens.SwapOrder{
		TxID:      msg.MessageID,
		FromToken: it.Symbol(),
		ToToken:   extraData.Symbol,
		From:      bech32Addr,
		To:        extraData.To,
		Amount:    new(big.Int).SetUint64(totalAmount),
	}
	ch <- order
	return nil
}
