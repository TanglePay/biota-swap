package smr

import (
	"bwrap/tokens"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"

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
	pubKey, err := hex.DecodeString(_publicKey)
	if err != nil {
		panic(err)
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
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("Get Shimmer Node Info error. %s, %v", t.rpc, err)}
		return
	}
	eventAPI, err := nodeAPI.EventAPI(context.Background())
	if err != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("Get shimmer event client error. %v", err)}
		return
	}
	if err := eventAPI.Connect(ctx); err != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("Connect to iota node error. %v", err)}
		return
	}

	outputChan, sub := eventAPI.OutputsByUnlockConditionAndAddress(t.address, t.hrp, nodeclient.UnlockConditionAddress)
	if sub == nil || sub.Error() != nil {
		ch <- &tokens.SwapOrder{Type: 0, Error: fmt.Errorf("Get sub from shimmer event client error. %v", sub.Error())}
		return
	}
	for {
		select {
		case recOutput := <-outputChan:
			if blockPayload, err := t.getBlock(nodeAPI, &info.Protocol, recOutput.Metadata.BlockID); err != nil {
				ch <- &tokens.SwapOrder{Type: 1, Error: fmt.Errorf("Get block error. %s, %v", recOutput.Metadata.BlockID, err)}
			} else if err := t.dealShimmerMsgWithTag(blockPayload, ch); err != nil {
				ch <- &tokens.SwapOrder{Type: 1, Error: err}
			}
		case err := <-eventAPI.Errors:
			ch <- &tokens.SwapOrder{Type: 1, Error: fmt.Errorf("Shimmer node event connect error. %v.", err)}
			return
		}
	}
}

func (t *ShimmerToken) dealShimmerMsgWithTag(block *BlockPayload, ch chan *tokens.SwapOrder) error {
	//caculate the total amount of message from outputs
	coins := make(map[string]*big.Int)
	for _, output := range block.Essence.Outputs {
		if output.Type != iotago.OutputBasic {
			continue
		}
		if len(output.UnlockConditions) != 1 {
			continue
		}
		if output.UnlockConditions[0].Type != iotago.UnlockConditionAddress {
			continue
		}
		addr, err := iotago.ParseEd25519AddressFromHexString(output.UnlockConditions[0].Address.PubKeyHash)
		if err != nil {
			return fmt.Errorf("The address is not a type 0. %s", output.UnlockConditions[0].Address.PubKeyHash)
		}
		if addr.Bech32(t.hrp) != t.walletAddr {
			continue
		}
		if len(output.NativeTokens) > 0 {
			for _, token := range output.NativeTokens {
				amount, b := new(big.Int).SetString(strings.TrimLeft(token.Amount, "0x"), 16)
				if !b {
					continue
				}
				if a, exist := coins[token.ID]; exist {
					coins[token.ID] = a.Add(a, amount)
				} else {
					coins[token.ID] = amount
				}
			}
		}
	}
	if len(block.Unlocks) == 0 {
		return fmt.Errorf("block.Unlocks is empty. %s", block.id)
	}

	if len(coins) == 0 {
		return fmt.Errorf("tokens of block is empty. %s : %d", block.id, len(coins))
	}
	pubKey, err := hexutil.Decode(block.Unlocks[0].Signature.PublicKey)
	if err != nil {
		return fmt.Errorf("block's publickey is error. %s, %v", block.Unlocks[0].Signature.PublicKey, err)
	}
	edAddr := iotago.Ed25519AddressFromPubKey(pubKey)
	from := edAddr.Bech32(t.hrp)
	if from == t.walletAddr { //transfer from the wallet to some one.
		return nil
	}

	amount, exist := coins[t.tokenID.ToHex()]
	if !exist {
		return fmt.Errorf("recieved token that was not supported. %s, %s, %v", block.id, t.tokenID, coins)
	}

	extraData := EssencePayloadData{}
	if err = json.Unmarshal(common.FromHex(block.Essence.Payload.Data), &extraData); err != nil {
		return fmt.Errorf("payload data Unmarshal error. %s : %s : %v", block.id, block.Essence.Payload.Data, err)
	}

	order := &tokens.SwapOrder{
		TxID:      block.id,
		FromToken: t.symbol,
		ToToken:   extraData.Symbol,
		From:      t.walletAddr,
		To:        extraData.To,
		Amount:    amount,
	}
	ch <- order
	return nil
}

func (t *ShimmerToken) getBlock(nodeAPI *nodeclient.Client, protoParas *iotago.ProtocolParameters, id string) (*BlockPayload, error) {
	blockId, err := iotago.BlockIDFromHexString(id)
	if err != nil {
		return nil, fmt.Errorf("block id error. %v", err)
	}
	block, err := nodeAPI.BlockByBlockID(context.Background(), blockId, protoParas)
	if err != nil {
		return nil, fmt.Errorf("get block with id from node error. %v", err)
	}
	data, err := block.Payload.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalJSON the payload of block error. %v", err)
	}
	blockPayload := &BlockPayload{}
	if err = json.Unmarshal(data, blockPayload); err != nil {
		return nil, fmt.Errorf("marshalJSON the payload to struct error. %s, %v", string(data), err)
	}
	blockPayload.id = id
	return blockPayload, nil
}
