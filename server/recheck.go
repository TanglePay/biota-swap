package server

import (
	"bwrap/gl"
	"bwrap/model"
	"encoding/hex"
	"time"
)

func runRecheckIota() {
	ticker := time.NewTicker(time.Minute * 1)
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
