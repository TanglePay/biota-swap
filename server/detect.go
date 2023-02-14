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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/onrik/ethrpc"
)

func init() {
	client = ethrpc.New(config.Smpc.NodeUrl)
}

func ListenTokens() {
	for src, dest := range config.WrapPairs {
		srcTokens[src] = NewSourceChain(config.Tokens[src])
		destTokens[dest] = NewDestinationChain(config.Tokens[dest])

		log.Infof("src: %s", srcTokens[src].Address())

		go ListenWrap(srcTokens[src], destTokens[dest])
		go ListenUnWrap(srcTokens[src], destTokens[dest])
	}
}

func ListenWrap(t1 tokens.SourceToken, t2 tokens.DestinationToken) {
	for {
		orderC := make(chan *tokens.SwapOrder, 10)
		go t1.StartListen(orderC)
		gl.OutLogger.Info("Begin to listen source token %s.", t1.Symbol())
	FOR:
		for {
			select {
			case order := <-orderC:
				gl.OutLogger.Info("Wrap Order : %v", order)
				if order.Error != nil {
					gl.OutLogger.Error(order.Error.Error())
					if order.Type == 0 {
						break FOR
					}
				} else {
					dealWrapOrder(t1, t2, order)
				}
			}
		}
		time.Sleep(time.Second * 5)
		gl.OutLogger.Error("try to connect node again.")
	}
}

func ListenUnWrap(t1 tokens.SourceToken, t2 tokens.DestinationToken) {
	for {
		orderC := make(chan *tokens.SwapOrder, 10)
		go t2.StartListen(orderC)
		gl.OutLogger.Info("Begin to listen dest token %s.", t2.Symbol())
	FOR:
		for {
			select {
			case order := <-orderC:
				gl.OutLogger.Info("UnWrap Order : %v", order)
				if order.Error != nil {
					gl.OutLogger.Error(order.Error.Error())
					if order.Type == 0 {
						break FOR
					}
				} else {
					dealUnWrapOrder(t1, t2, order)
				}
			}
		}
		time.Sleep(time.Second * 5)
		gl.OutLogger.Error("try to connect node again.")
	}
}

func dealWrapOrder(t1 tokens.SourceToken, t2 tokens.DestinationToken, order *tokens.SwapOrder) {
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
	}

	// check the chain tx
	if err := model.StoreSwapOrder(&wo); err != nil {
		gl.OutLogger.Error("store the wrap order to db error(%v). %v", err, wo)
	}

	// mint the wrapped token in chain t2
	id, err := t2.SendWrap(order.TxID, order.Amount, order.To)
	if err != nil {
		gl.OutLogger.Error("SendWrap error. %s, %v", order.TxID, err)
	} else {
		gl.OutLogger.Info("SendWrap. %s => %s OK. %s", wo.SrcToken, wo.DestToken, hex.EncodeToString(id))
	}
	return
}

func dealUnWrapOrder(t1 tokens.SourceToken, t2 tokens.DestinationToken, order *tokens.SwapOrder) {
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
	}

	// Check the chain tx
	if err := model.StoreSwapOrder(&wo); err != nil {
		gl.OutLogger.Error("store the unwrap order to db error(%v). %v", err, wo)
		if t1.MultiSignType() == tokens.SmpcSign {
			return
		}
	}

	// When the MultiSignType is Contract, this process don't need the smpc to sign.
	if t1.MultiSignType() == tokens.EvmMultiSign {
		id, err := t1.SendUnWrap(order.TxID, order.Amount, order.To)
		if err != nil {
			gl.OutLogger.Error("SendUnWrap error. %s, %v", order.TxID, err)
		} else {
			gl.OutLogger.Info("SendUnWrap. %s => %s OK. %s", order.FromToken, order.ToToken, hex.EncodeToString(id))
		}
		return
	}

	data, _ := json.Marshal(map[string]string{
		"txid":   wo.TxID,
		"from":   wo.From,
		"to":     wo.To,
		"amount": wo.Amount,
	})
	hash, txData, err := t1.CreateUnWrapTxData(order.To, order.Amount, data)
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
	keyID, err := smpc.Sign(common.Bytes2Hex(t1.PublicKey()), config.Smpc.Gid, string(msContext), hexutil.Encode(hash), config.Smpc.ThresHold, t1.KeyType())
	if err != nil {
		gl.OutLogger.Error("smpc.Sign error(%v). %v", err, order)
		return
	} else {
		gl.OutLogger.Info("Require Sign to smpc for unwrap. %s", keyID)
	}

	go detectSignStatus(keyID, txData, t1)
}

func detectSignStatus(keyID string, txData []byte, t tokens.SourceToken) {
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
			} else {
				gl.OutLogger.Info("SendSignedTxData OK. %s", hex.EncodeToString(txID))
			}
			break
		}
		time.Sleep(config.Server.DetectTime * time.Second)
	}
}
