package server

import (
	"biota_swap/gl"
	"biota_swap/smpc"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func Accept() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			//Get the sign data from smpc node
			infoDatas, err := smpc.GetCurNodeSignInfo()
			if err != nil {
				gl.OutLogger.Error("GetCurNodeSignInfo error. %v", err)
				continue
			}
			for i := range infoDatas {
				d := infoDatas[i]
				//validite the msgContext
				for i, msg := range d.MsgContext {
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
						t2.ValiditeWrapTxData(common.Hex2Bytes(d.MsgHash[i]), msgContext.TxData)
					} else if msgContext.Method == "unwrap" {
						t1.ValiditeUnWrapTxData(common.Hex2Bytes(d.MsgHash[i]), msgContext.TxData)
					}
				}
			}
		}
	}()
}
