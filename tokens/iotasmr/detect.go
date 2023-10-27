package iotasmr

import (
	"bwrap/tokens"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

// ShimmerToken
type IotaSmrToken struct {
	rpc        string
	nodeAPI    *nodeclient.Client
	protoParas *iotago.ProtocolParameters
	publicKey  []byte
	hrp        iotago.NetworkPrefix
	addr       iotago.Address
	addrBech32 string // bech32 address
	symbol     string
	tokenID    iotago.FoundryID
}

// NewShimmerToken
// url, node url, it contains the prefix of "https".
// _publicKey, the wallet's public key
// _symbol as string
// if tokenID is empty, it means smr or iota token; else it is L1 token
// hrp, smr or iota
func NewIotaSmrToken(url, _publicKey, _symbol, _tokenID, hrp string) *IotaSmrToken {
	var foundryID iotago.FoundryID
	nativeID := common.FromHex(_tokenID)
	if len(nativeID) == iotago.FoundryIDLength {
		copy(foundryID[:], nativeID)
	}
	pubKey := common.FromHex(_publicKey)
	if len(pubKey) != 32 {
		panic("NewIotaSmrToken, wrong public key : " + _publicKey)
	}
	edAddr := iotago.Ed25519AddressFromPubKey(pubKey)
	return &IotaSmrToken{
		rpc:        url,
		nodeAPI:    nodeclient.New(url),
		hrp:        iotago.NetworkPrefix(hrp),
		publicKey:  pubKey,
		addr:       &edAddr,
		addrBech32: edAddr.Bech32(iotago.NetworkPrefix(hrp)),
		symbol:     _symbol,
		tokenID:    foundryID,
	}
}

func (t *IotaSmrToken) StartWrapListen(ch chan *tokens.SwapOrder) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	nodeAPI := nodeclient.New(t.rpc)
	info, err := nodeAPI.Info(context.Background())
	if err != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("get Shimmer Node Info error. %s, %v", t.rpc, err)}
		return
	}
	t.protoParas = &info.Protocol
	eventAPI, err := nodeAPI.EventAPI(context.Background())
	if err != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("get shimmer event client error. %v", err)}
		return
	}
	if err := eventAPI.Connect(ctx); err != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("connect to iota node error. %v", err)}
		return
	}

	outputChan, sub := eventAPI.OutputsByUnlockConditionAndAddress(t.addr, t.hrp, nodeclient.UnlockConditionAddress)
	if sub == nil || sub.Error() != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("get sub from shimmer event client error. %v", sub.Error())}
		return
	}
	for {
		select {
		case recOutput := <-outputChan:
			if output, err := recOutput.Output(); err != nil {
				ch <- &tokens.SwapOrder{Type: 1, Error: fmt.Errorf("get output error. %s, %v", recOutput.Metadata.BlockID, err)}
			} else if err = t.dealNewOutput(output, recOutput.Metadata.BlockID, recOutput.Metadata.OutputIndex, ch); err != nil {
				ch <- &tokens.SwapOrder{Type: 1, Error: err}
			}
		case err := <-eventAPI.Errors:
			ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("shimmer node event connect error. %v", err)}
			return
		}
	}
}

func (t *IotaSmrToken) dealNewOutput(output iotago.Output, blockID string, index uint16, ch chan *tokens.SwapOrder) error {
	order, err := t.getWrapOrderByOutput(output, blockID)
	if order != nil {
		ch <- order
	}
	if index > 0 {
		return nil
	}
	return err
}

func (t *IotaSmrToken) getWrapOrderByOutput(output iotago.Output, blockID string) (*tokens.SwapOrder, error) {
	if output.Type() != iotago.OutputBasic {
		return nil, fmt.Errorf("output.Type error. %d", output.Type())
	}
	unlockConditions := output.UnlockConditionSet()
	if len(unlockConditions) != 1 || unlockConditions.Address() == nil {
		return nil, fmt.Errorf("output.UnlockCondition error. %v", output.UnlockConditionSet())
	}
	outputAddr := unlockConditions.Address().Address
	if !outputAddr.Equal(t.addr) {
		return nil, nil
	}

	userAmount := t.getAmountFromOutput(output)
	if userAmount.Sign() <= 0 {
		return nil, fmt.Errorf("tokens of output is empty. %s", blockID)
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
		return order, nil
	}
	return nil, fmt.Errorf("metadata of wrap order was error. %s, %v", blockID, err)
}

func (t *IotaSmrToken) getWrapOrderFromMetaData(features iotago.FeatureSet) (*WrapOrder, error) {
	if features.MetadataFeature() == nil {
		return nil, fmt.Errorf("meta data is null")
	}
	wrapOrder := &WrapOrder{}
	if err := json.Unmarshal(features.MetadataFeature().Data, wrapOrder); err != nil {
		return nil, fmt.Errorf("payload data Unmarshal error. %s : %v", hex.EncodeToString(features.MetadataFeature().Data), err)
	}
	wrapOrder.Tag = string(t.getTagDataFromFeature(features))
	return wrapOrder, nil
}

func (t *IotaSmrToken) getTagDataFromFeature(features iotago.FeatureSet) []byte {
	if features.TagFeature() != nil {
		return features.TagFeature().Tag
	}
	return nil
}

func (t *IotaSmrToken) getAmountFromOutput(output iotago.Output) *big.Int {
	var userAmount *big.Int = big.NewInt(0)
	coins := output.NativeTokenList()
	if len(coins) == 1 && coins[0].ID.Matches(t.tokenID) {
		userAmount = coins[0].Amount
	} else {
		userAmount = new(big.Int).SetUint64(output.Deposit())
	}
	return userAmount
}
