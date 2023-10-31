package server

import (
	"bwrap/config"
	"bwrap/gl"
	"bwrap/log"
	"bwrap/model"
	"bwrap/smpc"
	"bwrap/tokens"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func ListenTokens() {
	sentEvmTxes = make(map[string]*SentEvmTxQueue)
	for src, dest := range config.WrapPairs {
		srcTokens[src] = NewSourceChain(config.Tokens[src])
		destTokens[dest] = NewDestinationChain(config.Tokens[dest])

		log.Infof("src %s : %s, dest %s : %s", src, srcTokens[src].Address(), dest, destTokens[dest].Address())

		if srcTokens[src].MultiSignType() == tokens.EvmMultiSign {
			key := srcTokens[src].Address() + srcTokens[src].ChainID()
			if _, exist := sentEvmTxes[key]; !exist {
				sentEvmTxes[key] = NewSentEvmTxQueue()
			}
		}
		if destTokens[dest].MultiSignType() == tokens.EvmMultiSign {
			key := destTokens[dest].Address() + destTokens[dest].ChainID()
			if _, exist := sentEvmTxes[key]; !exist {
				sentEvmTxes[key] = NewSentEvmTxQueue()
			}
		}

		go listenWrap(srcTokens[src], destTokens[dest])
		go listenUnWrap(srcTokens[src], destTokens[dest])
	}

	for key, q := range sentEvmTxes {
		log.Infof("Start to recheck evm tx pending : %s", key)
		go recheckEvmTx(q)
	}
}

func listenWrap(t1 tokens.SourceToken, t2 tokens.DestinationToken) {
	for {
		orderC := make(chan *tokens.SwapOrder, 10)
		go t1.StartWrapListen(orderC)
		gl.OutLogger.Info("Begin to listen source token %s : %s.", t1.Symbol(), t1.Address())
		for order := range orderC {
			if order.Error != nil {
				gl.OutLogger.Error(order.Error.Error())
				if order.Type == 0 {
					break
				}
			} else {
				gl.OutLogger.Info("Wrap Order : %v", *order)
				if order.Amount.Cmp(config.Tokens[t1.Symbol()].MinAmount) < 0 {
					gl.OutLogger.Error("The amount of %s is smaller than %s", t1.Symbol(), config.Tokens[t1.Symbol()].MinAmount.String())
					continue
				}
				dealWrapOrder(t2, order)
			}
		}
		time.Sleep(time.Second * 3)
		gl.OutLogger.Error("try to connect node again.")
	}
}

func listenUnWrap(t1 tokens.SourceToken, t2 tokens.DestinationToken) {
	for {
		orderC := make(chan *tokens.SwapOrder, 10000)
		go t2.StartUnWrapListen(orderC)
		gl.OutLogger.Info("Begin to listen dest token %s : %s.", t2.Symbol(), t2.Address())
		for order := range orderC {
			if order.Error != nil {
				gl.OutLogger.Error(order.Error.Error())
				if order.Type == 0 {
					break
				}
			} else {
				gl.OutLogger.Info("UnWrap Order : %v", *order)
				if order.Amount.Cmp(config.Tokens[t2.Symbol()].MinAmount) < 0 {
					gl.OutLogger.Error("The amount of %s is smaller than %s", t1.Symbol(), config.Tokens[t2.Symbol()].MinAmount.String())
					continue
				}
				dealUnWrapOrder(t1, order)
			}
		}
		time.Sleep(time.Second * 5)
		gl.OutLogger.Error("try to connect node again.")
	}
}

func dealWrapOrder(t2 tokens.DestinationToken, order *tokens.SwapOrder) {
	if order.ToToken != t2.Symbol() {
		gl.OutLogger.Error("The tx order's target token is error. %s, %s", order.ToToken, t2.Symbol())
		return
	}
	wo := model.SwapOrder{
		TxID:      order.TxID,
		SrcToken:  order.FromToken,
		DestToken: order.ToToken,
		Wrap:      1,
		From:      order.From,
		To:        order.To,
		Amount:    order.Amount.String(),
		Ts:        time.Now().UnixMilli(),
		Org:       order.Org,
	}

	if !dealedOrders.Check(order.TxID) {
		return
	}

	// check the chain tx
	if err := model.StoreSwapOrder(&wo); err != nil {
		if !strings.HasPrefix(err.Error(), "Error 1062") {
			gl.OutLogger.Error("store the wrap order to db error(%v). %v", err, wo)
		}
	}

	// Get Private Key
	_, prv, err := config.GetPrivateKey(t2.Symbol())
	if err != nil {
		gl.OutLogger.Error("GetPrivateKey error. %s, %v", t2.Symbol(), err)
		return
	}
	// mint the wrapped token in chain t2
	id, err := t2.SendWrap(order.TxID, order.Amount, order.To, prv)
	if err != nil {
		gl.OutLogger.Error("SendWrap error. %s, %v", order.TxID, err)
	} else {
		gl.OutLogger.Info("SendWrap. %s => %s OK. %s", wo.SrcToken, wo.DestToken, hex.EncodeToString(id))
		if t2.MultiSignType() == tokens.EvmMultiSign {
			if t, ok := t2.(tokens.EvmToken); ok {
				key := t2.Address() + t2.ChainID()
				sentEvmTxes[key].Push(common.BytesToHash(id), t, time.Now().Unix())
				sentEvmUserTxes.Push(common.FromHex(order.TxID))
			} else {
				gl.OutLogger.Error("Don't support RecheckEvmTxes. %s", t2.Symbol())
			}
		}
	}
}

func dealUnWrapOrder(t1 tokens.SourceToken, order *tokens.SwapOrder) {
	if order.ToToken != t1.Symbol() {
		gl.OutLogger.Error("The tx unwrap order's target token is error. %s, %s", order.ToToken, t1.Symbol())
		return
	}

	wo := model.SwapOrder{
		TxID:      order.TxID,
		SrcToken:  order.ToToken,
		DestToken: order.FromToken,
		Wrap:      -1,
		From:      order.From,
		To:        order.To,
		Amount:    order.Amount.String(),
		Ts:        time.Now().UnixMilli(),
		Org:       order.Org,
	}

	if !dealedOrders.Check(order.TxID) {
		return
	}

	// Check the chain tx
	if err := model.StoreSwapOrder(&wo); err != nil {
		if !strings.HasPrefix(err.Error(), "Error 1062") {
			gl.OutLogger.Error("store the unwrap order to db error(%v). %v", err, wo)
		}
		if t1.MultiSignType() != tokens.EvmMultiSign {
			return
		}
	}

	// When the MultiSignType is Contract, this process don't need the smpc to sign.
	if t1.MultiSignType() != tokens.SmpcSign {
		// Get Private Key
		_, prv, err := config.GetPrivateKey(t1.Symbol())
		if err != nil {
			gl.OutLogger.Error("GetPrivateKey error. %s, %v", t1.Symbol(), err)
			return
		}

		id, err := t1.SendUnWrap(order.TxID, order.Amount, order.To, prv)
		if err != nil {
			gl.OutLogger.Error("SendUnWrap error. %s, %v", order.TxID, err)
		} else {
			gl.OutLogger.Info("SendUnWrap. %s => %s OK. %s", order.FromToken, order.ToToken, hex.EncodeToString(id))
			if t1.MultiSignType() == tokens.EvmMultiSign {
				if t, ok := t1.(tokens.EvmToken); ok {
					key := t1.Address() + t1.ChainID()
					sentEvmTxes[key].Push(common.BytesToHash(id), t, time.Now().Unix())
					sentEvmUserTxes.Push(common.FromHex(order.TxID))
				} else {
					gl.OutLogger.Error("Don't support RecheckEvmTxes. %s", t1.Symbol())
				}
			} else {
				time.Sleep(60 * time.Second) // wait for 1 minute to make sure the smr outputs update.
			}
		}
		return
	}

	tag := common.FromHex(wo.TxID)
	hash, txData, err := t1.CreateUnWrapTxData(order.To, order.Amount, tag)
	if err != nil {
		gl.OutLogger.Error("CreateUnsignTxData error. %v : %v", err, order)
		return
	}
	msContext, _ := json.Marshal(MsgContext{
		SrcToken:  wo.SrcToken,
		DestToken: wo.DestToken,
		Method:    UnwrapMethod,
		TxData:    txData,
		Timestamp: time.Now().Unix(),
	})
	// Get Private Key
	_, prv, err := config.GetPrivateKey("smpc")
	if err != nil {
		gl.OutLogger.Error("GetPrivateKey error. smpc, %v", err)
		return
	}
	keyID, err := smpc.Sign(common.Bytes2Hex(t1.PublicKey()), config.Smpc.Gid, string(msContext), hexutil.Encode(hash), config.Smpc.ThresHold, t1.KeyType(), prv)
	if err != nil {
		gl.OutLogger.Error("smpc.Sign error(%v). %v", err, order)
		return
	} else {
		gl.OutLogger.Info("Require Sign to smpc for unwrap. %s : %s", keyID, hexutil.Encode(hash))
	}

	if txid := detectSignStatus(keyID, txData, t1); txid != nil {
		time.Sleep(60 * time.Second)
		wo.Hash = hex.EncodeToString(txid)
		sentIotaTxes.push(&wo, t1)
	}
}

func detectSignStatus(keyID string, txData []byte, t tokens.SourceToken) []byte {
	gl.OutLogger.Info("Waiting %s to accept... ", keyID)
	for i := 0; i < config.Server.DetectCount; i++ {
		rsvs, err := smpc.GetSignStatus(keyID)
		if err != nil {
			// get sign accept data fail from db
			if len(err.Error()) != 33 {
				gl.OutLogger.Error("GetSignStatus error. %s : %v", keyID, err)
			}
		}
		if len(rsvs) > 0 {
			if txID, err := t.SendSignedTxData(rsvs[0], txData); err != nil {
				gl.OutLogger.Error("SendSignedTxData error. %v, %v", keyID, err)
				return nil
			} else {
				gl.OutLogger.Info("SendSignedTxData OK. %s : %s", hex.EncodeToString(txID), rsvs[0])
				return txID
			}
		}
		time.Sleep(config.Server.DetectTime * time.Second)
	}
	gl.OutLogger.Warn("Waiting %s to accept over time.", keyID)
	return nil
}
