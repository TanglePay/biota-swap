package server

import (
	"biota_swap/config"
	"biota_swap/gl"
	"biota_swap/model"
	"biota_swap/smpc"
	"biota_swap/tokens"
	"biota_swap/tokens/evm"
	"biota_swap/tokens/iota"
	"biota_swap/tokens/smrevm"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/onrik/ethrpc"
)

func init() {
	client = ethrpc.New(config.Sr.NodeUrl)
}

func ListenTokens() {
	srcTokens = make(map[string]tokens.SourceToken)
	destTokens = make(map[string]tokens.DestinationToken)
	for src, dest := range config.WrapPairs {
		if _, exist := srcTokens[src]; !exist {
			srcTokens[src] = NewSourceChain(config.Tokens[src])
		}
		if _, exist := destTokens[dest]; !exist {
			destTokens[dest] = NewDestinationChain(config.Tokens[dest])
		}
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
				fmt.Println(order)
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
				fmt.Println(order)
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
	wo := model.SwapOrder{
		TxID:      order.TxID,
		SrcToken:  t1.Symbol(),
		DestToken: t2.Symbol(),
		Wrap:      1,
		From:      order.From,
		To:        order.To,
		Amount:    order.Amount,
		Ts:        time.Now().UnixMilli(),
	}
	//check the chain tx
	if err := model.StoreSwapOrder(&wo); err != nil {
		gl.OutLogger.Error("store the wrap order to db error(%v). %v", err, wo)
		return
	}

	amount, _ := new(big.Int).SetString(wo.Amount, 10)
	hash, txData, err := t2.CreateWrapTxData(wo.To, amount, wo.TxID)
	if err != nil {
		gl.OutLogger.Error("CreateUnsignTxData error(%v). %v", err, order)
		return
	}

	msContext, _ := json.Marshal(MsgContext{SrcToken: wo.SrcToken, DestToken: wo.DestToken, Method: "wrap", TxData: txData})

	keyID, err := smpc.Sign(common.Bytes2Hex(t2.PublicKey()), config.Sr.Gid, string(msContext), hexutil.Encode(hash), config.Sr.ThresHold, t2.KeyType())
	fmt.Println(keyID)
	if err != nil {
		gl.OutLogger.Error("smpc.Sign error(%v). %v", err, order)
		return
	}

	go detectSignStatus(keyID, txData, t2)
}

func dealUnWrapOrder(t1 tokens.SourceToken, t2 tokens.DestinationToken, order *tokens.SwapOrder) {
	wo := model.SwapOrder{
		TxID:      order.TxID,
		SrcToken:  t1.Symbol(),
		DestToken: t2.Symbol(),
		Wrap:      -1,
		From:      order.From,
		To:        order.To,
		Amount:    order.Amount,
		Ts:        time.Now().UnixMilli(),
	}
	//check the chain tx
	if err := model.StoreSwapOrder(&wo); err != nil {
		gl.OutLogger.Error("store the wrap order to db error(%v). %v", err, wo)
		return
	}

	amount, _ := new(big.Int).SetString(wo.Amount, 10)
	data, _ := json.Marshal(map[string]interface{}{
		"txid":   wo.TxID,
		"from":   wo.From,
		"to":     wo.To,
		"amount": wo.Amount,
	})
	hash, txData, err := t1.CreateUnWrapTxData(wo.To, amount, data)
	if err != nil {
		gl.OutLogger.Error("CreateUnsignTxData error. %v : %v", err, order)
		return
	}

	msContext, _ := json.Marshal(MsgContext{SrcToken: wo.SrcToken, DestToken: wo.DestToken, Method: "wrap", TxData: txData})

	keyID, err := smpc.Sign(common.Bytes2Hex(t1.PublicKey()), config.Sr.Gid, string(msContext), hexutil.Encode(hash), config.Sr.ThresHold, t1.KeyType())
	fmt.Println(keyID)
	if err != nil {
		gl.OutLogger.Error("smpc.Sign error(%v). %v", err, order)
		return
	}

	go detectSignStatus(keyID, txData, t1)
}

func detectSignStatus(keyID string, txData []byte, t tokens.Token) {
	for i := 0; i < config.Sr.DetectCount; i++ {
		rsvs, err := smpc.GetSignStatus(keyID)
		if err != nil {
			gl.OutLogger.Error("GetSignStatus error. %s : %v", keyID, err)
		}
		if len(rsvs) > 0 {
			if txID, err := t.SendSignedTxData(rsvs[0], txData); err != nil {
				gl.OutLogger.Error("SendSignedTxData error. %v, %v", keyID, err)
			} else {
				gl.OutLogger.Info("SendSignedTxData OK. %s", hex.EncodeToString(txID))
			}
			break
		}
		time.Sleep(config.Sr.DetectTime * time.Second)
	}
}

func NewSourceChain(conf config.Token) tokens.SourceToken {
	switch conf.Symbol {
	case "IOTA":
		return iota.NewIotaToken(conf.NodeUrl, conf.Symbol, "iota", conf.PublicKey, "iota")
	case "ATOI":
		return iota.NewIotaToken(conf.NodeUrl, conf.Symbol, "iota", conf.PublicKey, "atoi")
	}
	return nil
}

func NewDestinationChain(conf config.Token) tokens.DestinationToken {
	var chain tokens.DestinationToken
	switch conf.Symbol {
	case "SMIOTA":
		chain, _ = smrevm.NewEvmSiota(conf.NodeUrl, conf.Contact, conf.PublicKey)
	case "MATIC":
		chain, _ = evm.NewEvmSiota(conf.NodeUrl, conf.Contact, conf.PublicKey)
	}
	return chain
}
