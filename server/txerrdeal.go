package server

import (
	"bwrap/config"
	"bwrap/gl"
	"bwrap/tokens"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

func ListenTxErrorRecord() {
	contract, err := tokens.NewTxErrorRecordContract(config.TxErrorRecord.NodeRpc, config.TxErrorRecord.NodeWss, config.TxErrorRecord.Contract, config.TxErrorRecord.ScanEventType, config.TxErrorRecord.TimePeriod)
	if err != nil {
		panic(err)
	}

	for {
		orderC := make(chan *tokens.TxErrorRecord)
		go contract.StartListen(orderC)
		gl.OutLogger.Info("Begin to listen TxErrorRecord : %s", config.TxErrorRecord.Contract)
	FOR:
		for {
			select {
			case order := <-orderC:
				if order.Error != nil {
					gl.OutLogger.Error("Listen TxErrorRecord. %v", order.Error)
					if order.Type == 0 {
						break FOR
					}
				} else {
					gl.OutLogger.Info("TxErrorRecord : %v", *order)
					dealTxErrorRecord(order)
				}
			}
		}
	}
}

func dealTxErrorRecord(o *tokens.TxErrorRecord) {
	var t1, t2 tokens.Token
	if o.D == -1 {
		t1 = destTokens[o.FromCoin]
		t2 = srcTokens[o.ToCoin]
	} else if o.D == 1 {
		t1 = srcTokens[o.FromCoin]
		t2 = destTokens[o.ToCoin]
	}

	//verify the txid
	from, to, amount, err := t1.CheckUserTx(o.Txid, o.ToCoin, o.D)
	if err != nil {
		gl.OutLogger.Error("txid check error in dealTxErrorRecord. %s : %v", hex.EncodeToString(o.Txid), err)
		return
	}

	for i := range o.FailedTxes {
		if err := t2.CheckTxFailed(o.FailedTxes[i], o.Txid, to, amount, o.D); err != nil {
			gl.OutLogger.Error("CheckTxFailed. %s : %v", hex.EncodeToString(o.FailedTxes[i]), err)
			return
		}
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

func DealWrapError(src, dest, txid, failedTxid string) ([]byte, error) {
	sentEvmTxes[src] = NewSentEvmTxQueue()
	sentEvmTxes[dest] = NewSentEvmTxQueue()

	t1 := NewSourceChain(config.Tokens[src])
	t2 := NewDestinationChain(config.Tokens[dest])

	hash := common.FromHex(txid)
	//verify the txid
	from, to, amount, err := t1.CheckUserTx(hash, dest, 1)
	if err != nil {
		return nil, fmt.Errorf("txid check error in dealTxErrorRecord. %s : %v", txid, err)
	}

	if len(failedTxid) > 0 {
		failedHash := common.FromHex(failedTxid)
		if err := t2.CheckTxFailed(failedHash, hash, to, amount, 1); err != nil {
			return nil, fmt.Errorf("CheckTxFailed. %s : %v", failedHash, err)
		}
	}

	order := &tokens.SwapOrder{
		TxID:      txid,
		FromToken: src,
		ToToken:   dest,
		From:      from,
		To:        to,
		Amount:    amount,
	}

	return dealWrapOrder(t2, order)
}
