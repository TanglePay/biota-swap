package server

import (
	"bwrap/config"
	"bwrap/gl"
	"bwrap/model"
	"bwrap/smpc"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var acceptedTxes map[string]bool

func Accept() {
	acceptedTxes = make(map[string]bool)
	go func() {
		ticker := time.NewTicker(config.Server.AcceptTime * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := model.UpdateAccountState(config.Smpc.Account); err != nil {
				gl.OutLogger.Error("UpdateAccountState error. %v", err)
			}

			//Get the sign data from smpc node
			infoDatas, err := smpc.GetCurNodeSignInfo()
			if err != nil {
				gl.OutLogger.Error("getCurNodeSignInfo error. %v", err)
				continue
			}
			for i := range infoDatas {
				d := infoDatas[i]
				if len(d.MsgContext) != 1 || len(d.MsgHash) != 1 {
					gl.OutLogger.Error("msgContext don't support multiple. %v", d.MsgContext)
					continue
				}
				msgContext := MsgContext{}
				if err := json.Unmarshal([]byte(d.MsgContext[0]), &msgContext); err != nil {
					//					gl.OutLogger.Error("json.Unmarshal to MsgContext error. %v", err)
					continue
				}
				if (time.Now().Unix() - msgContext.Timestamp) > config.Server.AcceptOverTime {
					continue
				}
				t1 := srcTokens[msgContext.SrcToken]
				t2 := destTokens[msgContext.DestToken]
				if t1 == nil || t2 == nil {
					gl.OutLogger.Error("token don't support. %s, %s", msgContext.SrcToken, msgContext.DestToken)
					continue
				}
				if msgContext.Method != UnwrapMethod {
					continue
				}
				hash := common.FromHex(d.MsgHash[0])
				if baseTx, err := t1.ValiditeUnWrapTxData(hash, msgContext.TxData); err != nil {
					gl.OutLogger.Error("validiteUnWrapTxData error. %s, %s, %v", d.MsgHash[0], string(msgContext.TxData), err)
					continue
				} else if err = t2.CheckUnWrapTx(baseTx.Txid, baseTx.To, t1.Symbol(), baseTx.Amount); err != nil {
					gl.OutLogger.Error("checkTxData error. %v : %v", baseTx, err)
					continue
				} else {
					if acceptedTxes[d.Key] {
						//gl.OutLogger.Error("txid has been unwrapped. txid: %s, to: %s, amount: %s", txid, baseTx.To, baseTx.Amount.String())
						continue
					}
					acceptedTxes[d.Key] = true
				}

				// Get Private Key
				_, prv, err := config.GetPrivateKey("smpc")
				if err != nil {
					gl.OutLogger.Error("getPrivateKey error. smpc, %v", err)
					return
				}
				if err = smpc.AcceptSign(d, true, prv); err != nil {
					gl.OutLogger.Error("smpc.AcceptSign error. %v : %v", d, err)
				} else {
					gl.OutLogger.Info("accept the info. %s", d.Key)
				}
			}
		}
	}()
}
