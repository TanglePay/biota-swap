package server

import (
	"bwrap/config"
	"bwrap/gl"
	"bwrap/log"
	"bwrap/tokens"
	"encoding/hex"
	"time"
)

func ListenTxErrorRecord() {
	contract, err := tokens.NewTxErrorRecordContract(config.TxErrorRecord.NodeRpc, config.TxErrorRecord.NodeWss, config.TxErrorRecord.Contract, config.TxErrorRecord.ScanEventType, config.TxErrorRecord.TimePeriod)
	if err != nil {
		panic(err)
	}

	log.Infof("Start to listen TxErrorRecord : %s", config.TxErrorRecord.Contract)
	for {
		orderC := make(chan *tokens.TxErrorRecord)
		go contract.StartListen(orderC)
		gl.OutLogger.Info("Begin to listen TxErrorRecord : %s", config.TxErrorRecord.Contract)
		for order := range orderC {
			if order.Error != nil {
				gl.OutLogger.Error("Listen TxErrorRecord. %v", order.Error)
				if order.Type == 0 {
					break
				}
			} else {
				gl.OutLogger.Info("Deal TxErrorRecord : (%s:%s), %s", order.FromCoin, order.ToCoin, hex.EncodeToString(order.Txid))
				dealTxErrorRecord(order)
			}
		}
		time.Sleep(time.Second * 3)
		gl.OutLogger.Error("try to connect TxErrorRecord node again.")
	}
}

func dealTxErrorRecord(o *tokens.TxErrorRecord) {
	var t1 tokens.Token
	if o.D == -1 {
		t1 = destTokens[o.FromCoin]
	} else if o.D == 1 {
		t1 = srcTokens[o.FromCoin]
	}

	//verify the txid
	from, to, amount, err := t1.CheckUserTx(o.Txid, o.ToCoin, o.D)
	if err != nil {
		gl.OutLogger.Error("txid check error in dealTxErrorRecord. %s : %v", hex.EncodeToString(o.Txid), err)
		return
	}

	order := &tokens.SwapOrder{
		TxID:      hex.EncodeToString(o.Txid),
		FromToken: o.FromCoin,
		ToToken:   o.ToCoin,
		From:      from,
		To:        to,
		Amount:    amount,
	}

	if o.D == 1 {
		dealWrapOrder(destTokens[o.ToCoin], order)
	} else {
		dealUnWrapOrder(srcTokens[o.ToCoin], order)
	}
}
