package server

import (
	"bwrap/config"
	"bwrap/gl"
	"bwrap/model"
	"encoding/hex"
	"time"
)

func RecheckIota() {
	ticker := time.NewTicker(time.Minute * 3)
	for range ticker.C {
		order, t := sentIotaTxes.pop()
		if order == nil {
			continue
		}
		hash, _ := hex.DecodeString(order.Hash)
		b, err := t.CheckSentTx(hash)
		if b {
			if err != nil {
				gl.OutLogger.Error("CheckSentTx error. %v", err)
				sentIotaTxes.push(order, t)
			} else {
				order.State = 1
				model.UpdateChainRecord(order)
			}
		} else {
			//sure error, delete from db
			order.State = 2
			if err := model.MoveOrderToError(order); err != nil {
				gl.OutLogger.Error("MoveOrderToError error. %v, %v", err, *order)
			}
		}
	}
}

func recheckEvmTx(q *SentEvmTxQueue) {
	ticker := time.NewTicker(time.Second * time.Duration(config.PendingTime))
	for range ticker.C {
		for {
			txOldHash, t, ts := q.Top()
			if ts == 0 || t == nil {
				break
			}
			if time.Now().Unix()-ts < config.PendingTime {
				break
			}

			// Get Private Key
			addr, prv, err := config.GetPrivateKey(t.Symbol())
			if err != nil {
				gl.OutLogger.Error("GetPrivateKey error. %s, %s, %v", addr.Hex(), t.Symbol(), err)
				break
			}
			txHash, err := t.CheckPendingAndSpeedUp(txOldHash, prv)
			if txHash.Hex() == txOldHash.Hex() {
				if err == nil {
					// tx is not pending ever, pop it and continue
					q.Pop()
					continue
				} else {
					// don't do anything, try again later
					gl.OutLogger.Error("Check pending tx error. %s, %s, %v", t.Symbol(), txOldHash.Hex(), err)
					break
				}
			} else {
				if err != nil {
					// maybe the tx was not pending state
					q.Pop()
					gl.OutLogger.Error("Send new tx error. %s, %s, %s, %v", t.Symbol(), txOldHash.Hex(), txHash.Hex(), err)
					continue
				} else {
					// have done the speed up
					q.UpdateTop(txHash, t, time.Now().Unix())
					gl.OutLogger.Info("Send speedUp tx. %s, %s, %s", t.Symbol(), txOldHash.Hex(), txHash.Hex())
					break
				}
			}
		}
	}
}
