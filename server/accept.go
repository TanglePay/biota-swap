package server

import (
	"bwrap/config"
	"bwrap/gl"
	"bwrap/smpc"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var acceptedTxes map[string]bool

func Accept() {
	acceptedTxes = make(map[string]bool)
	go func() {
		ticker := time.NewTicker(config.Server.AcceptTime * time.Second)
		for range ticker.C {
			//Get the sign data from smpc node
			infoDatas, err := smpc.GetCurNodeSignInfo()
			if err != nil {
				gl.OutLogger.Error("GetCurNodeSignInfo error. %v", err)
				continue
			}
			for i := range infoDatas {
				d := infoDatas[i]
				var agree bool = true
				//validite the msgContext
				for j, msg := range d.MsgContext {
					msgContext := MsgContext{}
					if err := json.Unmarshal([]byte(msg), &msgContext); err != nil {
						gl.OutLogger.Error("json.Unmarshal to MsgContext error. %v", err)
						continue
					}
					t1 := srcTokens[msgContext.SrcToken]
					t2 := destTokens[msgContext.DestToken]
					if t1 == nil || t2 == nil {
						gl.OutLogger.Error("token client have no. %s, %s", msgContext.SrcToken, msgContext.DestToken)
					}
					if msgContext.Method == "wrap" {
						if baseTx, err := t2.ValiditeWrapTxData(common.Hex2Bytes(d.MsgHash[j]), msgContext.TxData); err != nil {
							agree = false
							gl.OutLogger.Error("ValiditeWrapTxData error. %s, %s, %v", d.MsgHash[j], string(msgContext.TxData), err)
						} else if err = t1.CheckTxData(baseTx.Txid, baseTx.To, baseTx.Amount); err != nil {
							agree = false
							gl.OutLogger.Error("CheckTxData error. %v : %v", baseTx, err)
						}
					} else if msgContext.Method == "unwrap" {
						if baseTx, err := t1.ValiditeUnWrapTxData(common.Hex2Bytes(d.MsgHash[j]), msgContext.TxData); err != nil {
							agree = false
							gl.OutLogger.Error("ValiditeWrapTxData error. %s, %s, %v", d.MsgHash[j], string(msgContext.TxData), err)
						} else if err = t2.CheckTxData(baseTx.Txid, baseTx.To, baseTx.Amount); err != nil {
							agree = false
							gl.OutLogger.Error("CheckTxData error. %v : %v", baseTx, err)
						} else {
							txid := hex.EncodeToString(baseTx.Txid)
							if acceptedTxes[txid] {
								agree = false
								gl.OutLogger.Error("txid has been unwrapped. txid: %s, to: %s, amount: %s", txid, baseTx.To, baseTx.Amount.String())
							}
							acceptedTxes[txid] = true
						}
					}
				}
				if err = smpc.AcceptSign(d, agree); err != nil {
					gl.OutLogger.Error("smpc.AcceptSign error. %v : %v", d, err)
				} else {
					gl.OutLogger.Info("Accept the info. ")
				}
			}
		}
	}()
}
