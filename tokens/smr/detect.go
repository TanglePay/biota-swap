package smr

import (
	"bwrap/tokens"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

// ShimmerToken
type ShimmerToken struct {
	rpc        string
	nodeAPI    *nodeclient.Client
	publicKey  []byte
	hrp        iotago.NetworkPrefix
	address    iotago.Address
	walletAddr string // smr123456 address
	tokenID    iotago.FoundryID
	symbol     string
}

// NewShimmerToken
// url, node url, it contains the prefix of "https".
// cid, chainid
// addr, the wallet address like "smr1abc2334691..."
// _symbols, tokenid -> symbol
func NewShimmerToken(url, _publicKey, _symbol, _tokenID, hrp string) *ShimmerToken {
	nativeID, err := hex.DecodeString(_tokenID)
	if err != nil {
		log.Fatalf("tokenID error, %s : %v", _tokenID, err)
	}
	var foundryID iotago.FoundryID
	copy(foundryID[:], nativeID)
	pubKey := common.FromHex(_publicKey)
	if len(pubKey) != 32 {
		panic("NewShimmerToken, wrong public key : " + _publicKey)
	}
	edAddr := iotago.Ed25519AddressFromPubKey(pubKey)
	return &ShimmerToken{
		rpc:        url,
		nodeAPI:    nodeclient.New(url),
		hrp:        iotago.NetworkPrefix(hrp),
		publicKey:  pubKey,
		address:    &edAddr,
		walletAddr: edAddr.Bech32(iotago.NetworkPrefix(hrp)),
		symbol:     _symbol,
		tokenID:    foundryID,
	}
}

func (t *ShimmerToken) StartWrapListen(ch chan *tokens.SwapOrder) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	nodeAPI := nodeclient.New(t.rpc)
	info, err := nodeAPI.Info(context.Background())
	if err != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("get Shimmer Node Info error. %s, %v", t.rpc, err)}
		return
	}
	eventAPI, err := nodeAPI.EventAPI(context.Background())
	if err != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("get shimmer event client error. %v", err)}
		return
	}
	if err := eventAPI.Connect(ctx); err != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("connect to iota node error. %v", err)}
		return
	}

	outputChan, sub := eventAPI.OutputsByUnlockConditionAndAddress(t.address, t.hrp, nodeclient.UnlockConditionAddress)
	if sub == nil || sub.Error() != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("get sub from shimmer event client error. %v", sub.Error())}
		return
	}
	for {
		select {
		case recOutput := <-outputChan:
			if output, err := recOutput.Output(); err != nil {
				ch <- &tokens.SwapOrder{Type: 1, Error: fmt.Errorf("get output error. %s, %v", recOutput.Metadata.BlockID, err)}
			} else if err = t.dealNewOutput(&info.Protocol, output, recOutput.Metadata.BlockID, ch); err != nil {
				ch <- &tokens.SwapOrder{Type: 1, Error: err}
			}
		case err := <-eventAPI.Errors:
			ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("shimmer node event connect error. %v", err)}
			return
		}
	}
}

func (t *ShimmerToken) dealNewOutput(protoParas *iotago.ProtocolParameters, output iotago.Output, blockID string, ch chan *tokens.SwapOrder) error {
	if output.Type() != iotago.OutputBasic {
		return fmt.Errorf("output.Type error. %d", output.Type())
	}
	unlockConditions := output.UnlockConditionSet()
	if len(unlockConditions) != 1 || unlockConditions.Address() == nil {
		return fmt.Errorf("output.UnlockCondition error. %v", output.UnlockConditionSet())
	}
	outputAddr := unlockConditions.Address().Address
	if !outputAddr.Equal(t.address) {
		return nil
	}

	var userAmount *big.Int = big.NewInt(0)
	coins := output.NativeTokenList()
	if len(coins) == 1 && coins[0].ID.Matches(t.tokenID) {
		userAmount = coins[0].Amount
	} else {
		userAmount = new(big.Int).SetUint64(output.Deposit())
	}
	if userAmount.Sign() <= 0 {
		return fmt.Errorf("tokens of output is empty. %s", blockID)
	}

	// Get WrapOrder from MetaData first
	wrapOrder, err := t.getWrapOrderFromMetaData(output.FeatureSet())
	if err == nil && wrapOrder != nil {
		order := &tokens.SwapOrder{
			TxID:      blockID,
			FromToken: t.symbol,
			ToToken:   wrapOrder.Symbol,
			From:      wrapOrder.From,
			To:        wrapOrder.To,
			Amount:    userAmount,
		}
		ch <- order
		return nil
	}

	// Get from address and wrapOrder through block
	wrapOrder, err1 := t.getBlock(protoParas, blockID)
	if err1 != nil {
		return fmt.Errorf("unknow wrap order. %s : %v : %v", blockID, err, err1)
	}
	if wrapOrder == nil { // userAddr == walletAddr
		return nil
	}

	order := &tokens.SwapOrder{
		TxID:      blockID,
		FromToken: t.symbol,
		ToToken:   wrapOrder.Symbol,
		From:      wrapOrder.From,
		To:        wrapOrder.To,
		Amount:    userAmount,
		Org:       wrapOrder.Tag,
	}
	ch <- order
	return fmt.Errorf("use Payload Data to Wrap")
}

func (t *ShimmerToken) getBlock(protoParas *iotago.ProtocolParameters, id string) (*WrapOrder, error) {
	blockId, err := iotago.BlockIDFromHexString(id)
	if err != nil {
		return nil, fmt.Errorf("block id error. %v", err)
	}
	block, err := t.nodeAPI.BlockByBlockID(context.Background(), blockId, protoParas)
	if err != nil {
		return nil, fmt.Errorf("get block with id from node error. %v", err)
	}
	if block.Payload == nil || block.Payload.PayloadType() != iotago.PayloadTransaction {
		return nil, fmt.Errorf("payload type error. %s", id)
	}
	data, err := block.Payload.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalJSON the payload of block error. %v", err)
	}
	blockPayload := &BlockPayload{}
	if err = json.Unmarshal(data, blockPayload); err != nil {
		return nil, fmt.Errorf("marshalJSON the payload to struct error. %s, %v", string(data), err)
	}

	if len(blockPayload.Unlocks) < 1 {
		return nil, fmt.Errorf("block's publickey error. %s", id)
	}
	pubKey, err := hexutil.Decode(blockPayload.Unlocks[0].Signature.PublicKey)
	if err != nil || len(pubKey) != 32 {
		return nil, fmt.Errorf("block's publickey is error. %s, %v", blockPayload.Unlocks[0].Signature.PublicKey, err)
	}
	userAddr := iotago.Ed25519AddressFromPubKey(pubKey)
	if userAddr.Equal(t.address) {
		return nil, nil
	}

	if len(blockPayload.Essence.Payload.Data) > 0 {
		wrapOrder := &WrapOrder{}
		if err = json.Unmarshal(common.FromHex(blockPayload.Essence.Payload.Data), wrapOrder); err != nil {
			return nil, fmt.Errorf("payload data Unmarshal error. %s : %s", id, blockPayload.Essence.Payload.Data)
		}
		wrapOrder.From = userAddr.Bech32(t.hrp)
		wrapOrder.Tag = string(common.FromHex(blockPayload.Essence.Payload.Tag))
		return wrapOrder, nil
	}
	return nil, fmt.Errorf("payload data is null")
}

func (t *ShimmerToken) getWrapOrderFromMetaData(features iotago.FeatureSet) (*WrapOrder, error) {
	if features.MetadataFeature() == nil {
		return nil, fmt.Errorf("meta data is null")
	}
	wrapOrder := &WrapOrder{}
	if err := json.Unmarshal(features.MetadataFeature().Data, wrapOrder); err != nil {
		return nil, fmt.Errorf("payload data Unmarshal error. %s : %v", hex.EncodeToString(features.MetadataFeature().Data), err)
	}
	if features.TagFeature() != nil {
		wrapOrder.Tag = string(features.TagFeature().Tag)
	}
	return wrapOrder, nil
}
